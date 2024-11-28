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

func (handler *completionHandler) CreateChatCompletion(responseWriter http.ResponseWriter, request *http.Request) {
	context := request.Context()

	var completionRequest CreateChatCompletionRequest
	err := json.NewDecoder(request.Body).Decode(&completionRequest)

	if err != nil {
		_ = response.WriteJsonErrorResponse(responseWriter, request, http.StatusBadRequest, err)
		return
	}

	var chatCompletionResult chan *service.CreateChatCompletionResult

	payloads := make([]*chat.ChatPayload, 0, len(completionRequest.Messages))
	for _, message := range completionRequest.Messages {
		payloads = append(payloads, &chat.ChatPayload{
			Content: message.Content,
			Role:    message.Role,
		})
	}

	chatCompletionResult, err = handler.service.CreateChatCompletion(context,
		&service.CreateChatCompletionParameter{
			Payloads: payloads,
			Provider: completionRequest.Provider,
		})
	if err != nil {
		_ = response.WriteJsonErrorResponse(responseWriter, request, http.StatusInternalServerError, err)
		return
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
			case result, ok := <-chatCompletionResult:
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

				b, err := json.Marshal(result.Completion)
				if err != nil {
					responseWriter.WriteHeader(http.StatusInternalServerError)
					_ = response.WriteStreamErrorResponse(responseWriter, err)
					return
				}
				_ = response.WriteStreamResponse(responseWriter, b)
			}
		}
	}()

	waitGroup.Wait()
}
