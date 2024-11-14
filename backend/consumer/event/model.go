package main

import (
	"fmt"

	"bigkinds.or.kr/backend/model"
	pb "bigkinds.or.kr/proto/event"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/encoding/protojson"
)

func fromProtoReferenceToModelReference(reference *pb.Reference) (*model.Reference, error) {
	if reference == nil {
		return nil, fmt.Errorf("reference is nil")
	}

	if reference.GetId() == "" {
		return nil, fmt.Errorf("reference id is empty")
	}

	return &model.Reference{
		ID: reference.GetId(),
		Attributes: model.ReferenceAttributes{
			NewsID:      reference.GetAttributes().GetNewsId(),
			Title:       reference.GetAttributes().GetTitle(),
			PublishedAt: reference.GetAttributes().GetPublishedAt().AsTime(),
			Provider:    reference.GetAttributes().GetProvider(),
			Byline:      reference.GetAttributes().GetByline(),
			Content:     reference.GetAttributes().GetContent(),
		},
	}, nil
}

func convertMessageToEvent(message *kafka.Message) (*pb.Event, error) {
	var event pb.Event
	err := protojson.Unmarshal(message.Value, &event)
	if err != nil {
		return nil, err
	}

	return &event, nil
}

func convertEventToQuery(event *pb.Event) (*Query, error) {
	if event.GetQaId() == "" {
		return nil, fmt.Errorf("event id is empty")
	}

	if event.GetEvent() == nil {
		return nil, fmt.Errorf("event is empty")
	}

	id := event.GetQaId()
	if id == "" {
		return nil, fmt.Errorf("qa id is empty")
	}

	createdAt := event.GetCreatedAt()
	if createdAt == nil {
		return nil, fmt.Errorf("event created at is empty")
	}

	qa := &model.QA{
		ID:        id,
		UpdatedAt: createdAt.AsTime(),
	}

	shouldCreate := false

	switch t := event.GetEvent().(type) {
	case *pb.Event_QuestionCreated:
		chatId := t.QuestionCreated.GetChatId()
		if chatId == "" {
			return nil, fmt.Errorf("chat id is empty")
		}
		qa.ChatID = chatId
		SessionId := t.QuestionCreated.GetSessionId()
		if SessionId == "" {
			return nil, fmt.Errorf("session id is empty")
		}
		qa.SessionID = SessionId
		JobGroup := t.QuestionCreated.GetJobGroup()
		qa.JobGroup = JobGroup
		qa.Question = t.QuestionCreated.GetQuestion()
		qa.CreatedAt = createdAt.AsTime()
		shouldCreate = true
	case *pb.Event_AnswerUpdated:
		qa.Answer = t.AnswerUpdated.GetAnswer()
	case *pb.Event_TokenCountUpdated:
		qa.TokenCount = int(t.TokenCountUpdated.GetTokenCount())
	case *pb.Event_ReferencesCreated:
		for _, ref := range t.ReferencesCreated.GetReferences() {
			modelRef, err := fromProtoReferenceToModelReference(ref)
			if err != nil {
				return nil, err
			}
			qa.References = append(qa.References, modelRef)
		}
	case *pb.Event_KeywordsCreated:
		keywords := t.KeywordsCreated.GetKeywords()
		qa.Keywords = keywords
	case *pb.Event_RelatedQueriesCreated:
		relatedQueries := t.RelatedQueriesCreated.GetRelatedQueries()
		qa.RelatedQueries = relatedQueries
	}

	return &Query{
		QA:           qa,
		ShouldCreate: shouldCreate,
	}, nil
}
