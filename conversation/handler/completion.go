package handler

import (
	"encoding/json"
	"net/http"
	"sync"

	"bigkinds.or.kr/conversation/internal/http/response"
	service "bigkinds.or.kr/conversation/service"
	"bigkinds.or.kr/pkg/chat/v2"
)

type completionHandler struct {
	service *service.CompletionService
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

type CreateChatCompletionRequest struct {
	Messages []*Message `json:"messages"`
	Provider string     `json:"provider"`
}

func (h *completionHandler) CreateChatCompletion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateChatCompletionRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
		return
	}

	var ch chan *service.CreateChatCompletionResult

	payloads := make([]*chat.ChatPayload, 0, len(req.Messages))
	for _, m := range req.Messages {
		payloads = append(payloads, &chat.ChatPayload{
			Content: m.Content,
			Role:    m.Role,
		})
	}

	ch, err = h.service.CreateChatCompletion(ctx,
		&service.CreateChatCompletionParameter{
			Payloads: payloads,
			Provider: req.Provider,
		})
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}

	w.Header().Add("Content-Type", "text/event-stream;charset=utf-8")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case result, ok := <-ch:
				if !ok {
					return
				}
				if result.Error != nil {
					w.WriteHeader(http.StatusInternalServerError)
					_ = response.WriteStreamErrorResponse(w, result.Error)
					return
				} else if result.Done {
					_ = response.WriteStreamResponse(w, []byte("[DONE]"))
					return
				}

				b, err := json.Marshal(result.Completion)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					_ = response.WriteStreamErrorResponse(w, err)
					return
				}
				_ = response.WriteStreamResponse(w, b)
			}
		}
	}()

	wg.Wait()
}
