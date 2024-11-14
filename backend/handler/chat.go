package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"bigkinds.or.kr/backend/internal/http/response"
	"bigkinds.or.kr/backend/model"
	"bigkinds.or.kr/backend/service"
)

type ChatHandler struct {
	ChatService *service.ChatService
}

type CreateChatRequest struct {
	Session string `json:"session"`
}

func (h *ChatHandler) GetChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chatId := chi.URLParam(r, "chat_id")
	u, err := uuid.Parse(chatId)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
		return
	}
	chat, err := h.ChatService.GetChat(ctx, u)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusNotFound, err)
			return
		} else {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
			return
		}
	}
	_ = response.WriteJsonResponse(w, r, http.StatusOK, chat)
}

func (h *ChatHandler) CreateChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req := CreateChatRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
		return
	}

	chat, err := h.ChatService.CreateChat(ctx, req.Session)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}

	_ = response.WriteJsonResponse(w, r, http.StatusCreated, chat)
}

func (h *ChatHandler) ListChats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	session := r.URL.Query().Get("session")
	if session == "" {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, fmt.Errorf("session is required"))
		return
	}

	chats, err := h.ChatService.ListChats(ctx, session)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}

	_ = response.WriteJsonResponse(w, r, http.StatusOK, chats)
}

func (h *ChatHandler) ListChatQAs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chatID := chi.URLParam(r, "chat_id")
	session := r.URL.Query().Get("session")
	if session == "" {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, fmt.Errorf("session is required"))
		return
	}

	chats, err := h.ChatService.ListChatQAs(ctx, session, chatID)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}

	_ = response.WriteJsonResponse(w, r, http.StatusOK, chats)

}
func (h *ChatHandler) DeleteChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chatId := chi.URLParam(r, "chat_id")
	u, err := uuid.Parse(chatId)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
		return
	}

	err = h.ChatService.DeleteChat(ctx, u)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusNotFound, err)
			return
		}
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}
	_ = response.WriteJsonResponse(w, r, http.StatusNoContent, nil)
}

type UpdateChatTitleRequest struct {
	User  string `json:"user"`
	Title string `json:"title"`
}

func (h *ChatHandler) UpdateChatTitle(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req := &UpdateChatTitleRequest{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
		return
	}

	// validate title
	if req.Title == "" {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusUnprocessableEntity, fmt.Errorf("title must not be empty"))
		return
	}

	chatId := chi.URLParam(r, "chat_id")
	u, err := uuid.Parse(chatId)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
		return
	}
	chat, err := h.ChatService.UpdateChatTitle(ctx, u, req.Title)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			_ = response.WriteJsonErrorResponse(w, r, http.StatusNotFound, err)
			return
		}
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}
	_ = response.WriteJsonResponse(w, r, http.StatusOK, chat)
}

type ListChatQAsResponse struct {
	QAs []*model.QA `json:"qas"`
}
