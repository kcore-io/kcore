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
	"log/slog"
	"net"
	"strconv"
)

// ConnectionHandler is an interface for handling a single connection and will run in its own goroutine.
type ConnectionHandler interface {
	// HandleConnection handles a single connection.
	//
	// Each client connection has one instance of ConnectionHandler and runs in its own goroutine. This method will
	// block until the connection is closed.
	HandleConnection(conn net.Conn)
}

type ConnectionHandlerFactory func() ConnectionHandler

// TCPServer is a simple TCP server that listens for incoming connections and handles them with a ConnectionHandler.
type TCPServer struct {
	address        string
	port           int
	handlerFactory ConnectionHandlerFactory
	l              net.Listener
}

// NewTCPServer creates a new TCP server. It does not start the server.
func NewTCPServer(address string, port int, handlerFactory ConnectionHandlerFactory) *TCPServer {
	return &TCPServer{
		address:        address,
		port:           port,
		handlerFactory: handlerFactory,
	}
}

// Start starts the TCP server in a new goroutine.
func (s *TCPServer) Start() error {
	slog.Debug("Starting TCP server", "address", s.address, "port", s.port)
	l, err := net.Listen("tcp", s.address+":"+strconv.Itoa(s.port))
	if err != nil {
		slog.Error("Failed to start TCP server: %s", err)
		return err
	}
	slog.Debug("TCP server listening")
	s.l = l
	go func() {
		for {
			// When the server is stopped, the listener is closed and Accept() returns
			conn, err := l.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					slog.Debug("Connection closed, can't accept new connections")
					return
				}
				slog.Error("Failed to accept TCP connection", err)
				return
			}
			slog.Debug("Accepted new TCP connection", "remote address", conn.RemoteAddr())
			// TODO: Limit the number of concurrent connections
			go s.handlerFactory().HandleConnection(conn)
		}
	}()
	return nil
}

// Stop stops the TCP server.
func (s *TCPServer) Stop() error {
	slog.Debug("Stopping TCP server", "address", s.address, "port", s.port)
	if s.l == nil {
		slog.Debug("TCP server not running")
		return nil
	}
	err := s.l.Close()
	if err != nil {
		slog.Error("Failed to stop TCP server", err)
		return err
	}
	s.l = nil
	return nil
}
