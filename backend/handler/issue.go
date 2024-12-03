package handler

import (
	"encoding/json"
	"net/http"

	"bigkinds.or.kr/backend/internal/http/response"
	"bigkinds.or.kr/backend/service"
)

type IssueHandler struct {
	Service *service.IssueService
}

type CreateIssueTopicSummaryRequest struct {
	Topic string `json:"topic"`
}

func (h *IssueHandler) GetIssueTopicSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req CreateIssueTopicSummaryRequest
	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
		return
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusBadRequest, err)
	}

	its, err := h.Service.CreateIssueTopicSummary(ctx, reqBody)

	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}

	_ = response.WriteJsonResponse(w, r, http.StatusOK, its)
	return

}
