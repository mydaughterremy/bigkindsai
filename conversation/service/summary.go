package service

import (
	"bigkinds.or.kr/conversation/internal/llmclient"
	"bigkinds.or.kr/pkg/chat/v2"
	"context"
	"log/slog"
	"net/http"
	"time"
)

type SummaryService struct {
	client *http.Client
}
type SummaryResultResponse struct {
	Content string `json:"content"`
}

func NewSummaryService() *SummaryService {

	return &SummaryService{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (summaryService *SummaryService) ContentSummary(context context.Context, content string) (*SummaryResultResponse, error) {

	models := chat.GetLLMOptions()
	client, err := llmclient.NewClient(
		summaryService.client,
		models[0],
		models[1],
		0,
		chat.WithStreamDisabled,
	)
	if err != nil {
		return nil, err
	}
	summaryMessage := make([]*chat.ChatPayload, 1)
	summaryMessage[0] = &chat.ChatPayload{
		Role:    "system",
		Content: getSummaryPrompt(content),
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
	findTopicResponse := &SummaryResultResponse{
		Content: summaryCallResponse.Content,
	}
	return findTopicResponse, nil
}
