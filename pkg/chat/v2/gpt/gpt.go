package gpt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"bigkinds.or.kr/pkg/chat/v2"
	"github.com/sirupsen/logrus"
	"github.com/valyala/fastjson"
)

const AuthorizationTypeApiKeyHeader = "ApiKeyHeader"

type GPT struct {
	// config
	Client *http.Client
	Option *chat.GptOptions
}

func NewGPTCompitable(client *http.Client, opts ...func(*chat.GptOptions)) (*GPT, error) {
	options := chat.NewGptOptions(opts...)

	if client == nil {
		return nil, errors.New("client is required")
	}

	if strings.Contains(options.Endpoint, "openai.com") && options.ApiKey == "" {
		return nil, errors.New("ApiKey is required for openai.com endpoint")
	}

	if len(options.Models) == 0 {
		return nil, errors.New("at least one model is required")
	}

	return &GPT{
		Client: client,
		Option: options,
	}, nil
}

func (c *GPT) AvailableModels() []string {
	return c.Option.Models
}

func (g *GPT) CreateRequest(ctx context.Context, messages []*chat.ChatPayload, options chat.GptPredictionOptions) (*http.Request, error) {
	var (
		functions         []*json.RawMessage
		rawFunctionCall   json.RawMessage
		tools             []*json.RawMessage
		rawToolChoice     json.RawMessage
		rawResponseFormat json.RawMessage
	)

	if len(messages) == 0 {
		return nil, errors.New("messages is required")
	}

	if options.Model == "" {
		return nil, errors.New("model is required")
	}

	gptMessages := []ChatCompletionMessage{}

	for _, c := range messages {
		msg := ChatCompletionMessage{
			Role:    c.Role,
			Content: c.Content,
		}
		if c.Name != nil {
			msg.Name = *c.Name
		}
		if c.FunctionCall != nil {
			msg.FunctionCall = &ChatCompletionFunctionCallResp{
				Name:      c.FunctionCall.Name,
				Arguments: c.FunctionCall.Arguments,
			}
		}
		if c.ToolCalls != nil {
			msg.ToolCalls = make([]*ChatCompletionToolsResp, len(c.ToolCalls))
			for i, toolCall := range c.ToolCalls {
				msg.ToolCalls[i] = &ChatCompletionToolsResp{
					Id:   toolCall.Id,
					Type: toolCall.Type,
				}
				if toolCall.Function != nil {
					msg.ToolCalls[i].Function = &ChatCompletionFunctionCallResp{
						Name:      toolCall.Function.Name,
						Arguments: toolCall.Function.Arguments,
					}
				}
			}
		}
		if c.ToolCallID != "" {
			msg.ToolCallID = c.ToolCallID
		}

		gptMessages = append(gptMessages, msg)

	}

	if options.Functions != nil {
		rawFunctions := make([]*json.RawMessage, len(options.Functions))
		for i, f := range options.Functions {
			rm := json.RawMessage(f)
			rawFunctions[i] = &rm
		}
		functions = rawFunctions
	}

	if options.FunctionCall != nil {
		// json.RawMessage should be wrapped with double quotes or braces
		if !strings.HasPrefix(*options.FunctionCall, "{") {
			*options.FunctionCall = fmt.Sprintf("\"%s\"", *options.FunctionCall)
		}
		rawFunctionCall = json.RawMessage(*options.FunctionCall)
	}

	// tool call
	if options.Tools != nil {
		rawTools := make([]*json.RawMessage, len(options.Tools))
		for i, t := range options.Tools {
			rm := json.RawMessage(t)
			rawTools[i] = &rm
		}
		tools = rawTools
	}

	if options.ToolChoice != nil {
		// json.RawMessage should be wrapped with double quotes or braces
		if !strings.HasPrefix(*options.ToolChoice, "{") {
			*options.ToolChoice = fmt.Sprintf("\"%s\"", *options.ToolChoice)
		}
		rawToolChoice = json.RawMessage(*options.ToolChoice)
	}

	// response format
	if options.ResponseFormat != nil {
		err := fastjson.Validate(*options.ResponseFormat)
		if err != nil {
			logrus.Errorf("invalid JSON: %s, error: %v", *options.ResponseFormat, err)
			return nil, err
		}
		rawResponseFormat = json.RawMessage(*options.ResponseFormat)
	}

	gptQuery := ChatCompletion{
		Model:            options.Model,
		Messages:         gptMessages,
		Functions:        functions,
		FunctionCall:     rawFunctionCall,
		Tools:            tools,
		ToolChoice:       rawToolChoice,
		Temperature:      options.Temperature,
		TopP:             options.TopP,
		MaxTokens:        options.MaxTokens,
		PresencePenalty:  options.PresencePenalty,
		FrequencyPenalty: options.FrequencyPenalty,
		ResponseFormat:   rawResponseFormat,
		Stream:           options.Stream,
		Seed:             options.Seed,
	}

	queryBytes, err := json.Marshal(gptQuery)
	if err != nil {
		logrus.Errorf("error marshalling gpt query: %s", err)
		return nil, err
	}

	// send request to openai
	targetUrl := g.Option.Endpoint

	req, err := http.NewRequestWithContext(ctx, "POST", targetUrl, bytes.NewReader(queryBytes))
	if err != nil {
		logrus.Errorf("error creating request: %s", err)
		return nil, err
	}
	if g.Option.ApiKey != "" {
		if g.Option.ApiType == chat.GPTAPITypeAzure {
			req.Header.Set("api-key", g.Option.ApiKey)
		} else {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.Option.ApiKey))
		}
	}
	req.Header.Set("Content-Type", "application/json")

	chat.AddForwardHeadersFromIncomingContext(&ctx, &req.Header)
	return req, nil
}

func (c *GPT) CreateRequestSolar(ctx context.Context, messages []*chat.ChatPayload, options chat.GptPredictionOptions) (*http.Request, error) {
	var (
		functions         []*json.RawMessage
		rawFunctionCall   json.RawMessage
		tools             []*json.RawMessage
		rawToolChoice     json.RawMessage
		rawResponseFormat json.RawMessage
	)

	if len(messages) == 0 {
		return nil, errors.New("messages is required")
	}

	if options.Model == "" {
		return nil, errors.New("model is required")
	}

	var solarMessages []ChatCompletionMessage

	for _, chatPayload := range messages {

		message := ChatCompletionMessage{
			Role:    chatPayload.Role,
			Content: chatPayload.Content,
		}
		if chatPayload.Name != nil {
			message.Name = *chatPayload.Name
		}
		if chatPayload.FunctionCall != nil {
			message.FunctionCall = &ChatCompletionFunctionCallResp{
				Name:      chatPayload.FunctionCall.Name,
				Arguments: chatPayload.FunctionCall.Arguments,
			}
		}
		if chatPayload.ToolCalls != nil {
			message.ToolCalls = make([]*ChatCompletionToolsResp, len(chatPayload.ToolCalls))
			for i, toolCall := range chatPayload.ToolCalls {
				message.ToolCalls[i] = &ChatCompletionToolsResp{
					Id:   toolCall.Id,
					Type: toolCall.Type,
				}
				if toolCall.Function != nil {
					message.ToolCalls[i].Function = &ChatCompletionFunctionCallResp{
						Name:      toolCall.Function.Name,
						Arguments: toolCall.Function.Arguments,
					}
				}
			}
		}
		if chatPayload.ToolCallID != "" {
			message.ToolCallID = chatPayload.ToolCallID
		}
		solarMessages = append(solarMessages, message)

	}
	// 수정
	//if options.Functions != nil {
	//	rawFunctions := make([]*json.RawMessage, len(options.Functions))
	//	for i, function := range options.Functions {
	//		rm := json.RawMessage(function)
	//		rawFunctions[i] = &rm
	//	}
	//	functions = rawFunctions
	//}
	if options.Functions != nil {
		tools = make([]*json.RawMessage, len(options.Functions))
		for i, function := range options.Functions {
			// 함수를 tool 형식으로 변환
			toolStr := fmt.Sprintf(`{"type":"function","function":%s}`, function)
			rm := json.RawMessage(toolStr)
			tools[i] = &rm
		}
	}

	if options.FunctionCall != nil {
		// json.RawMessage should be wrapped with double quotes or braces
		if !strings.HasPrefix(*options.FunctionCall, "{") {
			*options.FunctionCall = fmt.Sprintf("\"%s\"", *options.FunctionCall)
		}
		rawFunctionCall = json.RawMessage(*options.FunctionCall)
	}
	// tool call
	//if options.Tools != nil {
	//	rawTools := make([]*json.RawMessage, len(options.Tools))
	//	for i, tool := range options.Tools {
	//		rm := json.RawMessage(tool)
	//		rawTools[i] = &rm
	//	}
	//	tools = rawTools
	//}

	if options.ToolChoice != nil {
		// json.RawMessage should be wrapped with double quotes or braces
		if !strings.HasPrefix(*options.ToolChoice, "{") {
			*options.ToolChoice = fmt.Sprintf("\"%s\"", *options.ToolChoice)
		}
		rawToolChoice = json.RawMessage(*options.ToolChoice)
	}

	// response format
	if options.ResponseFormat != nil {
		err := fastjson.Validate(*options.ResponseFormat)
		if err != nil {
			logrus.Errorf("invalid JSON: %s, error: %v", *options.ResponseFormat, err)
			return nil, err
		}
		rawResponseFormat = json.RawMessage(*options.ResponseFormat)
	}

	solarQuery := ChatCompletion{
		Model:            options.Model,
		Messages:         solarMessages,
		Functions:        functions,
		FunctionCall:     rawFunctionCall,
		Tools:            tools,
		ToolChoice:       rawToolChoice,
		Temperature:      options.Temperature,
		TopP:             options.TopP,
		MaxTokens:        options.MaxTokens,
		PresencePenalty:  options.PresencePenalty,
		FrequencyPenalty: options.FrequencyPenalty,
		ResponseFormat:   rawResponseFormat,
		Stream:           options.Stream,
		Seed:             options.Seed,
	}

	queryBytes, err := json.Marshal(solarQuery)
	if err != nil {
		logrus.Errorf("error marshalling gpt query: %s", err)
		return nil, err
	}

	// send request to openai
	targetUrl := c.Option.Endpoint

	req, err := http.NewRequestWithContext(ctx, "POST", targetUrl, bytes.NewReader(queryBytes))
	if err != nil {
		logrus.Errorf("error creating request: %s", err)
		return nil, err
	}
	if c.Option.ApiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Option.ApiKey))
	}
	req.Header.Set("Content-Type", "application/json")
	chat.AddForwardHeadersFromIncomingContext(&ctx, &req.Header)
	return req, nil
}

func (c *GPT) Chat(ctx context.Context, messages []*chat.ChatPayload, opts ...func(o *chat.GptPredictionOptions)) (response *chat.ChatResponse, err error) {
	option := chat.NewGptPredictionOptions(opts...)

	req, err := c.CreateRequest(ctx, messages, *option)
	if err != nil {
		logrus.Errorf("error creating request: %s", err)
		return nil, err
	}

	resp, err := c.Client.Do(req)
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
		return nil, fmt.Errorf("response status code is not 200: (%d), %v", resp.StatusCode, body_string)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("error reading response body: %s", err)
		return nil, err
	}

	gptResponse := &ChatCompletionResponse{}

	err = json.Unmarshal(body, &gptResponse)
	if err != nil {
		logrus.Errorf("error unmarshalling gpt response: %s", err)
		return nil, err
	}

	if len(gptResponse.Choices) == 0 {
		logrus.Errorf("length of choices is 0!!resp: %v", gptResponse)
		return nil, err
	}

	// handle gpt response

	response = &chat.ChatResponse{
		Payload: &chat.ChatPayload{
			Role:    gptResponse.Choices[0].Message.Role,
			Content: gptResponse.Choices[0].Message.Content,
		},
		FinishReason: gptResponse.Choices[0].FinishReason,
	}

	if gptResponse.Choices[0].Message.FunctionCall != nil {
		argString := string(gptResponse.Choices[0].Message.FunctionCall.Arguments)
		logrus.Debugf("argString: %s", argString)

		compactBuffer := new(bytes.Buffer)
		err := json.Compact(compactBuffer, []byte(argString))
		if err != nil {
			logrus.Errorf("error compacting json: %s", err)
			return nil, err
		}

		fCall := compactBuffer.String()
		response.FunctionCall = &chat.ChatFunction{
			Name:      gptResponse.Choices[0].Message.FunctionCall.Name,
			Arguments: fCall,
		}
	}

	return response, nil
}
