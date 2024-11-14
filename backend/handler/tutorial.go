package handler

import (
	"net/http"

	"bigkinds.or.kr/backend/internal/http/response"
	"bigkinds.or.kr/backend/service"
)

type tutorialHandler struct {
	service *service.TutorialService
}

type GetTutorialResponse struct {
	Tutorial string `json:"tutorial"`
}

func (h *tutorialHandler) GetTutorial(w http.ResponseWriter, r *http.Request) {

	tutorial, err := h.service.GetTutorial()
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}
	sampleResponse := GetTutorialResponse{
		Tutorial: tutorial,
	}

	_ = response.WriteJsonResponse(w, r, http.StatusOK, sampleResponse)
}
