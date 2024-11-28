package service

import (
	"context"
	"net/http"
)

type TopicService struct {
	client *http.Client
}

type FindTopicResponse struct {
}

func NewTopicService() *TopicService {
	topicService := &TopicService{}
	return topicService
}

func (topicService *TopicService) GetTopic(context context.Context, topicName string) (*FindTopicResponse, error) {
	//
	//models := chat.GetModels()
	//
	//gptOptions := &chat.GptOptions{}
	//
	//client, err := llmclient.NewClient(
	//	topicService.client,
	//	models[0],
	//	models[1],
	//	0,
	//	chat.WithStreamDisabled,
	//)
	//if err != nil {
	//	return nil, err
	//}
	//client.CreateChat()
	return nil, nil
}
