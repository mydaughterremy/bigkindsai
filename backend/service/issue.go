package service

import (
	"context"
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
			ResponseHeaderTimeout: 10 * time.Second,
		},
	}

	s := &IssueService{
		convEngine: convEngine,
		client:     client,
	}

	return s, nil
}

func (s *IssueService) CreateIssueTopicSummary(ctx context.Context, rb []byte) (*model.IssueTopicSummary, error) {
	its, err := request.ConvIssueTopicSummaryRequest(ctx, s.client, s.convEngine, rb)
	if err != nil {
		return nil, err
	}
	return its, nil
}
