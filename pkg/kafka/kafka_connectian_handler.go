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
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"net"

	"kcore/pkg/server"
)

const ProcessingQueueSize = 2

type KafkaConnectionHandler interface {
	server.ConnectionHandler
	run()
}

type kafkaConnectionHandler struct {
	conn           net.Conn
	ctx            context.Context
	cancel         context.CancelFunc
	requestHandler RequestHandler
}

func NewKafkaConnectionHandler(handler RequestHandler) KafkaConnectionHandler {
	ctx, cancel := context.WithCancel(context.Background())
	mgr := &kafkaConnectionHandler{
		requestHandler: handler,
		ctx:            ctx,
		cancel:         cancel,
	}
	// TODO: return error
	return mgr
}

func (h *kafkaConnectionHandler) HandleConnection(conn net.Conn) {
	h.conn = conn
	h.run()
}

/**
 * Starts reading from the connection
 * and handling requests.
 */
func (h *kafkaConnectionHandler) run() {
	defer h.conn.Close()
	for {
		// Read the request size (4 bytes)
		buffer := make([]byte, 4)
		slog.Debug("Reading request message size")
		n, err := h.conn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				return
			}
			slog.Error("Failed to read request message size from connection", err)
			return
		}
		if n != 4 {
			slog.Error("Failed to read request message size from connection", "read bytes", n, "Expected", 4)
			return
		}
		reqSize := binary.BigEndian.Uint32(buffer)
		slog.Debug("Read request message size from connection", "bytes", n, "request message size", reqSize)

		// Read the request (reqSize bytes)
		buffer = make([]byte, reqSize)
		n, err = h.conn.Read(buffer)
		if err != nil {
			slog.Error("Failed to read request from connection", err)
			return
		}
		if n != int(reqSize) {
			slog.Error("Failed to read request from connection", "read bytes", n, "Expected", reqSize)
			return
		}
		slog.Debug("Read request from connection", "size", n)

		// Handle the request
		resp, err := h.requestHandler.Handle(buffer)
		if err != nil {
			slog.Error("Failed to handle request", err)
			return
		}

		if _, err = h.conn.Write(resp); err != nil {
			slog.Error("Failed to write response to connection", err)
			return
		}
	}
}
