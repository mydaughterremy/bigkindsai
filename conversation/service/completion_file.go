package service

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"bigkinds.or.kr/conversation/internal/llmclient"
	"bigkinds.or.kr/conversation/internal/token_counter"
	model "bigkinds.or.kr/conversation/model"
	"bigkinds.or.kr/pkg/chat/v2"
	"bigkinds.or.kr/pkg/chat/v2/gpt"
	"bigkinds.or.kr/pkg/utils"
	"github.com/google/uuid"
)

type CompletionFileService struct {
	PromptService *PromptService

	client       *http.Client
	tokenCounter *token_counter.TokenCounter
}

type CreateChatCompletionFileResult struct {
	Completion *model.Completion `json:"completion"`
	Done       bool              `json:"done"`
	Error      error             `json:"error"`
}

type CreateChatCompletionFileParameter struct {
	Payloads []*chat.ChatPayload `json:"payloads"`
}

func NewCompletionFileService(tokenCounter *token_counter.TokenCounter) *CompletionFileService {
	client := &http.Client{
		Transport: &http.Transport{
			ResponseHeaderTimeout: 60 * time.Second,
		},
	}

	return &CompletionFileService{
		client:       client,
		tokenCounter: tokenCounter,
	}

}

func (s *CompletionFileService) createInitialPayloads(ct utils.CurrentTime, fm string) ([]*chat.ChatPayload, error) {
	p := s.PromptService.GetFileChatPrompt(ct.Time.Format("2006-01-02T15:04:05-07:00"), fm)
	var px []*chat.ChatPayload
	px = append(px, &chat.ChatPayload{
		Content: p,
		Role:    "system",
	})
	return px, nil
}

func (s *CompletionFileService) findLastUserMessage(px []*chat.ChatPayload) *chat.ChatPayload {
	for i := len(px) - 1; i >= 0; i-- {
		if px[i].Role == "user" {
			return px[i]
		}
	}

	return nil
}

func (s *CompletionFileService) CreateChatCompletionFile(ctx context.Context, p *CreateChatCompletionFileParameter) (chan *CreateChatCompletionFileResult, error) {
	px := p.Payloads
	cId := uuid.New().String()
	lp := "upstage"

	um := s.findLastUserMessage(px)

	fPx := px[:len(px)-1]
	fm, err := json.Marshal(fPx)
	if err != nil {
		return nil, err
	}

	ct, err := utils.GetCurrentKSTTime()
	if err != nil {
		return nil, err
	}

	sPx, err := s.createInitialPayloads(ct, string(fm))
	if err != nil {
		return nil, err
	}

	sPx = append(sPx, um)

	po := setPredictOpts()
	po = append(po, chat.WithStream)

	sp := &model.CompletionLLM{
		Provider:  lp,
		ModelName: "solar-pro",
	}
	po = append(po, chat.WithModel(sp.ModelName))
	po = append(po, chat.WithNilTollCall)
	po = append(po, chat.WithNilTollChoice)
	po = append(po, chat.WithNoFunctions)

	client, err := llmclient.NewClient(
		s.client,
		lp,
		sp.ModelName,
		0,
		chat.WithStreamEnabled,
	)
	if err != nil {
		return nil, err
	}

	resC := make(chan *CreateChatCompletionFileResult, 20)

	go func() {
		stream, err := client.CreateChatStream(ctx, sp.Provider, sPx, po...)
		if err != nil {
			slog.Info("===== ===== CreateChatCompletionFile -> CreateChatStream error")
			resC <- &CreateChatCompletionFileResult{
				Error: err,
			}
			return
		}
		defer func(stream gpt.ChatStream) {
			err := stream.Close()
			if err != nil {
				slog.Error("===== ===== CreateChatCompletinFile -> failed to Close stream", "error", err)
			}
		}(stream)

		done := false

		for {
			if done {
				break
			}

			select {
			case <-ctx.Done():
				resC <- &CreateChatCompletionFileResult{
					Error: ctx.Err(),
				}
				return
			default:
				resp, err := stream.Recv(lp)
				if err != nil && err != io.EOF {
					resC <- &CreateChatCompletionFileResult{
						Error: err,
					}
					return
				}
				if err == io.EOF {
					done = true
					resC <- &CreateChatCompletionFileResult{
						Done: true,
					}
					break
				}

				resC <- &CreateChatCompletionFileResult{
					Completion: &model.Completion{
						Object:  "chat.completion",
						Id:      cId,
						Created: int(time.Now().Unix()),
						Delta: model.CompletionDelta{
							Content: resp.Payload.Content,
						},
						TokenUsage: s.tokenCounter.CountTokens(resp.Payload.Content),
					},
				}
			}

		}
	}()

	return resC, nil
}
