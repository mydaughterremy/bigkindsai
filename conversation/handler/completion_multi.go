package handler

import (
	"encoding/json"
	"net/http"
	"sync"

	"bigkinds.or.kr/conversation/internal/http/response"
	service "bigkinds.or.kr/conversation/service"
	"bigkinds.or.kr/pkg/chat/v2"
)

type completionMultiHandler struct {
	s *service.CompletionMultiService
}

type CreateChatCompletionMultiRequest struct {
	Messages []*Message `json:"messages"`
	Provider string     `json:"provider"`
}

func (h *completionMultiHandler) CreateChatCompletionMulti(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var completionMultiRequest CreateChatCompletionMultiRequest
	err := json.NewDecoder(r.Body).Decode(&completionMultiRequest)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
	}

	var chatCompletionMultiResult chan *service.CreateChatCompletionMultiResult
	payloads := make([]*chat.ChatPayload, 0, len(completionMultiRequest.Messages))
	for _, m := range completionMultiRequest.Messages {
		payloads = append(payloads, &chat.ChatPayload{
			Content: m.Content,
			Role:    m.Role,
		})
	}

	chatCompletionMultiResult, err = h.s.CreateChatCompletionMulti(ctx, &service.CreateChatCompletionMultiParameter{
		Payloads: payloads,
		Provider: completionMultiRequest.Provider,
	})
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
	}

	w.Header().Add("Content-Type", "text/event-stream;charset=utf-8")

	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case result, ok := <-chatCompletionMultiResult:
				if !ok {
					return
				}
				if result.Error != nil {
					w.WriteHeader(http.StatusInternalServerError)
					_ = response.WriteStreamErrorResponse(w, result.Error)
				} else if result.Done {
					_ = response.WriteStreamResponse(w, []byte("[DONE]"))
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

	waitGroup.Wait()

}
