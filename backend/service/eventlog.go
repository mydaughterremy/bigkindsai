package service

import (
	"context"
	"errors"
	"log/slog"

	"bigkinds.or.kr/proto/event"
	kafka "github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type EventLogService struct {
	writer *kafka.Writer
}

func NewEventLogService(writer *kafka.Writer) *EventLogService {
	return &EventLogService{
		writer: writer,
	}
}

func (s *EventLogService) WriteEvent(ctx context.Context, event *event.Event) error {
	slog.Info("writing event", "event", event)
	if event.GetQaId() == "" {
		return errors.New("qa_id is required")
	}

	if event.GetCreatedAt() == nil {
		event.CreatedAt = timestamppb.Now()
	}

	b, err := protojson.Marshal(event)
	if err != nil {
		return err
	}

	err = s.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.GetQaId()),
		Value: b,
	})

	if err != nil {
		slog.Error("failed to write event", "error", err)
	}

	return err
}
