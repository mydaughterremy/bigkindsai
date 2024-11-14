package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"

	"bigkinds.or.kr/backend/internal/http/response"
	"bigkinds.or.kr/backend/model"
	"bigkinds.or.kr/backend/service"
	"github.com/go-chi/chi/v5"
)

type completionHandler struct {
	service *service.CompletionService
}

type CreateChatCompletionRequest struct {
	Message  string `json:"message"`
	Session  string `json:"session"`
	JobGroup string `json:"job_group"`
	Provider string `json:"provider"`
}

func (h *completionHandler) CreateChatCompletion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateChatCompletionRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
		return
	}

	chatId := chi.URLParam(r, "chat_id")

	var ch chan *service.CreateChatCompletionResult
	useMultiTurn, ok := os.LookupEnv("USE_MULTI_TURN")
	if ok && useMultiTurn == "true" {
		ch, err = h.service.CreateChatCompletionWithChatHistory(ctx,
			&service.CreateChatCompletionParameter{
				ChatID:   chatId,
				Session:  req.Session,
				JobGroup: req.JobGroup,
				Messages: []*model.Message{
					{
						Content: req.Message,
						Role:    "user",
					},
				},
				Provider: req.Provider,
			})
		if err != nil {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
			return
		}
	} else {
		ch, err = h.service.CreateChatCompletion(ctx,
			&service.CreateChatCompletionParameter{
				ChatID:   chatId,
				Session:  req.Session,
				JobGroup: req.JobGroup,
				Messages: []*model.Message{
					{
						Content: req.Message,
						Role:    "user",
					},
				},
				Provider: req.Provider,
			})
		if err != nil {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
			return
		}
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
