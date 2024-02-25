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
	"log/slog"

	"github.com/k-streamer/sarama"
)

const API_VERSIONS_API_KEY = 18

type Response []byte

type MetadataManager interface {
	HandleRequest(req sarama.Request) (Response, error)
}

type metadataManager struct {
}

func NewMetadataManager() MetadataManager {
	return &metadataManager{}
}

func (mgr *metadataManager) HandleRequest(req sarama.Request) (Response, error) {
	//TODO: Implmenet
	slog.Debug("Handling request", "request", req)
	apiVersionResp := sarama.ApiVersionsResponse{
		ApiKeys: []sarama.ApiVersionsResponseKey{
			{
				ApiKey:     API_VERSIONS_API_KEY,
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
