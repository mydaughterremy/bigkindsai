package handler

import (
	"encoding/json"
	"net/http"
	"sync"

	"bigkinds.or.kr/conversation/internal/http/response"
	service "bigkinds.or.kr/conversation/service"
	"bigkinds.or.kr/pkg/chat/v2"
)

type CompletionFileHandler struct {
	s *service.CompletionFileService
}

type CreateChatCompletionFileRequest struct {
	Messages []*Message `json:"messages"`
	Provider string     `json:"provider"`
}

func (h *CompletionFileHandler) CreateChatCompletionFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var cReq CreateChatCompletionFileRequest
	err := json.NewDecoder(r.Body).Decode(&cReq)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
	}

	var resC chan *service.CreateChatCompletionFileResult

	payloads := make([]*chat.ChatPayload, 0, len(cReq.Messages))
	for _, m := range cReq.Messages {
		payloads = append(payloads, &chat.ChatPayload{
			Content: m.Content,
			Role:    m.Role,
		})
	}

	resC, err = h.s.CreateChatCompletionFile(ctx, &service.CreateChatCompletionFileParameter{
		Payloads: payloads,
	})

	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
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
			case res, ok := <-resC:
				if !ok {
					return
				}
				if res.Error != nil {
					w.WriteHeader(http.StatusInternalServerError)
					_ = response.WriteStreamErrorResponse(w, res.Error)
				} else if res.Done {
					_ = response.WriteStreamResponse(w, []byte("[DONE]"))
				}

				b, err := json.Marshal(res.Completion)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					_ = response.WriteStreamErrorResponse(w, err)
				}

				_ = response.WriteStreamResponse(w, b)
			}

		}

	}()

	wg.Wait()

}
