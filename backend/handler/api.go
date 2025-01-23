package handler

import (
	"encoding/json"
	"net/http"

	"bigkinds.or.kr/backend/internal/http/response"
	"bigkinds.or.kr/backend/model"
	"bigkinds.or.kr/backend/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ApiHandler struct {
	ApiService *service.ApiService
}

func (h *ApiHandler) CreateApikey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req model.Apikey
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
	}

	req.ID = uuid.New().String()
	// req.Apikey = uuid.New().String()

	ak, err := h.ApiService.CreateApikey(ctx, req)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
	}

	_ = response.WriteJsonResponse(w, r, http.StatusOK, ak)

}

func (h *ApiHandler) GetApikey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	k := chi.URLParam(r, "apikey")

	ak, err := h.ApiService.GetApikey(ctx, k)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
	}

	_ = response.WriteJsonResponse(w, r, http.StatusOK, ak)

}

func (h *ApiHandler) UpdateApikey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req model.Apikey
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
	}

	ak, err := h.ApiService.UpdateApikey(ctx, req)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
	}

	_ = response.WriteJsonResponse(w, r, http.StatusOK, ak)
}

func (h *ApiHandler) DeleteApikey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	k := chi.URLParam(r, "apikey")

	err := h.ApiService.DeleteApikey(ctx, k)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
	}

	_ = response.WriteJsonResponse(w, r, http.StatusOK, nil)

}

func (h *ApiHandler) ChatCompletion(w http.ResponseWriter, r *http.Request) {

}

func (h *ApiHandler) Summary(w http.ResponseWriter, r *http.Request) {

}

func (h *ApiHandler) Translate(w http.ResponseWriter, r *http.Request) {

}

func (h *ApiHandler) Article(w http.ResponseWriter, r *http.Request) {

}

func (h *ApiHandler) FileChatCompletion(w http.ResponseWriter, r *http.Request) {

}
