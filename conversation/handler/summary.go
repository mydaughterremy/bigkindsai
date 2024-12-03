package handler

import (
	"bigkinds.or.kr/conversation/internal/http/response"
	"bigkinds.or.kr/conversation/service"
	"encoding/json"
	"net/http"
)

type summaryHandler struct {
	service *service.SummaryService
}
type summaryContentRequest struct {
	Content string `json:"content"`
}

func (summaryHandler *summaryHandler) SummaryContent(responseWriter http.ResponseWriter, request *http.Request) {
	context := request.Context()
	var summaryRequest summaryContentRequest
	err := json.NewDecoder(request.Body).Decode(&summaryRequest)
	if err != nil {
		_ = response.WriteJsonErrorResponse(responseWriter, request, http.StatusBadRequest, err)
	}
	topicResponse, err := summaryHandler.service.ContentSummary(context, summaryRequest.Content)
	if err != nil {
		_ = response.WriteJsonErrorResponse(responseWriter, request, http.StatusBadRequest, err)
	}
	_ = response.WriteJsonResponse(responseWriter, request, http.StatusOK, topicResponse)
}
