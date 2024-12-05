package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"bigkinds.or.kr/backend/internal/http/request"
	"bigkinds.or.kr/backend/model"
)

type IssueService struct {
	convEngine string
	client     *http.Client
}

type IssueTopicSummaryParam struct {
	Topic string `json:"topic"`
}

func NewIssueService() (*IssueService, error) {
	convEngine, ok := os.LookupEnv("UPSTAGE_CONVERSATION_ENGINE")
	if !ok {
		return nil, errors.New("UPSTAGE_CONVERSATION_ENGINE is not set")
	}

	client := &http.Client{
		Transport: &http.Transport{
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}

	s := &IssueService{
		convEngine: convEngine,
		client:     client,
	}

	return s, nil
}

func (s *IssueService) CreateIssueTopicSummary(ctx context.Context, topic string) (*model.IssueTopicSummary, error) {
	itp := IssueTopicSummaryParam{
		Topic: topic,
	}

	reqBody, err := json.Marshal(itp)
	if err != nil {
		return nil, err
	}

	its, err := request.ConvIssueTopicSummaryRequest(ctx, s.client, s.convEngine, reqBody)
	if err != nil {
		return nil, err
	}
	return its, nil
}
