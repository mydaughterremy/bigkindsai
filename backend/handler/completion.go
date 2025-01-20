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
	Message  string `json:"message" example:"트럼프 당선에 대해서 알려줘"`
	Session  string `json:"session" example:"session_id_value"`
	JobGroup string `json:"job_group" example:"통계용"`
	Provider string `json:"provider" example:""`
}

type CreateChatCompletionFileRequest struct {
	Message  string `json:"message" example:"파일에서 제일 중요한 내용 찾아줘"`
	Session  string `json:"session" example:"session_id_value"`
	JobGroup string `json:"job_group" example:"통계용"`
}

type DevCreateChatCompletionMulti struct {
	ChatId string `json:"chat_id"`
}

// CreateChatCompletionMulti godoc
// @Summary Create a new CompletionMulti
// @Description Create a new CompletionMulti
// @Tags chats
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer {token}" default(Bearer upstage_kindsai_key)
// @Param chat_id path string true "chat_id" default(ffacea9b-d5a1-4844-8a0f-520b69a93ac3)
// @Param message body CreateChatCompletionRequest true "CreateChatCompletionRequest"
// @Success 201 {object} service.CreateChatCompletionResult
// @Router /v2/chats/{chat_id}/completions/multi [post]
func (h *completionHandler) CreateChatCompletionMulti(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateChatCompletionRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
		return
	}

	chatId := chi.URLParam(r, "chat_id")

	var chatChannel chan *service.CreateChatCompletionResult
	chatChannel, err = h.service.CreateChatCompletionMulti(ctx,
		&service.CreateChatCompletionParameter{
			ChatID:   chatId,
			Session:  req.Session,
			JobGroup: req.JobGroup,
			Messages: []*model.Message{
				{
					Role:    "user",
					Content: req.Message,
				},
			},
			Provider: req.Provider,
		})
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
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
			case result, ok := <-chatChannel:
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

				body, err := json.Marshal(result.Completion)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					_ = response.WriteStreamErrorResponse(w, err)
					return
				}
				_ = response.WriteStreamResponse(w, body)
			}
		}
	}()

	waitGroup.Wait()
}

// CreateChatCompletion godoc
// @Summary Create a new Completion
// @Description Create a new Completion
// @Tags chats
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer {token}" default(Bearer upstage_kindsai_key)
// @Param chat_id path string true "chat_id" default(ffacea9b-d5a1-4844-8a0f-520b69a93ac3)
// @Param message body CreateChatCompletionRequest true "CreateChatCompletionRequest"
// @Success 201 {object} service.CreateChatCompletionResult
// @Router /v1/chats/{chat_id}/completions [post]
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
