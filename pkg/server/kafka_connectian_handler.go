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

package server

import (
	"context"
	"encoding/binary"
	"log/slog"
	"net"

	"github.com/k-streamer/sarama"
)

const ProcessingQueueSize = 2

type KafkaConnectionHandler interface {
	ConnectionHandler
	run()
}

type kafkaConnectionHandler struct {
	conn            net.Conn
	ctx             context.Context
	cancel          context.CancelFunc
	metadataManager MetadataManager
}

func NewKafkaConnectionHandler(metadataManager MetadataManager) KafkaConnectionHandler {
	ctx, cancel := context.WithCancel(context.Background())
	mgr := &kafkaConnectionHandler{
		metadataManager: metadataManager,
		ctx:             ctx,
		cancel:          cancel,
	}
	//TODO: return error
	return mgr
}

func (h *kafkaConnectionHandler) HandleConnection(conn net.Conn) {
	h.conn = conn
	h.run()
}

/**
 * Starts reading from the connection
 *
 * For each connection we need to do the following:
* 1. Read the request from the connection
* 2. Parse the request
* 3. Call the appropriate API manager.
* 4. Write the response to the connection.
* 5. Repeat.
*/
func (h *kafkaConnectionHandler) run() {
	defer h.conn.Close()
	for {
		// Read the request size (4 bytes)
		buf := make([]byte, 4)
		slog.Debug("Reading request message size")
		n, err := h.conn.Read(buf)
		if err != nil {
			slog.Error("Failed to read request message size from connection", err)
			return
		}
		if n != 4 {
			slog.Error("Failed to read request message size from connection", "read bytes", n, "Expected", 4)
			return
		}
		reqSize := binary.BigEndian.Uint32(buf)
		slog.Debug("Read request message size from connection", "bytes", n, "request message size", reqSize)

		// Read the request (reqSize bytes)
		buf = make([]byte, reqSize)
		n, err = h.conn.Read(buf)
		if err != nil {
			slog.Error("Failed to read request from connection", err)
			return
		}
		if n != int(reqSize) {
			slog.Error("Failed to read request from connection", "read bytes", n, "Expected", reqSize)
			return
		}
		slog.Debug("Read request from connection", "bytes", n, "request", string(buf[:n]))

		// Parse the request
		req := sarama.Request{}
		err = req.Decode(&sarama.RealDecoder{Raw: buf})
		if err != nil {
			slog.Error("Failed to parse request", err)
			return
		}
		slog.Debug("Parsed request", "client id", req.ClientID, "correlation id", req.CorrelationID, "api key", req.Body.APIKey(), "api version", req.Body.APIVersion(), "body", req.Body)
		slog.Debug("Calling API manager")
		//TODO: We need a generic API manager that can handle all requests.
		resp, err := h.metadataManager.HandleRequest(req)
		if err != nil {
			slog.Error("Failed to handle request", err)
			return
		}
		// TODO: Write the response to the connection
		slog.Debug("Sending response", "response", resp)
		respHeader := sarama.ResponseHeaderStruct{
			CorrelationID: req.CorrelationID,
			Length:        int32(len(resp) + 8),
		}

		headerBuf, err := sarama.Encode(&respHeader, nil)
		if err != nil {
			slog.Error("Failed to encode response header", err)
			return
		}

		h.conn.Write(headerBuf)
		h.conn.Write(resp)
	}
}
