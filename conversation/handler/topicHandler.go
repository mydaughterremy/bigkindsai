package handler

import (
	"bigkinds.or.kr/conversation/internal/http/response"
	"bigkinds.or.kr/conversation/service"
	"encoding/json"
	"net/http"
)

type topicHandler struct {
	service *service.TopicService
}

type findTopicSummaryRequest struct {
	message string `json:"message"`
}

func (topicHandler *topicHandler) handleTopic(responseWriter http.ResponseWriter, request *http.Request) {

	context := request.Context()
	var topicRequest findTopicSummaryRequest
	err := json.NewDecoder(request.Body).Decode(&topicRequest)
	if err != nil {
		_ = response.WriteJsonErrorResponse(responseWriter, request, http.StatusBadRequest, err)
	}
	topicResponse, err := topicHandler.service.GetTopic(context, topicRequest.message)
	if err != nil {
		_ = response.WriteJsonErrorResponse(responseWriter, request, http.StatusBadRequest, err)
	}
	_ = response.WriteJsonResponse(responseWriter, request, http.StatusOK, topicResponse)
}
