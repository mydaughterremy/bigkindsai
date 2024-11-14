package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bigkinds.or.kr/backend/service"
	"github.com/joho/godotenv"
	"github.com/segmentio/kafka-go"
)

type Worker struct {
	Reader    *kafka.Reader
	QAService *service.QAService
}

func (w *Worker) Run() {
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-sigch:
			slog.Info("received signal, exiting")
			w.Reader.Close()
			return
		default:
			messages, err := readBatchMessages(w.Reader)
			if err != nil {
				slog.Error("failed to read batch messages", "error", err)
				continue
			}
			if len(messages) == 0 {
				continue
			}

			failed := make([]*kafka.Message, 0, len(messages))

			queries := make([]*Query, 0, len(messages))
			for _, message := range messages {
				event, err := convertMessageToEvent(&message)
				if err != nil {
					slog.Error("failed to convert message to event", "error", err)
					failed = append(failed, &message)
				}

				query, err := convertEventToQuery(event)
				if err != nil {
					slog.Error("failed to convert event to qa", "error", err)
					failed = append(failed, &message)
				} else {
					queries = append(queries, query)
				}
			}

			if len(failed) > 0 {
				slog.Error("failed to convert messages to events", "failed", len(failed))
			}

			updates := mergeQueries(queries)

			for _, query := range updates {
				if query == nil {
					continue
				}

				b, _ := json.Marshal(query)

				if query.ShouldCreate {
					qa, err := w.QAService.CreateQA(context.Background(), query.QA)
					if err != nil {
						slog.Error("failed to create qa", "error", err)
						continue
					}
					slog.Info("query created", "query", string(b))
					query.QA = qa
				} else {
					result, err := w.QAService.UpdateQA(context.Background(), query.QA)
					if err != nil || result == nil {
						slog.Error("failed to update qa", "error", err)
						continue
					}
					slog.Info("query updated", "query", string(b))
				}

			}

			err = w.Reader.CommitMessages(context.Background(), messages...)
			if err != nil {
				slog.Error("failed to commit messages", "error", err)
				continue
			}
		}
	}
}

func main() {
	_ = godotenv.Load()

	slog.Info("starting consumer")
	reader, err := NewReader()
	if err != nil {
		slog.Error("failed to create kafka reader", "error", err)
		return
	}
	defer reader.Close()
	slog.Info("created kafka reader")

	slog.Info("creating qa service")
	kst, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		panic(err)
	}

	qaService, err := NewQAService(kst)
	if err != nil {
		panic(err)
	}
	slog.Info("created qa service")

	worker := &Worker{
		Reader:    reader,
		QAService: qaService,
	}

	slog.Info("starting worker")
	worker.Run()
}
