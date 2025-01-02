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

type LoginChatRequest struct {
	ChatId string `json:"chat_id"`
	User   string `json:"user"`
}

type CreateUserChatRequest struct {
	User string `json:"user"`
}

type ChatError struct {
	ErrMessage string
}

func (c *ChatError) Error() string {
	return c.ErrMessage
}

func (h *ChatHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := LoginChatRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
	}

	chatqas, err := h.ChatService.ChatLogin(ctx, req.ChatId, req.User)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
	}

	_ = response.WriteJsonResponse(w, r, http.StatusOK, chatqas)

}

func (h *ChatHandler) CreateUserChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := CreateUserChatRequest{}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
	}

	chat, err := h.ChatService.CreateUserChat(ctx, req.User)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
	}

	_ = response.WriteJsonResponse(w, r, http.StatusCreated, chat)

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

// 채팅방 대화내용 가져오기 godoc
// @Summary 채팅방의 대화 이력 조회
// @Description 채팅방 아이디를 전달하면 해당 채팅방 아이디에 해당하는 대화이력을 조회 함
// @Tags chats
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer {token}" default(Bearer upstage_kindsai_key)
// @Param chat_id path string true "chat_id" default(ffacea9b-d5a1-4844-8a0f-520b69a93ac3)
// @Router /v2/chats/{chat_id}/qas [get]
func (h *ChatHandler) GetQAs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chatId := chi.URLParam(r, "chat_id")

	qas, err := h.ChatService.GetQAs(ctx, chatId)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
	}

	_ = response.WriteJsonResponse(w, r, http.StatusOK, qas)

}

func (h *ChatHandler) GetUserChats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	u := r.URL.Query().Get("user")

	if u == "" {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, &ChatError{"user value is empty"})
	}

	chatqas, err := h.ChatService.GetChatQAs(ctx, u)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
	}

	_ = response.WriteJsonResponse(w, r, http.StatusOK, chatqas)

}

// CreateChat godoc
// @Summary Create a new chat
// @Description Create a new chat
// @Tags chats
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer {token}" default(Bearer upstage_kindsai_key)
// @Param message body CreateChatRequest true "CreateChatRequest"
// @Router /v1/chats/ [post]
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
	// User  string `json:"user"`
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
