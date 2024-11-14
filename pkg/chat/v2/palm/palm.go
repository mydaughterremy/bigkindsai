package palm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"bigkinds.or.kr/pkg/chat/v2"
	"github.com/sirupsen/logrus"
)

type Palm struct {
	// LLMOptionsInterface
	Client *http.Client
	Option *PalmOptions
}

func (p *Palm) validateModel(model string) bool {
	validModels := p.Option.Models

	for _, m := range validModels {
		if m == model {
			return true
		}
	}

	return false
}

func (p *Palm) AvailableModels() []string {
	return p.Option.Models
}

func NewPalmModel(client *http.Client, opts ...func(*PalmOptions)) (*Palm, error) {
	options := NewPalmOptions(opts...)

	if client == nil {
		return nil, errors.New("client is required")
	}

	if len(options.ProjectID) == 0 {
		return nil, errors.New("projectID is required")
	}

	if len(options.Token) == 0 {
		return nil, errors.New("token is required")
	}

	if len(options.Models) == 0 {
		return nil, errors.New("At least one model is required")
	}

	return &Palm{
		Client: client,
		Option: options,
	}, nil
}

func (p *Palm) Chat(ctx context.Context, messages []*chat.ChatPayload, opts ...func(*PalmPredictionOptions)) (response *chat.ChatResponse, err error) {
	const endpointTemplate = "https://%s/v1/projects/%s/locations/us-central1/publishers/google/models/%s:predict"
	var context string

	po := NewPalmPredictionOptions(opts...)

	if len(messages) == 0 {
		return nil, errors.New("messages is required")
	}

	if len(po.Model) == 0 {
		return nil, errors.New("model is required")
	}

	if !p.validateModel(po.Model) {
		return nil, errors.New("invalid model")
	}

	if po.Context != nil {
		context = *po.Context
	}

	endpoint := fmt.Sprintf(endpointTemplate, p.Option.Endpoint, p.Option.ProjectID, po.Model)
	logrus.Debugf("Endpoint: %s", endpoint)

	palmMessages := []VertexAIPalmMessagePayload{}
	for _, msg := range messages {
		palmMessages = append(palmMessages, VertexAIPalmMessagePayload{
			Author:  msg.Role,
			Content: msg.Content,
		})
	}

	palmInstance := VertexAIPalmInstace{
		Context:  context,
		Messages: palmMessages,
	}

	palmParameter := buildPlamParms(po)

	palmQuery := VertexAIPalm{
		Instances:  []VertexAIPalmInstace{palmInstance},
		Parameters: palmParameter,
	}

	queryBytes, err := json.Marshal(palmQuery)
	if err != nil {
		return nil, err
	}

	// print queryBytes
	logrus.Debugf("Query: %s", queryBytes)

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(queryBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.Option.Token)

	chat.AddForwardHeadersFromIncomingContext(&ctx, &req.Header)
	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		body_string := string(body)
		logrus.Errorf("Response status != 200: %s", body_string)
		return nil, errors.New("No 200 Response: " + body_string)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	palmResponse := &VertexAIPalmResponse{}

	err = json.Unmarshal(body, &palmResponse)
	if err != nil {
		return nil, err
	}

	if len(palmResponse.Predections) == 0 {
		return nil, errors.New("No Predections")
	}

	palmPrediction := palmResponse.Predections[0] // size of Predections must be 1
	if len(palmPrediction.Candidates) == 0 {
		return nil, errors.New("No Candidates")
	}

	messageFromPalm := chat.ChatPayload{
		Role:    palmPrediction.Candidates[0].Author,
		Content: palmPrediction.Candidates[0].Content,
	}

	return &chat.ChatResponse{
		Payload:      &messageFromPalm,
		FinishReason: "stop",
	}, nil
}

// helper functions

func buildPlamParms(option *PalmPredictionOptions) (parameter VertexAIPalmParameters) {
	if option.Temperature != nil {
		parameter.Temparature = option.Temperature
	}
	if option.MaxOutputTokens != nil {
		parameter.MaxOutputTokens = option.MaxOutputTokens
	}
	if option.TopP != nil {
		parameter.TopP = option.TopP
	}
	if option.TopK != nil {
		parameter.TopK = option.TopK
	}

	return parameter
}
