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

package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"kstreamer/pkg/server"
)

var (
	verbose bool
	address string
	port    int
)

func init() {
	flag.BoolVar(&verbose, "verbose", true, "Enable verbose logging")
	flag.StringVar(&address, "address", "127.0.0.1", "Address to listen on")
	flag.IntVar(&port, "port", 9092, "Port to listen on")
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle termination signals gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("Received termination signal")
		cancel()
	}()

	l := slog.LevelInfo
	if verbose {
		l = slog.LevelDebug
		slog.Info("Verbose logging enabled")
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l})
	slog.SetDefault(slog.New(h))
	s := server.NewTCPServer(address, port, func() server.ConnectionHandler {
		return server.NewKafkaConnectionHandler(server.NewMetadataManager())
	})
	slog.Info("Starting kstreamer...")
	go s.Start()
	<-ctx.Done()
	slog.Info("Shutting down kstreamer...")
	s.Stop()
}
