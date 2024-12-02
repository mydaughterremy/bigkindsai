package handler

import (
	"encoding/json"
	"net/http"

	"bigkinds.or.kr/conversation/internal/http/response"
	"bigkinds.or.kr/conversation/service"
)

type topicHandler struct {
	service *service.TopicService
}

type findTopicSummaryRequest struct {
	Topic string `json:"topic"`
}

func (topicHandler *topicHandler) handleTopic(responseWriter http.ResponseWriter, request *http.Request) {

	context := request.Context()
	var topicRequest findTopicSummaryRequest
	err := json.NewDecoder(request.Body).Decode(&topicRequest)
	if err != nil {
		_ = response.WriteJsonErrorResponse(responseWriter, request, http.StatusBadRequest, err)
	}
	topicResponse, err := topicHandler.service.GetTopic(context, topicRequest.Topic)
	if err != nil {
		_ = response.WriteJsonErrorResponse(responseWriter, request, http.StatusBadRequest, err)
	}
	if topicResponse == nil{
		_ = response.WriteJsonResponse(responseWriter, request, http.StatusNoContent, topicResponse)	
	}else{
		_ = response.WriteJsonResponse(responseWriter, request, http.StatusOK, topicResponse)
	}
	
}
