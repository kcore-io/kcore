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

package server

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sort"
	"strconv"
	"sync"
	"syscall"
	"testing"
)

const (
	TEST_ADDRESS = "127.0.0.1"
	TEST_PORT    = 8080
	MESSAGE_SIZE = 9
)

type MessageHandler func(message []byte, conn net.Conn)

type MockConnectionHandler struct {
	messageHandler MessageHandler
}

func (h *MockConnectionHandler) HandleConnection(conn net.Conn) {
	defer conn.Close()
	for {
		buf := make([]byte, MESSAGE_SIZE)
		n, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				slog.Debug("EOF reached, no more data to read from connection")
				return
			}
			slog.Error("Failed to read from connection", err)
			return
		}
		if n == 0 {
			return
		}
		slog.Debug("Received message", "message", string(buf))
		h.messageHandler(buf, conn)
	}
}

func TestMain(m *testing.M) {
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: false, Level: slog.LevelDebug})
	slog.SetDefault(slog.New(h))
	m.Run()
}

// TestStartSendStop tests that the server starts, receives messages in order and stops correctly
func TestStartReceiveStop(t *testing.T) {
	mc := make(chan []byte)
	messageHandler := func(message []byte, conn net.Conn) {
		mc <- message
	}

	handler := MockConnectionHandler{messageHandler: messageHandler}
	// Start the server
	s := NewTCPServer(
		TEST_ADDRESS, TEST_PORT, func() ConnectionHandler {
			return &handler
		},
	)
	err := s.Start()
	if err != nil {
		t.Fatalf("Failed to start TCP server: %s", err)
	}

	// Connect to the server
	conn, err := net.Dial("tcp", TEST_ADDRESS+":"+strconv.Itoa(TEST_PORT))

	if err != nil {
		t.Fatalf("Failed to connect to TCP server: %s", err)
	}

	dataToSend := make([][]byte, 4)

	for i := 0; i < 4; i++ {
		dataToSend[i] = []byte(fmt.Sprintf("Message %d", i))
	}

	for _, data := range dataToSend {
		n, err := conn.Write(data)
		if err != nil {
			t.Fatalf("Failed to write to TCP server: %s", err)
		}
		if n != len(data) {
			t.Fatalf("Failed to write all bytes to TCP server: wrote %d bytes, expected %d", n, len(data))
		}
		message := <-mc
		slog.Debug("Echoed message", "message", string(message))
		if string(message) != string(data) {
			t.Fatalf("Received message is not the same as the sent message: received %s, expected %s", message, data)
		}
	}
	if err = conn.Close(); err != nil {
		t.Fatalf("Failed to close connection: %s", err)
	}
	if err = s.Stop(); err != nil {
		t.Fatalf("Failed to stop server: %s", err)
	}
	// Check that the server stopped
	conn, err = net.Dial("tcp", TEST_ADDRESS+":"+strconv.Itoa(TEST_PORT))
	if err == nil {
		conn.Close()
		t.Fatalf("Server did not stop, was able to connect to it")
	}
	if !errors.Is(err, syscall.ECONNREFUSED) {
		// Connection refused error detected
		t.Fatalf("Got an unexpected error. Expected %s, got %s", syscall.ECONNREFUSED, err)
	}
}

// TestParallelClients tests that the server can handle multiple clients
func TestParallelClients(t *testing.T) {
	// Create a map to store the received data for each client
	receivedData := make(map[string][][]byte, 4)
	var mut sync.RWMutex

	// Create a message handler that stores the received data in the client's corresponding slice in the map
	messageHandler := func(message []byte, conn net.Conn) {
		mut.Lock()
		defer mut.Unlock()
		if _, ok := receivedData[conn.RemoteAddr().String()]; !ok {
			receivedData[conn.RemoteAddr().String()] = make([][]byte, 0)
		}
		receivedData[conn.RemoteAddr().String()] = append(receivedData[conn.RemoteAddr().String()], message)
	}

	// Start the server
	s := NewTCPServer(
		TEST_ADDRESS, TEST_PORT, func() ConnectionHandler {
			return &MockConnectionHandler{messageHandler: messageHandler}
		},
	)
	err := s.Start()
	if err != nil {
		t.Fatalf("Failed to start TCP server: %s", err)
	}

	// Create clients and send data
	clients := make([]net.Conn, 4)
	clientIDs := make([]string, 4)
	for i := 0; i < 4; i++ {
		conn, err := net.Dial("tcp", TEST_ADDRESS+":"+strconv.Itoa(TEST_PORT))
		if err != nil {
			t.Fatalf("Failed to connect to TCP server: %s", err)
		}
		clients[i] = conn
		clientIDs[i] = conn.LocalAddr().String()
		dataToSend := []byte(fmt.Sprintf("Message %d", i))
		n, err := conn.Write(dataToSend)
		if err != nil {
			t.Fatalf("Failed to write to TCP server: %s", err)
		}
		if n != len(dataToSend) {
			t.Fatalf("Failed to write all bytes to TCP server: wrote %d bytes, expected %d", n, len(dataToSend))
		}
	}
	// Wait for the data to be received
	for _, conn := range clients {
		conn.Close()
	}
	if err = s.Stop(); err != nil {
		t.Fatalf("Failed to stop server: %s", err)
	}
	// Check that the server stopped
	conn, err := net.Dial("tcp", TEST_ADDRESS+":"+strconv.Itoa(TEST_PORT))
	if err == nil {
		conn.Close()
		t.Fatalf("Server did not stop, was able to connect to it")
	}
	if !errors.Is(err, syscall.ECONNREFUSED) {
		// Connection refused error detected
		t.Fatalf("Got an unexpected error. Expected %s, got %s", syscall.ECONNREFUSED, err)
	}
	mut.RLock()
	defer mut.RUnlock()
	for addr, data := range receivedData {
		if len(data) != 1 {
			t.Fatalf("Expected to receive 1 message, received %d", len(data))
		}
		id := sort.SearchStrings(clientIDs, addr)
		want := fmt.Sprintf("Message %d", id)
		if string(data[0]) != want {
			t.Fatalf("Received message is not the same as the sent message: received %s, expected %s", data[0], want)
		}
	}
}
