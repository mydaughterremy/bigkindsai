package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"bigkinds.or.kr/backend/internal/http/request"
	"bigkinds.or.kr/backend/model"

	pb "bigkinds.or.kr/proto/event"
)

type CompletionService struct {
	ChatService     *ChatService
	EventLogService *EventLogService

	convEngineEndpoint string
	client             *http.Client
}

type CreateChatCompletionParameter struct {
	ChatID   string           `json:"chat_id"`
	Messages []*model.Message `json:"payloads"`
	Session  string           `json:"session"`
	JobGroup string           `json:"job_group"`
	Provider string           `json:"provider"`
}

type CreateChatCompletionResult struct {
	Completion *model.Completion `json:"completion"`
	Done       bool              `json:"done"`
	Error      error             `json:"error"`
}

func NewCompletionService(
	chatService *ChatService,
	eventLogService *EventLogService,
) (*CompletionService, error) {

	convEngineEndpoint, ok := os.LookupEnv("UPSTAGE_CONVERSATION_ENGINE_ENDPOINT")
	if !ok {
		return nil, errors.New("UPSTAGE_CONVERSATION_ENGINE_ENDPOINT is not set")
	}

	client := &http.Client{
		Transport: &http.Transport{
			ResponseHeaderTimeout: 20 * time.Second,
		},
	}

	service := &CompletionService{
		ChatService:        chatService,
		EventLogService:    eventLogService,
		convEngineEndpoint: convEngineEndpoint,
		client:             client,
	}

	return service, nil
}

func (s *CompletionService) CreateChatCompletionMulti(ctx context.Context, param *CreateChatCompletionParameter) (chan *CreateChatCompletionResult, error) {
	timeoutctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	qas, err := s.ChatService.ListChatQAsLimit(timeoutctx, param.ChatID, 5)
	if err != nil {
		slog.Error("error getting chat qa at database", "error", err.Error())
		qas = make([]*model.QA, 0)
	}

	messages := make([]*model.Message, 0)

	for i := 0; i < len(qas); i++ {
		qa := qas[i]
		if qa.Answer == "" || qa.Question == "" {
			continue
		}

		messages = append(messages, &model.Message{
			Content: qa.Answer,
			Role:    "assistant",
		})

		if qa.References != nil {
			b, err := json.Marshal(qa.References)
			if err != nil {
				return nil, err
			}

			functionName := "search"

			messages = append(messages, &model.Message{
				Content: string(b),
				Role:    "tool",
				Name:    functionName,
			})
		}

		messages = append(messages, &model.Message{
			Role:    "user",
			Content: qa.Question,
		})
	}

	slices.Reverse(messages)

	messages = append(messages, param.Messages...)

	qaId := uuid.New().String()
	chatId := param.ChatID
	sessionId := param.Session
	jobGroup := param.JobGroup
	articleProvider := param.Provider
	if chatId == "" || sessionId == "" {
		return nil, fmt.Errorf("chat id, session id is empty")
	}

	if sessionId == "error-test" {
		return nil, fmt.Errorf("error test")
	}

	// send QuestionCreated event
	lastUserMessage := s.findLastUserMessage(messages)
	if lastUserMessage == nil {
		return nil, fmt.Errorf("there is no user message")
	}
	questionCreatedEvent := &pb.Event{
		QaId:      qaId,
		CreatedAt: timestamppb.New(time.Now()),
		Event: &pb.Event_QuestionCreated{
			QuestionCreated: &pb.QuestionCreated{
				ChatId:    chatId,
				SessionId: sessionId,
				JobGroup:  jobGroup,
				Question:  lastUserMessage.Content,
			},
		},
	}
	err = s.EventLogService.WriteEvent(ctx, questionCreatedEvent)
	if err != nil {
		return nil, err
	}

	completionMultiChannel := make(chan *CreateChatCompletionResult, 10)

	go func() {
		defer close(completionMultiChannel)

		if sessionId == "stream-error-test" {
			completionMultiChannel <- &CreateChatCompletionResult{
				Error: fmt.Errorf("stream error test"),
			}
			return
		}

		lastUserPayloads := messages[len(messages)-1]
		if lastUserPayloads.Role != "user" {
			completionMultiChannel <- &CreateChatCompletionResult{
				Error: fmt.Errorf("last payload role should be user"),
			}
			return
		}
		if len(strings.Split(lastUserPayloads.Content, " ")) == 1 {
			completion := &model.Completion{
				Object:  "chat.completion",
				Id:      qaId,
				Created: int(time.Now().Unix()),
				Delta: &model.CompletionDelta{
					Content: "더 구체적인 질문을 해 주세요",
				},
			}
			completionMultiChannel <- &CreateChatCompletionResult{
				Completion: completion,
			}
			completionMultiChannel <- &CreateChatCompletionResult{
				Done: true,
			}
			return
		}
		tokenCount := 0

		completionRequest := model.CreateChatCompletionRequest{
			Messages: messages,
			Provider: articleProvider,
		}
		requestBody, err := json.Marshal(completionRequest)
		if err != nil {
			completionMultiChannel <- &CreateChatCompletionResult{
				Error: err,
			}
			return
		}
		stream, err := request.CreateChatStream(ctx, s.client, s.convEngineEndpoint+"/multi", requestBody)
		if err != nil {
			completionMultiChannel <- &CreateChatCompletionResult{
				Error: err,
			}
			return
		}
		defer stream.Close()

		for {
			select {
			case <-ctx.Done():
				completionMultiChannel <- &CreateChatCompletionResult{
					Error: ctx.Err(),
				}
				return
			default:
				resp, err := stream.Recv()
				if err != nil && err != io.EOF {
					completionMultiChannel <- &CreateChatCompletionResult{
						Error: err,
					}

					return
				}
				if err == io.EOF {
					return
				}

				completion := &model.Completion{
					Object:  "chat.completion",
					Id:      qaId,
					Created: int(time.Now().Unix()),
					Delta:   resp.Delta,
				}

				completionMultiChannel <- &CreateChatCompletionResult{
					Completion: completion,
				}

				// count token
				tokenCount += resp.TokenUsage
				_ = s.EventLogService.WriteEvent(ctx, &pb.Event{
					QaId: qaId,
					Event: &pb.Event_TokenCountUpdated{
						TokenCountUpdated: &pb.TokenCountUpdated{
							TokenCount: int32(tokenCount),
						},
					},
					CreatedAt: timestamppb.New(time.Now()),
				})

				// write event
				merged := stream.ReadUntilNow()

				if merged.Delta.Content != "" {
					answerUpdatedEvent := &pb.Event{
						QaId: qaId,
						Event: &pb.Event_AnswerUpdated{
							AnswerUpdated: &pb.AnswerUpdated{
								Answer: merged.Delta.Content,
								// llm model, llm provider 없는데 ?
							},
						},
						CreatedAt: timestamppb.New(time.Now()),
					}
					_ = s.EventLogService.WriteEvent(ctx, answerUpdatedEvent)
				}

				if len(merged.Delta.References) > 0 {
					referencesCreatedEvent := &pb.Event{
						QaId: qaId,
						Event: &pb.Event_ReferencesCreated{
							ReferencesCreated: &pb.ReferencesCreated{
								References: model.FromModelReferencesToProtoReferences(merged.Delta.References),
							},
						},
						CreatedAt: timestamppb.New(time.Now()),
					}
					_ = s.EventLogService.WriteEvent(ctx, referencesCreatedEvent)
				}

				if len(merged.Delta.Keywords) > 0 {
					keywordsCreatedEvent := &pb.Event{
						QaId: qaId,
						Event: &pb.Event_KeywordsCreated{
							KeywordsCreated: &pb.KeywordsCreated{
								Keywords: merged.Delta.Keywords,
							},
						},
						CreatedAt: timestamppb.New(time.Now()),
					}
					_ = s.EventLogService.WriteEvent(ctx, keywordsCreatedEvent)
				}

				if len(merged.Delta.RelatedQueries) > 0 {
					relatedQueriesCreatedEvent := &pb.Event{
						QaId: qaId,
						Event: &pb.Event_RelatedQueriesCreated{
							RelatedQueriesCreated: &pb.RelatedQueriesCreated{
								RelatedQueries: merged.Delta.RelatedQueries,
							},
						},
						CreatedAt: timestamppb.New(time.Now()),
					}
					_ = s.EventLogService.WriteEvent(ctx, relatedQueriesCreatedEvent)
				}
			}
		}

	}()

	return completionMultiChannel, nil
}

func (s *CompletionService) CreateChatCompletionWithChatHistory(ctx context.Context, param *CreateChatCompletionParameter) (chan *CreateChatCompletionResult, error) {
	timeoutctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	qas, err := s.ChatService.ListChatQAs(timeoutctx, param.Session, param.ChatID)
	if err != nil {
		slog.Error("error getting chat qas", "error", err.Error())
		qas = make([]*model.QA, 0)
	}

	messages := make([]*model.Message, 0)

	for i := len(qas) - 1; i >= 0; i-- {
		// start iteration from latest to use only latest reference

		qa := qas[i]

		if qa.Answer == "" || qa.Question == "" {
			continue
		}

		// payload flow : question -> reference -> answer
		// should be reversed to append to payloads

		messages = append(messages, &model.Message{
			Content: qa.Answer,
			Role:    "assistant",
		})

		if qa.References != nil {
			b, err := json.Marshal(qa.References)
			if err != nil {
				return nil, err
			}

			functionName := "search"

			messages = append(messages, &model.Message{
				Content: string(b),
				Role:    "function",
				Name:    functionName,
			})

		}

		messages = append(messages, &model.Message{
			Role:    "user",
			Content: qa.Question,
		})
	}

	slices.Reverse(messages)

	param.Messages = append(messages, param.Messages...)

	return s.CreateChatCompletion(ctx, param)
}

func (s *CompletionService) findLastUserMessage(messages []*model.Message) *model.Message {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			return messages[i]
		}
	}
	return nil
}

func (s *CompletionService) CreateChatCompletion(context context.Context, param *CreateChatCompletionParameter) (chan *CreateChatCompletionResult, error) {
	messages := param.Messages

	qaId := uuid.New().String()
	chatId := param.ChatID
	sessionId := param.Session
	jobGroup := param.JobGroup
	articleProvider := param.Provider
	if chatId == "" || sessionId == "" {
		return nil, fmt.Errorf("chat id, session id is empty")
	}

	if sessionId == "error-test" {
		return nil, fmt.Errorf("error test")
	}

	// send QuestionCreated event
	lastUserMessage := s.findLastUserMessage(messages)
	if lastUserMessage == nil {
		return nil, fmt.Errorf("there is no user message")
	}
	questionCreatedEvent := &pb.Event{
		QaId:      qaId,
		CreatedAt: timestamppb.New(time.Now()),
		Event: &pb.Event_QuestionCreated{
			QuestionCreated: &pb.QuestionCreated{
				ChatId:    chatId,
				SessionId: sessionId,
				JobGroup:  jobGroup,
				Question:  lastUserMessage.Content,
			},
		},
	}
	err := s.EventLogService.WriteEvent(context, questionCreatedEvent)
	if err != nil {
		return nil, err
	}

	completionChannel := make(chan *CreateChatCompletionResult, 10)

	go func() {
		defer close(completionChannel)

		if sessionId == "stream-error-test" {
			completionChannel <- &CreateChatCompletionResult{
				Error: fmt.Errorf("stream error test"),
			}
			return
		}

		lastUserPayloads := messages[len(messages)-1]
		if lastUserPayloads.Role != "user" {
			completionChannel <- &CreateChatCompletionResult{
				Error: fmt.Errorf("last payload role should be user"),
			}
			return
		}
		if len(strings.Split(lastUserPayloads.Content, " ")) == 1 {
			completion := &model.Completion{
				Object:  "chat.completion",
				Id:      qaId,
				Created: int(time.Now().Unix()),
				Delta: &model.CompletionDelta{
					Content: "더 구체적인 질문을 해 주세요",
				},
			}
			completionChannel <- &CreateChatCompletionResult{
				Completion: completion,
			}
			completionChannel <- &CreateChatCompletionResult{
				Done: true,
			}
			return
		}
		tokenCount := 0

		completionRequest := model.CreateChatCompletionRequest{
			Messages: messages,
			Provider: articleProvider,
		}
		requestBody, err := json.Marshal(completionRequest)
		if err != nil {
			completionChannel <- &CreateChatCompletionResult{
				Error: err,
			}
			return
		}
		stream, err := request.CreateChatStream(context, s.client, s.convEngineEndpoint, requestBody)
		if err != nil {
			completionChannel <- &CreateChatCompletionResult{
				Error: err,
			}
			return
		}
		defer stream.Close()

		for {
			select {
			case <-context.Done():
				completionChannel <- &CreateChatCompletionResult{
					Error: context.Err(),
				}
				return
			default:
				resp, err := stream.Recv()
				if err != nil && err != io.EOF {
					completionChannel <- &CreateChatCompletionResult{
						Error: err,
					}

					return
				}
				if err == io.EOF {
					return
				}

				completion := &model.Completion{
					Object:  "chat.completion",
					Id:      qaId,
					Created: int(time.Now().Unix()),
					Delta:   resp.Delta,
				}

				completionChannel <- &CreateChatCompletionResult{
					Completion: completion,
				}

				// count token
				tokenCount += resp.TokenUsage
				_ = s.EventLogService.WriteEvent(context, &pb.Event{
					QaId: qaId,
					Event: &pb.Event_TokenCountUpdated{
						TokenCountUpdated: &pb.TokenCountUpdated{
							TokenCount: int32(tokenCount),
						},
					},
					CreatedAt: timestamppb.New(time.Now()),
				})

				// write event
				merged := stream.ReadUntilNow()

				if merged.Delta.Content != "" {
					answerUpdatedEvent := &pb.Event{
						QaId: qaId,
						Event: &pb.Event_AnswerUpdated{
							AnswerUpdated: &pb.AnswerUpdated{
								Answer: merged.Delta.Content,
								// llm model, llm provider 없는데 ?
							},
						},
						CreatedAt: timestamppb.New(time.Now()),
					}
					_ = s.EventLogService.WriteEvent(context, answerUpdatedEvent)
				}

				if len(merged.Delta.References) > 0 {
					referencesCreatedEvent := &pb.Event{
						QaId: qaId,
						Event: &pb.Event_ReferencesCreated{
							ReferencesCreated: &pb.ReferencesCreated{
								References: model.FromModelReferencesToProtoReferences(merged.Delta.References),
							},
						},
						CreatedAt: timestamppb.New(time.Now()),
					}
					_ = s.EventLogService.WriteEvent(context, referencesCreatedEvent)
				}

				if len(merged.Delta.Keywords) > 0 {
					keywordsCreatedEvent := &pb.Event{
						QaId: qaId,
						Event: &pb.Event_KeywordsCreated{
							KeywordsCreated: &pb.KeywordsCreated{
								Keywords: merged.Delta.Keywords,
							},
						},
						CreatedAt: timestamppb.New(time.Now()),
					}
					_ = s.EventLogService.WriteEvent(context, keywordsCreatedEvent)
				}

				if len(merged.Delta.RelatedQueries) > 0 {
					relatedQueriesCreatedEvent := &pb.Event{
						QaId: qaId,
						Event: &pb.Event_RelatedQueriesCreated{
							RelatedQueriesCreated: &pb.RelatedQueriesCreated{
								RelatedQueries: merged.Delta.RelatedQueries,
							},
						},
						CreatedAt: timestamppb.New(time.Now()),
					}
					_ = s.EventLogService.WriteEvent(context, relatedQueriesCreatedEvent)
				}
			}
		}

	}()

	return completionChannel, nil
}
