package handler

import (
	"net/http"

	"bigkinds.or.kr/backend/internal/http/response"
	"bigkinds.or.kr/backend/service"
)

type questionGuidesHandler struct {
	service *service.QuestionGuidesService
}

type GetQuestionGuidesResponse struct {
	Guide string `json:"guide"`
}

type GetQuestionGuidesTipsResponse struct {
	Tips []string `json:"tips"`
}

func (h *questionGuidesHandler) GetQuestionGuides(w http.ResponseWriter, r *http.Request) {

	guide, err := h.service.GetQuestionGuides()
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}
	sampleResponse := GetQuestionGuidesResponse{
		Guide: guide,
	}

	_ = response.WriteJsonResponse(w, r, http.StatusOK, sampleResponse)
}

func (h *questionGuidesHandler) GetQuestionGuidesTips(w http.ResponseWriter, r *http.Request) {
	tips, err := h.service.GetQuestionGuidesTips()
	if err != nil {
		_ = response.WriteJsonErrorResponse(w, r, http.StatusInternalServerError, err)
		return
	}
	sampleResponse := GetQuestionGuidesTipsResponse{
		Tips: tips,
	}

	_ = response.WriteJsonResponse(w, r, http.StatusOK, sampleResponse)
}
