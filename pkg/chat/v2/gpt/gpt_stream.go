package gpt

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"bigkinds.or.kr/pkg/chat/v2"
	"github.com/sirupsen/logrus"
)

type ChatStream interface {
	ReadUntilNow() *ChatCompletionResponse
	Recv(provider string) (*chat.ChatResponse, error)
	Close() error
}

func (c *GPT) CreateChatStream(ctx context.Context, provider string, messages []*chat.ChatPayload, opts ...func(o *chat.GptPredictionOptions)) (ChatStream, error) {

	options := chat.NewGptPredictionOptions(opts...)

	if !c.Option.Streamable {
		return nil, fmt.Errorf("model %s cannot stream", options.Model)
	}

	var (
		req *http.Request
		err error
	)

	switch provider {
	case "upstage":
		req, err = c.CreateRequestSolar(ctx, messages, *options)
	case "openai":
		req, err = c.CreateRequest(ctx, messages, *options)
	}

	if err != nil {
		slog.Info("CreateRequest error...")
		return nil, err
	}
	slog.Info("===== ===== ===== before c.Client.DO")
	resp, err := c.Client.Do(req)
	// slog.Info(fmt.Sprintf("%d\n", resp.StatusCode))
	if err != nil {
		slog.Info("===== ===== ===== error resp, err := c.Client.Do(req)")
		return nil, err
	}

	if resp.StatusCode >= 300 {
		slog.Info("===== ===== ===== resp.StatusCode >= 300")
		body, err := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			slog.Info("===== ===== ===== io.ReadAll(resp.Body) error")
			// slog.Info(string(body))
			return nil, err
		}
		return nil, fmt.Errorf("response status code is not in 200-299, status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return &GPTStream{
		reader: bufio.NewReader(resp.Body),
		body:   resp.Body,
	}, nil
}
func (c *GPT) CreateChat(ctx context.Context, provider string, messages []*chat.ChatPayload, opts ...func(o *chat.GptPredictionOptions)) (*http.Response, error) {
	options := chat.NewGptPredictionOptions(opts...)

	var (
		req *http.Request
		err error
	)

	switch provider {
	case "upstage":
		req, err = c.CreateRequestSolar(ctx, messages, *options)
	case "openai":
		req, err = c.CreateRequest(ctx, messages, *options)
	}

	if err != nil {
		return nil, err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("response status code is not in 200-299, status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

type ChatGPTStreamToken string

const (
	ChatGPTStreamDataStartToken  ChatGPTStreamToken = "data: "
	ChatGPTStreamDoneToken       ChatGPTStreamToken = "[DONE]"
	ChatGPTStreamErrorStartToken ChatGPTStreamToken = "{error"
)

type GPTStream struct {
	body   io.ReadCloser
	reader *bufio.Reader
	merged *ChatCompletionResponse
}

func (c *GPTStream) Close() error {
	return c.body.Close()
}

func (c *GPTStream) Recv(provider string) (*chat.ChatResponse, error) {
	shouldbeMerged := false // determine if all response should be merged although this is stream mode. ex) function call argument
	isError := false
	errorRawMessage := ""

	for {
		resp, err := c.reader.ReadBytes('\n')
		// slog.Info(string(resp))

		if provider != "upstage" {
			if errors.Is(err, io.EOF) {
				if isError {
					return nil, errors.New(errorRawMessage)
				}
				return nil, errors.New("v2 EOF comes before [DONE]")
			}
			if err != nil {
				return nil, err
			}
		}

		resp = bytes.TrimSpace(resp)

		if !bytes.HasPrefix(resp, []byte(ChatGPTStreamDataStartToken)) {
			continue
		}

		data := bytes.TrimPrefix(resp, []byte(ChatGPTStreamDataStartToken))

		if bytes.Equal(data, []byte(ChatGPTStreamErrorStartToken)) {
			isError = true
		}

		if isError {
			errorRawMessage += string(resp)
			continue
		}

		if bytes.Equal(data, []byte(ChatGPTStreamDoneToken)) {
			slog.Info("===== ===== ===== got [DONE]")
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

		var chunk ChatCompletionResponseChunk
		err = json.Unmarshal(data, &chunk)
		if err != nil {
			return nil, err
		}

		if len(chunk.Choices) < 1 {
			continue
		}

		choice := chunk.Choices[0]
		if choice.Delta != nil && (choice.Delta.FunctionCall != nil || choice.Delta.ToolCalls != nil) {
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
			resp, err := completionResponse.ChatResponse()
			if err == nil {
				logrus.Debugf("resp: %+v", *resp)
			}
			return resp, err
		}
	}
}

func (c *GPTStream) ReadUntilNow() *ChatCompletionResponse {
	return c.merged
}
