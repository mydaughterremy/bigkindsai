package model

import (
	"bytes"
	"encoding/json"
	"fmt"

	"bigkinds.or.kr/proto"
	"github.com/sirupsen/logrus"
)

// request
type ChatCompletion struct {
	Model        string                  `json:"model"`
	Messages     []ChatCompletionMessage `json:"messages"`
	Functions    []*json.RawMessage      `json:"functions,omitempty"`
	FunctionCall string                  `json:"function_call,omitempty"`
	Stream       bool                    `json:"stream"`
	Temperature  float32                 `json:"temperature,omitempty"`
	MaxTokens    int                     `json:"max_tokens,omitempty"`
}

type ChatCompletionMessage struct {
	Role         string                          `json:"role"`
	Content      string                          `json:"content"`
	Name         string                          `json:"name,omitempty"`
	FunctionCall *ChatCompletionFunctionCallResp `json:"function_call,omitempty"`
}

type ChatCompletionFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// response
type ChatCompletionResponse struct {
	Id      string                         `json:"id"`
	Object  string                         `json:"object"`
	Created int64                          `json:"created"`
	Choices []ChatCompletionResponseChoice `json:"choices"`
	Usage   ChatCompletionResponseUsage    `json:"usage"`
}

type ChatCompletionResponseChoice struct {
	Index        int64                 `json:"index"`
	Message      ChatCompletionMessage `json:"message"`
	FinishReason string                `json:"finish_reason"`
}

type ChatCompletionResponseUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatCompletionFunctionCallResp struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // ChatCompletionFunctionCallParams
}

type ChatCompletionFunctionCallParams struct {
	AssistantMessage string `json:"assistant_message"`
}

type ChatCompletionResponseChunk struct {
	Id      string                               `json:"id"`
	Object  string                               `json:"object"`
	Created int64                                `json:"created"`
	Model   string                               `json:"model"`
	Choices []*ChatCompletionResponseChunkChoice `json:"choices"`
}

type ChatCompletionResponseChunkChoice struct {
	Index        int                    `json:"index"`
	Delta        *ChatCompletionMessage `json:"delta"`
	FinishReason string                 `json:"finish_reason"`
}

func (r *ChatCompletionResponse) ChatResponse() (*proto.ChatResponse, error) {
	chatResponse := &proto.ChatResponse{
		Messages: &proto.ChatPayload{
			Role:    r.Choices[0].Message.Role,
			Content: r.Choices[0].Message.Content,
		},
		FinishReason: r.Choices[0].FinishReason,
	}

	if r.Choices[0].FinishReason == "function_call" {
		argString := r.Choices[0].Message.FunctionCall.Arguments

		logrus.Infof(argString)

		/*
			argString, err := strconv.Unquote(argString)
			if err != nil {
				logrus.Errorf("error unquoting function call params: %s", err)
				return nil, err
			}
		*/
		compactBuffer := new(bytes.Buffer)
		err := json.Compact(compactBuffer, []byte(argString))
		if err != nil {
			logrus.Errorf("error compacting json: %s", err)
			return nil, err
		}

		var params ChatCompletionFunctionCallParams
		err = json.Unmarshal(compactBuffer.Bytes(), &params)
		if err != nil {
			logrus.Errorf("error unmarshalling function call params: %s", err)
			return nil, err
		}

		chatResponse.Messages.Content = params.AssistantMessage
		fCall := compactBuffer.String()
		chatResponse.FunctionCall = &proto.FunctionCall{
			Name:      r.Choices[0].Message.FunctionCall.Name,
			Arguments: fCall,
		}
	}

	return chatResponse, nil
}

func (c *ChatCompletionResponseChunk) ChatCompletionResponse() *ChatCompletionResponse {
	r := &ChatCompletionResponse{
		Id:      c.Id,
		Object:  c.Object,
		Created: c.Created,
	}

	choices := make([]ChatCompletionResponseChoice, len(c.Choices))
	for i, choice := range c.Choices {
		choices[i] = ChatCompletionResponseChoice{
			Index:        int64(choice.Index),
			Message:      *choice.Delta,
			FinishReason: choice.FinishReason,
		}
	}
	r.Choices = choices

	return r
}

func (r *ChatCompletionResponse) Merge(delta *ChatCompletionResponseChunk) error {
	if r.Id != delta.Id {
		return fmt.Errorf("responses must have same id: %v, %v", r, delta)
	}

	// merge only first choice
	choice := r.Choices[0]
	deltaChoice := delta.Choices[0]

	choice.FinishReason += deltaChoice.FinishReason
	choice.Message.Role += deltaChoice.Delta.Role
	choice.Message.Content += deltaChoice.Delta.Content
	if deltaChoice.Delta.FunctionCall != nil {
		if choice.Message.FunctionCall == nil {
			choice.Message.FunctionCall = &ChatCompletionFunctionCallResp{}
		}

		choice.Message.FunctionCall.Name += deltaChoice.Delta.FunctionCall.Name
		choice.Message.FunctionCall.Arguments += deltaChoice.Delta.FunctionCall.Arguments
	}

	r.Choices[0] = choice

	return nil
}
