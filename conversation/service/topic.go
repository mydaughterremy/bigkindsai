package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"bigkinds.or.kr/backend/model"
	"bigkinds.or.kr/conversation/internal/llmclient"
	"bigkinds.or.kr/conversation/service/function"
	"bigkinds.or.kr/pkg/chat/v2"
	"bigkinds.or.kr/pkg/chat/v2/gpt"
	"bigkinds.or.kr/pkg/utils"
)

type TopicService struct {
	client *http.Client
	search *function.SearchPlugin
}

type FindTopicResponse struct {
	Title   string `json:"title"`
	Content string `json:"content"`
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
	callResponse, err := parsingKeywordResponse(response)
	if err != nil {
		return nil, err
	}

	callResponse.Arguments, err = convertArgumentsToUTF8IfNot(callResponse.Arguments, topicMessage)
	if err != nil {
		return nil, err
	}
	arguments, err := function.ParseFunctionArguments(callResponse.Arguments)
	if err != nil {
		return nil, err
	}
	extraArgs := &function.ExtraArgs{
		RawQuery: topicMessage,
	}

	searchByte, err := topicService.search.Call(context, arguments, extraArgs)
	if err != nil {
		return nil, err
	}
	var items struct {
		Items []model.Reference `json:"items"`
	}
	if err = json.Unmarshal(searchByte, &items); err != nil {
		return nil, err
	}
	var contents string
	for _, item := range items.Items {
		contents += item.Attributes.Content
	}

	summaryMessage := make([]*chat.ChatPayload, 1)
	summaryMessage[0] = &chat.ChatPayload{
		Role:    "system",
		Content: getSummaryPrompt(contents),
	}
	summaryResponse, err := client.CreateChat(context, models[0], summaryMessage, chat.WithModel(models[1]))
	if err != nil {
		slog.Error("error : ", err)
		return nil, err
	}
	summaryCallResponse, err := parsingSummaryResponse(summaryResponse)
	if err != nil {
		return nil, err
	}
	findTopicResponse := &FindTopicResponse{
		Title:   summaryCallResponse.Title,
		Content: summaryCallResponse.Content,
	}
	return findTopicResponse, nil
}

func parsingKeywordResponse(response *http.Response) (*gpt.ChatCompletionFunctionCallResp, error) {
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
func parsingSummaryResponse(response *http.Response) (*gpt.ChatCompletionSummaryResp, error) {
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
			var chatCompletionSummaryResponse gpt.ChatCompletionSummaryResp
			if err := json.Unmarshal([]byte(content), &chatCompletionSummaryResponse); err != nil {
				return nil, err
			}
			return &chatCompletionSummaryResponse, nil
		}
	}
	
	return nil, errors.New("no finish")
}
