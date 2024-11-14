package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

func NewReader() (*kafka.Reader, error) {
	brokers := strings.Split(os.Getenv("UPSTAGE_KAFKA_BROKERS"), ",")
	group := os.Getenv("UPSTAGE_KAFKA_GROUP")
	topic := os.Getenv("UPSTAGE_KAFKA_TOPIC")

	var err error
	batchSize := 100
	batchSizeStr, ok := os.LookupEnv("UPSTAGE_KAFKA_BATCH_SIZE")
	if ok {
		batchSize, err = strconv.Atoi(batchSizeStr)
		if err != nil {
			slog.Error("failed to parse batch size", "error", err)
			return nil, err
		}
	}

	timeout := 500
	timeoutStr, ok := os.LookupEnv("UPSTAGE_KAFKA_TIMEOUT")
	if ok {
		timeout, err = strconv.Atoi(timeoutStr)
		if err != nil {
			slog.Error("failed to parse timeout", "error", err)
			return nil, err
		}
	}

	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:          brokers,
		GroupID:          group,
		Topic:            topic,
		MaxWait:          time.Duration(timeout) * time.Millisecond,
		ReadBatchTimeout: time.Duration(timeout) * time.Millisecond,
		QueueCapacity:    batchSize,
	}), nil
}

func readBatchMessages(reader *kafka.Reader) ([]kafka.Message, error) {
	var messages []kafka.Message
	batchSize := reader.Config().QueueCapacity
	timeOut := reader.Config().MaxWait

	timer := time.NewTimer(timeOut)

	for {
		select {
		case <-timer.C:
			return messages, nil
		default:
			ctx, cancel := context.WithTimeout(context.Background(), timeOut)
			defer cancel()
			m, err := reader.FetchMessage(ctx)

			if err != nil {
				if err == context.DeadlineExceeded {
					continue
				}
				return nil, err
			}

			slog.Info("message received", "message", string(m.Value))
			messages = append(messages, m)
			if len(messages) == batchSize {
				return messages, nil
			}
		}
	}
}
