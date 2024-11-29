package service

import (
	"bigkinds.or.kr/backend/model"
	"bigkinds.or.kr/conversation/internal/llmclient"
	"bigkinds.or.kr/conversation/service/function"
	"bigkinds.or.kr/pkg/chat/v2"
	"bigkinds.or.kr/pkg/chat/v2/gpt"
	"bigkinds.or.kr/pkg/utils"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type TopicService struct {
	client *http.Client
	search *function.SearchPlugin
}

type FindTopicResponse struct {
}

func NewTopicService() *TopicService {
	mockTime, _ := time.Parse(time.RFC3339, "2023-05-11T12:00:00.000+09:00")
	topicService := &TopicService{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		search: &function.SearchPlugin{
			CurrentTime: utils.CurrentTime{
				Time: mockTime,
			},
		},
	}
	return topicService
}

func (topicService *TopicService) GetTopic(context context.Context, topicMessage string) (*FindTopicResponse, error) {

	models := chat.GetLLMOptions()
	client, err := llmclient.NewClient(
		topicService.client,
		models[0],
		models[1],
		0,
		chat.WithStreamDisabled,
	)
	if err != nil {
		return nil, err
	}
	messages := make([]*chat.ChatPayload, 1)
	messages[0] = &chat.ChatPayload{
		Role:    "system",
		Content: getKeywordPrompt(topicMessage),
	}
	response, err := client.CreateChat(context, models[0], messages, chat.WithModel(models[1]))
	if err != nil {
		slog.Error("error : ", err)
		return nil, err
	}
	callResponse, err := parsingResponse(response)
	if err != nil {
		return nil, err
	}
	arguments, err := function.ParseFunctionArguments(callResponse.Arguments)
	if err != nil {
		return nil, err
	}
	searchByte, err := topicService.search.Call(context, arguments, &function.ExtraArgs{})
	if err != nil {
		return nil, err
	}
	var items struct {
		Items []model.Reference `json:"items"`
	}
	err = json.Unmarshal(searchByte, &items)

	if err != nil {
		return nil, err
	}

	return nil, nil
}

func parsingResponse(response *http.Response) (*gpt.ChatCompletionFunctionCallResp, error) {
	bodyBytes, _ := io.ReadAll(response.Body)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(response.Body)
	var chatResponse gpt.ChatCompletionResponse

	if err := json.Unmarshal(bodyBytes, &chatResponse); err != nil {
		slog.Error("failed to parse solar response", "error", err)
	}

	if len(chatResponse.Choices) == 0 {
		return nil, errors.New("no choice")
	}

	if chatResponse.Choices[0].FinishReason == "stop" {
		if chatResponse.Choices[0].Message.Content != "" {
			content := chatResponse.Choices[0].Message.Content
			return &gpt.ChatCompletionFunctionCallResp{
				Name:      "search",
				Arguments: content,
			}, nil
		}
	}
	return nil, errors.New("no finish")
}
