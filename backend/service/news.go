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

type NewsService struct {
	convEngine string
	client     *http.Client
}

func NewNewsService() (*NewsService, error) {
	convEngine, ok := os.LookupEnv("UPSTAGE_CONVERSATION_ENGINE")
	if !ok {
		return nil, errors.New("os env UPSTAGE_CONVERSATION_ENGINE is not set")
	}

	client := &http.Client{
		Transport: &http.Transport{
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}

	s := &NewsService{
		convEngine: convEngine,
		client:     client,
	}

	return s, nil
}

func (s *NewsService) GetNewsSummary(ctx context.Context, nsrq *model.NewsSummaryRequest) (*model.NewsSummaryResponse, error) {
	rb, err := json.Marshal(nsrq)
	if err != nil {
		return nil, err
	}

	nsrp, err := request.ConvNewsSummaryRequest(ctx, s.client, s.convEngine, rb)
	if err != nil {
		return nil, err
	}

	return nsrp, nil
}
