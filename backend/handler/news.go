package handler

import (
	"encoding/json"
	"net/http"

	"bigkinds.or.kr/backend/internal/http/response"
	"bigkinds.or.kr/backend/model"
	"bigkinds.or.kr/backend/service"
)

type NewsHandler struct {
	Service *service.NewsService
}

func (h *NewsHandler) GetNewsSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req *model.NewsSummaryRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
		return
	}

	ns, err := h.Service.GetNewsSummary(ctx, req)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}

	_ = response.WriteJsonResponse(w, r, http.StatusOK, ns)

}
