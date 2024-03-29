/*
Copyright 2024 KCore Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kafka

import (
	"io"
	"log/slog"
	"net"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/kcore-io/sarama"
)

const (
	ClusterID    = "kcore-cluster"
	ControllerId = 0
)

func TestMain(m *testing.M) {
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: false, Level: slog.LevelInfo})
	slog.SetDefault(slog.New(h))
	m.Run()
}

func TestAPIVersionsRequest(t *testing.T) {
	apiVersionRequest := sarama.ApiVersionsRequest{
		Version:               3,
		ClientSoftwareName:    "sarama",
		ClientSoftwareVersion: "1.27.0",
	}
	request := sarama.Request{
		CorrelationID: 1,
		ClientID:      "sarama",
		Body:          &apiVersionRequest,
	}
	expectedResp := sarama.Response{
		Version:     ResponseHeaderVersion,
		Body:        &sarama.ApiVersionsResponse{},
		BodyVersion: apiVersionRequest.Version,
	}

	conn := NewMockConnection().WithRequest(request).ExpectResponse(
		expectedResp.Version, expectedResp.Body, expectedResp.BodyVersion,
	)

	handler := NewKafkaConnectionHandler(NewKafkaApi(ClusterID, ControllerId))

	handler.HandleConnection(conn)

	resp, err := conn.ReadResponse()

	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if resp == nil {
		t.Fatalf("Expected response to be non-nil")
	} else if resp.CorrelationID != request.CorrelationID {
		t.Fatalf("Expected correlation id to be %d, got %d", request.CorrelationID, resp.CorrelationID)
	}

	if resp.Body == nil {
		t.Fatalf("Expected response body to be non-nil")
	}

	apiVersionsResponse := resp.Body.(*sarama.ApiVersionsResponse)

	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if apiVersionsResponse.ErrorCode != 0 {
		t.Fatalf("Expected error code to be 0, got %d", apiVersionsResponse.ErrorCode)
	}

	if len(apiVersionsResponse.ApiKeys) == 0 {
		t.Fatalf("Expected api keys to be non-empty")
	}
}

func Test_kafkaApi_HandleApiVersions(t *testing.T) {
	type args struct {
		correlationId int32
		clientId      string
		request       sarama.ApiVersionsRequest
		response      *sarama.Response
	}
	tests := []struct {
		name    string
		args    args
		want    *sarama.ApiVersionsResponse
		wantErr bool
	}{
		{
			name: "Test API Versions",
			args: args{
				correlationId: 1,
				clientId:      "kcore-client",
				request: sarama.ApiVersionsRequest{
					Version:               3,
					ClientSoftwareName:    "kcore",
					ClientSoftwareVersion: "1.0.0",
				},
				response: &sarama.Response{
					Body:        &sarama.ApiVersionsResponse{},
					BodyVersion: 3,
				},
			},
			want: &sarama.ApiVersionsResponse{

			},
		},
	}
	for _, tt := range tests {
		t.Run(
			tt.name, func(t *testing.T) {
				k := &kafkaApi{}
				got, err := k.HandleApiVersions(tt.args.correlationId, tt.args.clientId, tt.args.request)
				if (err != nil) != tt.wantErr {
					t.Errorf("HandleApiVersions() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("HandleApiVersions() got = %v, want %v", got, tt.want)
				}
			},
		)
	}
}

// MockConnection is a mock kafka client connection for testing. It allows you to set Kafka requests in order and
// read the responses written to the connection in the same order.
type MockConnection struct {
	out  [][]byte
	in   [][]byte
	reqs []*ReqRespPair
}

type ReqRespPair struct {
	Request             *sarama.Request
	ResponseVresion     int16
	ResponseBodyType    reflect.Type
	ResponseBodyVersion int16
}

// NewMockConnection creates a new mock connection.
func NewMockConnection() *MockConnection {
	return &MockConnection{}
}

// Read is expected to be called by the Kafka connection handler to read the request from the client.
//
// For testing purposes, we will return the request set by the test case in the order they were set using WithRequest.
func (m *MockConnection) Read(b []byte) (n int, err error) {
	if len(m.out) == 0 {
		return 0, io.EOF
	}
	copy(b, m.out[0])
	m.out = m.out[1:]
	return len(b), nil
}

// Write is expected to be called by the Kafka connection handler to write the response to the client.
//
// For testing purposes, we will store the response in the connection so that it can be read by the test case by
// calling ReadResponse.
func (m *MockConnection) Write(b []byte) (n int, err error) {
	if m.in == nil {
		m.in = make([][]byte, 0)
	}
	m.in = append(m.in, b)
	return len(b), nil
}

func (m *MockConnection) Close() error {
	return nil
}

func (m *MockConnection) LocalAddr() net.Addr {
	return nil
}

func (m *MockConnection) RemoteAddr() net.Addr {
	return nil
}

func (m *MockConnection) SetDeadline(t time.Time) error {
	return nil
}

func (m *MockConnection) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *MockConnection) SetWriteDeadline(t time.Time) error {
	return nil
}

// WithRequest sets the request to be sent by the client. The request will be encoded and stored to be later read by
// the Kafka connection handler.
func (m *MockConnection) WithRequest(request sarama.Request) *MockConnection {
	if m.reqs == nil {
		m.reqs = make([]*ReqRespPair, 0)
	}
	m.reqs = append(m.reqs, &ReqRespPair{Request: &request})

	buf, err := sarama.Encode(&request, nil)

	if err != nil {
		return nil
	}
	if len(m.out) == 0 {
		m.out = make([][]byte, 0)
	}

	m.out = append(m.out, buf[:4])
	m.out = append(m.out, buf[4:])
	return m
}

// ExpectResponse sets the response expected for the request set by the test case using WithRequest. The response will be
// written to the connection by the Kafka connection handler.
//
// If no request has been set using WithRequest, this function will panic.
func (m *MockConnection) ExpectResponse(
	responseVersion int16,
	responseBody sarama.ProtocolBody,
	bodyVersion int16,
) *MockConnection {
	if len(m.reqs) == 0 {
		panic("no request to respond to, call WithRequest first")
	}

	m.reqs[0].ResponseVresion = responseVersion
	m.reqs[0].ResponseBodyType = reflect.TypeOf(responseBody).Elem()
	m.reqs[0].ResponseBodyVersion = bodyVersion

	return m
}

// ReadResponse reads the response written to the connection by the Kafka connection handler. Responses are read in the
// order they were set by the test case using WithRequest and
func (m *MockConnection) ReadResponse() (*sarama.Response, error) {
	resp := &sarama.Response{}
	orig := reflect.New(m.reqs[0].ResponseBodyType).Elem()
	orig.FieldByName("Version").SetInt(int64(m.reqs[0].ResponseVresion))
	resp.Body = (orig.Addr().Interface()).(sarama.ProtocolBody)
	resp.BodyVersion = m.reqs[0].ResponseBodyVersion

	m.reqs = m.reqs[1:]

	if len(m.in) == 0 {
		return nil, nil
	}

	buf := m.in[0]
	m.in = m.in[1:]

	err := sarama.VersionedDecode(buf, resp, ResponseHeaderVersion, nil)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
