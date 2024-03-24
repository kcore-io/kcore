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
	"errors"
	"fmt"
	"log/slog"

	"github.com/kcore-io/sarama"
)

type EncodedRequest []byte
type EncodedResponse []byte

// RequestHandler is an interface for handling Kafka requests.
// A single handler can handle multiple request types (i.e. API keys).
type RequestHandler interface {
	// Handle handles the Kafka request and returns the response.
	Handle(encodedReq EncodedRequest) (EncodedResponse, error)
}

type KafkaApi interface {
	HandleApiVersions(
		correlationId int32,
		clientId string,
		request sarama.ApiVersionsRequest,
	) (*sarama.ApiVersionsResponse, error)
}

type kafkaApi struct {
}

func NewKafkaApi() RequestHandler {
	return &kafkaApi{}
}

func (k *kafkaApi) Handle(encodedRequest EncodedRequest) (EncodedResponse, error) {
	// Parse the request
	req := sarama.Request{}
	err := req.Decode(&sarama.RealDecoder{Raw: encodedRequest})
	if err != nil {
		slog.Error("Failed to decode request", "error", err)
		return nil, fmt.Errorf("failed to decode request: %w", err)
	}
	slog.Debug(
		"Decoded request. Dispatching...", "client id", req.ClientID, "correlation id", req.CorrelationID,
		"api key",
		req.Body.APIKey(), "api version", req.Body.APIVersion(), "body", req.Body,
	)

	resp, err := k.dispatch(&req)
	if err != nil {
		slog.Error("Failed to dispatch request", "error", err)
		return nil, fmt.Errorf("failed to dispatch request: %w", err)
	}

	encodedResp, err := sarama.Encode(resp, nil)
	if err != nil {
		slog.Error("Failed to encode response", err)
		return nil, fmt.Errorf("failed to encode response: %w", err)
	}
	return encodedResp, nil
}

func (k *kafkaApi) dispatch(req *sarama.Request) (*sarama.Response, error) {
	var responseBody sarama.ProtocolBody
	var err error

	switch req.Body.APIKey() {
	case ApiVersionsApiKey:
		apiVersionsReq, ok := req.Body.(*sarama.ApiVersionsRequest)
		if !ok {
			return nil, errors.New("invalid request type")
		}
		slog.Debug("Dispatching request", "api key", req.Body.APIKey(), "ApiVersions request", apiVersionsReq)
		responseBody, err = k.HandleApiVersions(req.CorrelationID, req.ClientID, *apiVersionsReq)
		if err != nil {
			return nil, fmt.Errorf("error while handling ApiVersions request: %w", err)
		}
	default:
		return nil, errors.New("no handler found for request")
	}

	return &sarama.Response{
		CorrelationID: req.CorrelationID,
		Version:       ResponseHeaderVersion,
		Body:          responseBody,
	}, nil
}

func (k *kafkaApi) HandleApiVersions(
	correlationId int32,
	clientId string,
	request sarama.ApiVersionsRequest,
) (*sarama.ApiVersionsResponse, error) {

	// TODO: Make the ApiKeys dynamic
	return &sarama.ApiVersionsResponse{
		ApiKeys: []sarama.ApiVersionsResponseKey{
			{
				ApiKey:     ApiVersionsApiKey,
				MinVersion: ApiVersionsRequestVersion,
				MaxVersion: ApiVersionsRequestVersion,
			},
		},
		Version:   ApiVersionsRequestVersion,
		ErrorCode: 0,
	}, nil

}
