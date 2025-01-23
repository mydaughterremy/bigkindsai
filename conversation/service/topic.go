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
	Title   string   `json:"topic_title"`
	Content string   `json:"topic_content"`
	Count   int      `json:"news_count"`
	Ids     []string `json:"news_ids"`
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

// 오늘의 이슈 요약, 뉴스 정보 조회
func (topicService *TopicService) GetTopic(context context.Context, topicMessage string) (*FindTopicResponse, error) {

	models := chat.GetLLMOptions()
	client, err := llmclient.NewClient(
		topicService.client,
		models[0],
		models[1],
		0,
		// streaming 비활성화
		chat.WithStreamDisabled,
	)
	if err != nil {
		return nil, err
	}
	messages := []*chat.ChatPayload{
		{
			Role:    "system",
			Content: getTopicKeywordPrompt(),
		},
		{
			Role:    "user",
			Content: topicMessage,
		},
	}

	// 기존 llm request 이용
	response, err := client.CreateChat(context, models[0], messages, chat.WithModel(models[1]))
	if err != nil {
		slog.Error("create keyword chat", "error", err)
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

	//"{\"published_date_range\":{\"end_date\":\"2025-01-16T23:59:59+09:00\",\"start_date\":\"2025-01-16T00:00:00+09:00\"},\"standalone_query\":\"오늘 뉴스\"}"

	arguments, err := function.ParseFunctionArguments(callResponse.Arguments)
	if err != nil {
		return nil, err
	}

	extraArgs := &function.ExtraArgs{
		RawQuery: topicMessage,
		// 최대 뉴스 조회 개수
		Topk: 2000,
		// 최대 뉴스 병합 size
		MaxChunkSize: 180000,
		// 최대 뉴스 병합 개수
		MaxChunkNumber: 2000,
	}

	// 기존 search 소스코드 활용
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
	var news_ids []string
	for i, item := range items.Items {
		news_ids = append(news_ids, item.Attributes.NewsID)
		if i < 5 {
			contents += item.Attributes.Content
		}
	}

	summaryMessage := make([]*chat.ChatPayload, 1)
	summaryMessage[0] = &chat.ChatPayload{
		Role:    "system",
		Content: getSummaryPrompt(contents),
	}

	// 기존 llm request 이용
	summaryResponse, err := client.CreateChat(context, models[0], summaryMessage, chat.WithModel(models[1]))
	if err != nil {
		slog.Error("create summary chat ", "error", err)
		return nil, err
	}
	summaryCallResponse, err := parsingSummaryResponse(summaryResponse)
	if err != nil {
		return nil, err
	}
	findTopicResponse := &FindTopicResponse{
		Title:   summaryCallResponse.Title,
		Content: summaryCallResponse.Content,
		Count:   len(items.Items),
		Ids:     news_ids,
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
