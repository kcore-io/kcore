/*
Copyright 2023 KStreamer Authors

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
	"net"
	"testing"
	"time"

	"github.com/k-streamer/sarama"
)

type MockConnection struct {
	out [][]byte
	in  [][]byte
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

func NewMockConanection() *MockConnection {
	return &MockConnection{}
}

func (m *MockConnection) WithRequest(request sarama.Request) *MockConnection {
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

func (m *MockConnection) ReadResponseHeader() (*sarama.ResponseHeaderStruct, error) {
	header := &sarama.ResponseHeaderStruct{}

	if len(m.in) == 0 {
		return nil, nil
	}

	buf := m.in[0]
	m.in = m.in[1:]

	err := sarama.VersionedDecode(buf, header, 0, nil)
	if err != nil {
		return nil, err
	}

	return header, nil
}

func (m *MockConnection) ReadResponse() []byte {
	if len(m.in) == 0 {
		return nil
	}
	buf := m.in[0]
	m.in = m.in[1:]
	return buf
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

	conn := NewMockConanection().WithRequest(request)

	mgr := NewMetadataManager()
	handler := NewKafkaConnectionHandler(mgr)

	handler.HandleConnection(conn)

	header, err := conn.ReadResponseHeader()

	if err != nil {
		t.Errorf("Failed to read response header: %v", err)
	}

	if header == nil {
		t.Errorf("Expected response header to be non-nil")

	}

	if header.CorrelationID != request.CorrelationID {
		t.Errorf("Expected correlation id to be %d, got %d", request.CorrelationID, header.CorrelationID)
	}

	response := sarama.ApiVersionsResponse{}

	respBuf := conn.ReadResponse()

	if respBuf == nil {
		t.Errorf("Expected response to be non-nil")
	}

	err = sarama.VersionedDecode(respBuf, &response, apiVersionRequest.APIVersion(), nil)

	if err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if response.ErrorCode != 0 {
		t.Errorf("Expected error code to be 0, got %d", response.ErrorCode)
	}

	if len(response.ApiKeys) == 0 {
		t.Errorf("Expected api keys to be non-empty")
	}
}
