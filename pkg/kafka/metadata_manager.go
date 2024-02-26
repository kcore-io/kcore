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
	"log/slog"

	"github.com/k-streamer/sarama"
)

const ApiVersionsApiKey = 18

type Response []byte

type RequestHandler interface {
	// Handle handles the Kafka request and returns the response.
	Handle(req sarama.Request) (Response, error)
	// ShouldHandle returns true if the request should be handled by this handler based on the API key.
	ShouldHandle(req sarama.Request) bool
}

type metadataManager struct {
}

func NewMetadataManager() RequestHandler {
	return &metadataManager{}
}

func (mgr *metadataManager) Handle(req sarama.Request) (Response, error) {
	slog.Debug("Handling request", "request", req)
	apiVersionResp := sarama.ApiVersionsResponse{
		ApiKeys: []sarama.ApiVersionsResponseKey{
			{
				ApiKey:     ApiVersionsApiKey,
				MinVersion: 0,
				MaxVersion: 3,
			},
		},
		Version:   3,
		ErrorCode: 0,
	}

	resp, err := sarama.Encode(&apiVersionResp, nil)
	return resp, err
}

func (mgr *metadataManager) ShouldHandle(req sarama.Request) bool {
	return req.Body.APIKey() == ApiVersionsApiKey
}
