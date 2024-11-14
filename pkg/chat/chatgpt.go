package chat

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"bigkinds.or.kr/proto"
	"github.com/sashabaranov/go-openai"
	"github.com/sirupsen/logrus"

	m "bigkinds.or.kr/pkg/chat/model"
	chat_v2 "bigkinds.or.kr/pkg/chat/v2"
)

type ChatGPT struct {
	// config
	openaiKey    string
	solarKey     string
	model        string
	OpenAIClient *openai.Client
}

type ChatGPTConfig struct {
	OpenaiKey string
	SolarKey  string
	Model     string
}

type ChatGPTOptions struct {
	Functions    *[]string `json:"functions"`
	FunctionCall *string   `json:"function_call"`
	Model        *string   `json:"model"`
	Stream       bool      `json:"stream"`
	Temperature  float32   `json:"Temperature"`
}

const (
	openaiChatGPTUrl = "https://api.openai.com/v1/chat/completions"
	upstageLLMUrl    = "https://upstage_llama.sung.devstage.ai/v1/chat/completions"
	VLLM30BUrl       = "http://172.16.201.19:17283/v1/chat/completions"
	VLLM70BUrl       = "http://172.16.201.19:27283/v1/chat/completions"
	SolarProxy       = "https://api.upstage.ai/v1/solar/chat/completions"
	Solarv005Url     = "http://172.16.201.11:7861/v1/chat/completions"
	SolarV010Url     = "http://172.16.200.14:8000/v1/chat/completions"
	Solar1Url        = "http://172.16.200.14:8000/v1/chat/completions"
)

func NewOpenAIChatGPTModel(config ChatGPTConfig) (*ChatGPT, error) {
	if len(config.OpenaiKey) == 0 {
		return nil, errors.New("openaiKey is required")
	}
	if !GPTModelValid(config.Model) {
		return nil, fmt.Errorf("invalid model: %s", config.Model)
	}

	return &ChatGPT{
		openaiKey: config.OpenaiKey,
		solarKey:  config.SolarKey,
		model:     config.Model,
	}, nil
}

func GPTModelValid(model string) bool {
	// TODO: use openai api to validate model
	validModels := []string{
		"gpt-4",
		"gpt-4-0314",
		"gpt-4-32k",
		"gpt-4-32k-0314",
		"gpt-4-0613",
		"gpt-3.5-turbo",
		"gpt-3.5-turbo-0301",
		"gpt-3.5-turbo-0613",
		"gpt-3.5-turbo-16k",
		"gpt-3.5-turbo-16k-0613",
		"gpt-3.5-turbo-1106",
		"upstage_llama_30b_orca_50k",
		"upstage_llama_30b_orca_100k",
		"upstage_llama_30b_orca_all",
		"/data/project/public/checkpoints/Upstage-30b",
		"/data/project/public/checkpoints/Upstage-70b",
		"upstage/solar-0-70b-16bit",
		"upstage/solar-1-7b-dev0-instruct",
		"upstage/solar-1-13b-dev0-instruct",
		"Ups13B-v005",
		"upstage-13b-v010",
		"upstage-SOLAR-1",
		"upstage/solar-1-mini-chat",
	}
	// fine-tuned models are valid
	if isGPT(model) {
		return true
	}

	// check if model in validModels
	for _, m := range validModels {
		if m == model {
			return true
		}
	}

	return false
}

func isGPT(model string) bool {
	return strings.HasPrefix(model, "gpt") || strings.HasPrefix(model, "ft:gpt")
}

func (c *ChatGPT) createAPIRequest(ctx context.Context, request *proto.ChatRequest) (*http.Request, error) {
	model := request.Model
	if model == nil {
		return nil, errors.New("model isn't set")
	}
	optionWrapper, ok := model.Options.(*proto.ChatModel_GptOptions)
	if !ok {
		return nil, errors.New("options is not for GPT")
	}
	options := optionWrapper.GptOptions
	messages := request.Messages

	gptMessages := []m.ChatCompletionMessage{}
	for _, c := range messages {
		gptMessages = append(gptMessages, m.ChatCompletionMessage{
			Role:    c.Role,
			Content: c.Content,
		})
	}

	gptQuery := m.ChatCompletion{
		Model:    c.model,
		Messages: gptMessages,
	}

	if options != nil {
		if options.Functions != nil {
			rawFunctions := make([]*json.RawMessage, len(options.Functions))
			for i, f := range options.Functions {
				rm := json.RawMessage(f)
				rawFunctions[i] = &rm
			}
			gptQuery.Functions = rawFunctions
		}
		if options.FunctionCall != nil {
			gptQuery.FunctionCall = *options.FunctionCall
		}
		if model.Name != "" {
			if !GPTModelValid(model.Name) {
				return nil, fmt.Errorf("invalid model: %s", model.Name)
			}
			gptQuery.Model = model.Name
		}
		gptQuery.Stream = options.Stream
		if options.Temperature != nil {
			gptQuery.Temperature = *options.Temperature
		}
		if options.MaxTokens != nil {
			gptQuery.MaxTokens = int(*options.MaxTokens)
		}
	}

	queryBytes, err := json.Marshal(gptQuery)
	if err != nil {
		logrus.Errorf("error marshalling gpt query: %s", err)
		return nil, err
	}

	// send request to openai
	var targetUrl string
	if isGPT(gptQuery.Model) {
		targetUrl = openaiChatGPTUrl
	} else if strings.HasPrefix(gptQuery.Model, "upstage/solar") {
		targetUrl = SolarProxy
	} else if strings.HasPrefix(gptQuery.Model, "upstage_llama") {
		targetUrl = upstageLLMUrl
	} else if gptQuery.Model == "/data/project/public/checkpoints/Upstage-30b" {
		targetUrl = VLLM30BUrl
	} else if gptQuery.Model == "/data/project/public/checkpoints/Upstage-70b" {
		targetUrl = VLLM70BUrl
	} else if gptQuery.Model == "Ups13B-v005" {
		targetUrl = Solarv005Url
	} else if gptQuery.Model == "upstage-13b-v010" {
		targetUrl = SolarV010Url
	} else if gptQuery.Model == "upstage-SOLAR-1" {
		targetUrl = Solar1Url
	} else {
		return nil, fmt.Errorf("can not find endpoint for model: %s", gptQuery.Model)
	}

	req, err := http.NewRequest("POST", targetUrl, bytes.NewReader(queryBytes))
	if err != nil {
		logrus.Errorf("error creating request: %s", err)
		return nil, err
	}
	if isGPT(gptQuery.Model) {
		req.Header.Set("Authorization", "Bearer "+c.openaiKey)
	} else if gptQuery.Model == "upstage-13b-v010" {
		req.Header.Set("Authorization", "Bearer upstage000")
	} else if gptQuery.Model == "upstage-SOLAR-1" {
		req.Header.Set("Authorization", "Bearer upstage000")
	} else if gptQuery.Model == "upstage/solar-1-mini-chat" {
		req.Header.Set("Authorization", "Bearer "+c.solarKey)
	}

	req.Header.Set("Content-Type", "application/json")

	chat_v2.AddForwardHeadersFromIncomingContext(&ctx, &req.Header)
	logrus.Debugf("request: %v, %v", req.Body, req.Header)
	return req, nil
}

func (c *ChatGPT) Chat(ctx context.Context, request *proto.ChatRequest) (*proto.ChatResponse, error) {
	// create ChatGPT Query
	req, err := c.createAPIRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logrus.Errorf("error sending request: %s", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			logrus.Errorf("error reading response body: %s", err)
			return nil, err
		}
		body_string := string(body)
		logrus.Errorf("response status code is not 200: (%d), %v", resp.StatusCode, body_string)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("error reading response body: %s", err)
		return nil, err
	}

	gptResponse := &m.ChatCompletionResponse{}
	err = json.Unmarshal(body, &gptResponse)
	if err != nil {
		logrus.Errorf("error unmarshalling gpt response: %s", err)
		return nil, err
	}

	if len(gptResponse.Choices) == 0 {
		logrus.Errorf("length of choices is 0!!resp: %v", gptResponse)
		return nil, err
	}

	if len(gptResponse.Choices) != 1 {
		logrus.Warningf("length of choices is not 1!!(%v)", len(gptResponse.Choices))
	}

	// handle gpt response
	return gptResponse.ChatResponse()
}

func (c *ChatGPT) CreateChatStream(ctx context.Context, request *proto.ChatRequest) (ChatStream, error) {
	req, err := c.createAPIRequest(ctx, request)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("response status code is not in 200-299, status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return &ChatGPTStream{
		reader: bufio.NewReader(resp.Body),
	}, nil
}

type ChatGPTStreamToken string

const (
	ChatGPTStreamDataStartToken  ChatGPTStreamToken = "data: "
	ChatGPTStreamDoneToken       ChatGPTStreamToken = "[DONE]"
	ChatGPTStreamErrorStartToken ChatGPTStreamToken = "{error"
)

type ChatGPTStream struct {
	reader *bufio.Reader
	merged *m.ChatCompletionResponse
}

func (c *ChatGPTStream) Recv() (*proto.ChatResponse, error) {
	shouldbeMerged := false // determine if all response should be merged although this is stream mode. ex) function call argument

	for {
		resp, err := c.reader.ReadBytes('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}

		resp = bytes.TrimSpace(resp)
		data := bytes.TrimPrefix(resp, []byte(ChatGPTStreamDataStartToken))

		if bytes.Equal(data, []byte(ChatGPTStreamErrorStartToken)) {
			return nil, errors.New(string(data))
		}

		if bytes.Equal(data, []byte(ChatGPTStreamDoneToken)) {
			if shouldbeMerged { // if shouldbeMerged is true, stream would be continued and this return value is a delta value from start to now
				chatResponse, err := c.merged.ChatResponse()
				if err != nil {
					return nil, err
				}
				return chatResponse, io.EOF
			} else {
				return nil, io.EOF
			}
		}

		if errors.Is(err, io.EOF) {
			return nil, errors.New("EOF comes before [DONE]")
		}

		if !bytes.HasPrefix(resp, []byte(ChatGPTStreamDataStartToken)) {
			continue
		}

		var chunk m.ChatCompletionResponseChunk
		err = json.Unmarshal(data, &chunk)
		if err != nil {
			return nil, err
		}

		if len(chunk.Choices) < 1 {
			return nil, fmt.Errorf("no choices")
		}

		choice := chunk.Choices[0]
		if choice.Delta == nil {
			return nil, fmt.Errorf("delta is nil")
		}
		if choice.Delta.FunctionCall != nil {
			shouldbeMerged = true
		}

		if c.merged != nil {
			err = c.merged.Merge(&chunk)
		} else {
			c.merged = chunk.ChatCompletionResponse()
		}
		if err != nil {
			return nil, err
		}

		if !shouldbeMerged {
			completionResponse := chunk.ChatCompletionResponse()
			return completionResponse.ChatResponse()
		}
	}
}

func (c *ChatGPTStream) ReadUntilNow() *m.ChatCompletionResponse {
	return c.merged
}
