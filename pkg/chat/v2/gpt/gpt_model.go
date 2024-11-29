package gpt

import (
	"bytes"
	"encoding/json"
	"fmt"

	"bigkinds.or.kr/pkg/chat/v2"
	"github.com/sirupsen/logrus"
)

// request
type ChatCompletion struct {
	Model            string                  `json:"model"`
	Messages         []ChatCompletionMessage `json:"messages"`
	Functions        []*json.RawMessage      `json:"functions,omitempty"`
	FunctionCall     json.RawMessage         `json:"function_call,omitempty"`
	Tools            []*json.RawMessage      `json:"tools,omitempty"`
	ToolChoice       json.RawMessage         `json:"tool_choice,omitempty"`
	Temperature      *float32                `json:"temperature,omitempty"`
	TopP             *float32                `json:"top_p,omitempty"`
	MaxTokens        *int32                  `json:"max_tokens,omitempty"`
	PresencePenalty  *float32                `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float32                `json:"frequency_penalty,omitempty"`
	ResponseFormat   json.RawMessage         `json:"response_format,omitempty"`
	Stream           bool                    `json:"stream,omitempty"`
	Seed             *int64                  `json:"seed,omitempty"`
}

type ChatCompletionMessage struct {
	Role         string                          `json:"role"`
	Content      string                          `json:"content"`
	Name         string                          `json:"name,omitempty"`
	FunctionCall *ChatCompletionFunctionCallResp `json:"function_call,omitempty"`
	ToolCalls    []*ChatCompletionToolsResp      `json:"tool_calls,omitempty"`
	ToolCallID   string                          `json:"tool_call_id,omitempty"`
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
type ChatCompletionSummaryResp struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}
type ChatCompletionFunctionCallResp struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // ChatCompletionFunctionCallParams
}

type ChatCompletionToolsResp struct {
	Index    int                             `json:"index"`
	Id       string                          `json:"id"`
	Type     string                          `json:"type"`
	Function *ChatCompletionFunctionCallResp `json:"function"`
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

func (r *ChatCompletionResponse) ChatResponse() (*chat.ChatResponse, error) {
	chatResponse := &chat.ChatResponse{
		Payload: &chat.ChatPayload{
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

		chatResponse.Payload.Content = params.AssistantMessage
		fCall := compactBuffer.String()
		chatResponse.FunctionCall = &chat.ChatFunction{
			Name:      r.Choices[0].Message.FunctionCall.Name,
			Arguments: fCall,
		}
	} else if r.Choices[0].FinishReason == "tool_calls" {
		toolCalls := r.Choices[0].Message.ToolCalls
		chatResponse.ToolCalls = make([]*chat.ChatTool, len(toolCalls))
		for i, toolCall := range toolCalls {
			chatResponse.ToolCalls[i] = &chat.ChatTool{
				Id:   toolCall.Id,
				Type: toolCall.Type,
			}
			if toolCall.Function != nil {
				argString := toolCall.Function.Arguments
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

				chatResponse.Payload.Content = params.AssistantMessage
				fCall := compactBuffer.String()
				chatResponse.ToolCalls[i].Function = &chat.ChatFunction{
					Name:      toolCall.Function.Name,
					Arguments: fCall,
				}
			}
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
			FinishReason: choice.FinishReason,
		}
		if choice.Delta != nil {
			choices[i].Message = *choice.Delta
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
	if deltaChoice.Delta != nil {
		choice.Message.Role += deltaChoice.Delta.Role
		choice.Message.Content += deltaChoice.Delta.Content
		if deltaChoice.Delta.FunctionCall != nil {
			if choice.Message.FunctionCall == nil {
				choice.Message.FunctionCall = &ChatCompletionFunctionCallResp{}
			}

			choice.Message.FunctionCall.Name += deltaChoice.Delta.FunctionCall.Name
			choice.Message.FunctionCall.Arguments += deltaChoice.Delta.FunctionCall.Arguments
		} else if deltaChoice.Delta.ToolCalls != nil {
			for _, toolCall := range deltaChoice.Delta.ToolCalls {
				idx := toolCall.Index
				if idx >= len(choice.Message.ToolCalls) {
					choice.Message.ToolCalls = append(choice.Message.ToolCalls, &ChatCompletionToolsResp{})
				}

				choice.Message.ToolCalls[idx].Id += toolCall.Id
				choice.Message.ToolCalls[idx].Type += toolCall.Type
				if toolCall.Function != nil {
					if choice.Message.ToolCalls[idx].Function == nil {
						choice.Message.ToolCalls[idx].Function = &ChatCompletionFunctionCallResp{}
					}

					choice.Message.ToolCalls[idx].Function.Name += toolCall.Function.Name
					choice.Message.ToolCalls[idx].Function.Arguments += toolCall.Function.Arguments
				}
			}
		}
	}

	r.Choices[0] = choice

	return nil
}
