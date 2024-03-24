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
	"testing"
	"time"

	"github.com/kcore-io/sarama"
)

type MockConnection struct {
	out  [][]byte
	in   [][]byte
	reqs []sarama.ProtocolBody
}

func (m *MockConnection) Read(b []byte) (n int, err error) {
	if len(m.out) == 0 {
		return 0, io.EOF
	}
	copy(b, m.out[0])
	m.out = m.out[1:]
	return len(b), nil
}

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

func NewMockConnection() *MockConnection {
	return &MockConnection{}
}

func (m *MockConnection) WithRequest(request sarama.Request) *MockConnection {
	if m.reqs == nil {
		m.reqs = make([]sarama.ProtocolBody, 0)
	}
	m.reqs = append(m.reqs, request.Body)

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

func (m *MockConnection) ReadResponse() (*sarama.Response, error) {
	resp := &sarama.Response{
		Body:        &sarama.ApiVersionsResponse{},
		BodyVersion: m.reqs[0].APIVersion(),
	}
	m.reqs = m.reqs[1:]

	if len(m.in) == 0 {
		return nil, nil
	}

	buf := m.in[0]
	m.in = m.in[1:]

	err := sarama.VersionedDecode(buf, resp, 0, nil)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

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
	// buf, err := sarama.Encode(&request, nil)

	conn := NewMockConnection().WithRequest(request)

	handler := NewKafkaConnectionHandler(NewKafkaApi())

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
