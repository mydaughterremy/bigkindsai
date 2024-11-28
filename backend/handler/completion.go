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

func (h *completionHandler) CreateChatCompletion(responseWriter http.ResponseWriter, request *http.Request) {
	context := request.Context()

	var completionRequest CreateChatCompletionRequest
	err := json.NewDecoder(request.Body).Decode(&completionRequest)

	if err != nil {
		_ = response.WriteJsonErrorResponse(responseWriter, request, http.StatusBadRequest, err)
		return
	}

	chatId := chi.URLParam(request, "chat_id")

	var chatChannel chan *service.CreateChatCompletionResult
	useMultiTurn, ok := os.LookupEnv("USE_MULTI_TURN")
	if ok && useMultiTurn == "true" {
		chatChannel, err = h.service.CreateChatCompletionWithChatHistory(context,
			&service.CreateChatCompletionParameter{
				ChatID:   chatId,
				Session:  completionRequest.Session,
				JobGroup: completionRequest.JobGroup,
				Messages: []*model.Message{
					{
						Role:    "user",
						Content: completionRequest.Message,
					},
				},
				Provider: completionRequest.Provider,
			})
		if err != nil {
			_ = response.WriteJsonErrorResponse(responseWriter, request, http.StatusInternalServerError, err)
			return
		}
	} else {
		chatChannel, err = h.service.CreateChatCompletion(context,
			&service.CreateChatCompletionParameter{
				ChatID:   chatId,
				Session:  completionRequest.Session,
				JobGroup: completionRequest.JobGroup,
				Messages: []*model.Message{
					{
						Role:    "user",
						Content: completionRequest.Message,
					},
				},
				Provider: completionRequest.Provider,
			})
		if err != nil {
			_ = response.WriteJsonErrorResponse(responseWriter, request, http.StatusInternalServerError, err)
			return
		}
	}

	responseWriter.Header().Add("Content-Type", "text/event-stream;charset=utf-8")

	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		for {
			select {
			case <-context.Done():
				return
			case result, ok := <-chatChannel:
				if !ok {
					return
				}
				if result.Error != nil {
					responseWriter.WriteHeader(http.StatusInternalServerError)
					_ = response.WriteStreamErrorResponse(responseWriter, result.Error)
					return
				} else if result.Done {
					_ = response.WriteStreamResponse(responseWriter, []byte("[DONE]"))
					return
				}

				body, err := json.Marshal(result.Completion)
				if err != nil {
					responseWriter.WriteHeader(http.StatusInternalServerError)
					_ = response.WriteStreamErrorResponse(responseWriter, err)
					return
				}
				_ = response.WriteStreamResponse(responseWriter, body)
			}
		}
	}()

	waitGroup.Wait()
}
