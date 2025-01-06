package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"bigkinds.or.kr/conversation/internal/llmclient"
	"bigkinds.or.kr/conversation/internal/token_counter"
	model "bigkinds.or.kr/conversation/model"
	"bigkinds.or.kr/conversation/service/function"
	"bigkinds.or.kr/pkg/chat/v2"
	"bigkinds.or.kr/pkg/chat/v2/gpt"
	"bigkinds.or.kr/pkg/utils"
	"github.com/google/uuid"
)

type CompletionMultiService struct {
	PromptService   *PromptService
	FunctionService *function.FunctionService

	client       *http.Client
	tokenCounter *token_counter.TokenCounter
}

type CreateChatCompletionMultiResult struct {
	Completion *model.Completion `json:"completion"`
	Done       bool              `json:"done"`
	Error      error             `json:"error"`
}

type CreateChatCompletionMultiParameter struct {
	Payloads []*chat.ChatPayload `json:"payloads"`
	Provider string              `json:"provider"`
}

func NewCompletionMultiService(functionService *function.FunctionService, tokenCounter *token_counter.TokenCounter) *CompletionMultiService {
	client := &http.Client{
		Transport: &http.Transport{
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}

	return &CompletionMultiService{
		FunctionService: functionService,
		client:          client,
		tokenCounter:    tokenCounter,
	}
}

func (s *CompletionMultiService) createKeywordsRelatedQueries(ctx context.Context, ch chan *CreateChatCompletionMultiResult, id, provider, modelName, sargs string) {
	keywordsRelatedQueriesMode := os.Getenv("KEYWORDS_RELATED_QUERIES_MODE")
	switch keywordsRelatedQueriesMode {
	case "llm":
		keywordsRelatedQueriesService := &KeywordsRelatedQueriesService{
			tokenCounter: s.tokenCounter,
		}
		var (
			keywordsRelatedQueries *model.KeywordsRelatedQueries
			tokens                 int
			err                    error
		)
		switch provider {
		case "upstage":
			keywordsRelatedQueries, tokens, err = keywordsRelatedQueriesService.GenerateKeywordsRelatedQueriesSolar(ctx, modelName, sargs)
		case "openai":
			keywordsRelatedQueries, tokens, err = keywordsRelatedQueriesService.GenerateKeywordsRelatedQueriesGpt(ctx, provider, modelName, sargs)
		}
		if err != nil {
			slog.Error("error getting keywords related queries", "error", err.Error())
			keywordsRelatedQueries = &model.KeywordsRelatedQueries{
				Keywords:       []string{},
				RelatedQueries: []string{},
			}
		}
		concatenatedKeywords := strings.Join(keywordsRelatedQueries.Keywords, " ")
		ch <- &CreateChatCompletionMultiResult{
			Completion: &model.Completion{
				Object:  "chat.completion",
				Id:      id,
				Created: int(time.Now().Unix()),
				Delta: model.CompletionDelta{
					Keywords:       []string{concatenatedKeywords},
					RelatedQueries: keywordsRelatedQueries.RelatedQueries,
				},
				TokenUsage: tokens,
			},
		}
	case "mock":
		ch <- &CreateChatCompletionMultiResult{
			Completion: &model.Completion{
				Object:  "chat.completion",
				Id:      id,
				Created: int(time.Now().Unix()),
				Delta: model.CompletionDelta{
					Keywords:       []string{"키워드1", "키워드2"},
					RelatedQueries: []string{"연관 질문1", "연관 질문2"},
				},
				TokenUsage: 0,
			},
		}
	default:
		slog.Info("keywords related queries is disabled. Send empty keywords and related queries")
		ch <- &CreateChatCompletionMultiResult{
			Completion: &model.Completion{
				Object:  "chat.completion",
				Id:      id,
				Created: int(time.Now().Unix()),
				Delta: model.CompletionDelta{
					Keywords:       []string{},
					RelatedQueries: []string{},
				},
				TokenUsage: 0,
			},
		}
	}

}

func (s *CompletionMultiService) createInitialPayloads(currentTime utils.CurrentTime, payloads []*chat.ChatPayload) ([]*chat.ChatPayload, error) {
	prompt := s.PromptService.GetChatPrompt(currentTime.Time.Format("2006-01-02-T15:04:05-07:00"))
	systemPayload := &chat.ChatPayload{
		Content: prompt,
		Role:    "system",
	}
	payloads = append([]*chat.ChatPayload{systemPayload}, payloads...)
	return payloads, nil
}

func (s *CompletionMultiService) findLastUserMessage(payloads []*chat.ChatPayload) *chat.ChatPayload {
	for i := len(payloads) - 1; i >= 0; i-- {
		if payloads[i].Role == "user" {
			return payloads[i]
		}
	}

	return nil
}

// func (s *CompletionMultiService) findHistoryUserMessage(payloads []*chat.ChatPayload) []*chat.ChatPayload {
// 	res := []*chat.ChatPayload{}
// 	for i := 0; i < len(payloads)-1; i++ {
// 		res = append(res, payloads[i])
// 	}

// 	return res
// }

type CreateChatResponse struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Choices []CreateChatResponseChoice
	Usage   CreateChatResponseUsage
}

type CreateChatResponseChoice struct {
	Index        int `json:"index"`
	Message      CreateChatResponseChoiceMessage
	FinishReason string `json:"finish_reason"`
}

type CreateChatResponseChoiceToolCall struct {
	Id   string `json:"id"`
	Type string `json:"type"`
}

type CreateChatResponseChoiceToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type CreateChatResponseChoiceMessage struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	ToolCalls []CreateChatResponseChoiceToolCall
}

type CreateChatResponseUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `jons:"total_tokens"`
}

func (s *CompletionMultiService) CreateChatCompletionMultiPrompt(ctx context.Context, param *CreateChatCompletionMultiParameter) (chan *CreateChatCompletionMultiResult, error) {
	// 사용자 메세지 추출
	slog.Info("===== ===== CreateChatCompletionMultiPrompt")

	payloads := param.Payloads
	slog.Info("===== ===== payloads")

	for i := 0; i < len(payloads); i++ {
		slog.Info(fmt.Sprintf("===== ===== i: %d", i))
		slog.Info(payloads[i].Role)
		slog.Info(payloads[i].Content)
	}

	userMessage := s.findLastUserMessage(payloads)
	// completionId := uuid.New().String()

	llmProvider := "upstage"

	if userMessage == nil {
		return nil, fmt.Errorf("there is no user message in parameter")
	}

	currentTime, err := utils.GetCurrentKSTTime()
	if err != nil {
		return nil, err
	}

	// system prompt를 추가
	var userPayloads []*chat.ChatPayload
	userPayloads = append(userPayloads, userMessage)

	reqPayloads, err := s.createInitialPayloads(currentTime, userPayloads)
	if err != nil {
		return nil, err
	}

	// function 추출
	// predict setting
	predictOpts := setPredictOpts()

	functions := s.FunctionService.ListFunctions(currentTime)
	if len(functions) > 0 {
		functionRawJson := make([]string, 0, len(functions))
		for _, gptFunction := range functions {
			definition := gptFunction.Definition()
			marshal, err := json.Marshal(definition)
			if err != nil {
				return nil, err
			}
			functionRawJson = append(functionRawJson, string(marshal))
		}
		predictOpts = append(predictOpts, chat.WithFunctions(functionRawJson))
	}
	predictOpts = append(predictOpts, chat.WithoutStream)

	// model setting
	solarMini := &model.CompletionLLM{
		Provider:  llmProvider,
		ModelName: "solar-mini",
	}

	// 사용자 메세지를 바탕으로 function 추출

	slog.Info("===== ===== before go func")

	// references := make([]model.Reference, 0)

	slog.Info("try to create response", "provider", solarMini.Provider, "model", solarMini.ModelName, "maxFallbackCount", solarMini.MaxFallbackCount)
	predictOpts = append(predictOpts, chat.WithModel(solarMini.ModelName))
	client, err := llmclient.NewClient(s.client, solarMini.Provider, solarMini.ModelName, 0, chat.WithStreamEnabled)
	if err != nil {
		return nil, err
	}

	predictOpts = append(predictOpts, chat.WithNilTollCall)
	predictOpts = append(predictOpts, chat.WithNilTollChoice)
	// 수정시 볼 곳
	resp, err := client.CreateChat(ctx, llmProvider, reqPayloads, predictOpts...)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	bodyString := string(bodyBytes)
	// data := strings.Split(bodyString, "data:")

	slog.Info(bodyString)

	var chatResponse CreateChatResponse
	keyIndex := 0
	err = json.Unmarshal(bodyBytes, &chatResponse)
	if err != nil {
		return nil, err
	}

	isToolCalls := chatResponse.Choices[0].Message.ToolCalls
	completionMultiResultChannel := make(chan *CreateChatCompletionMultiResult, 10)

	// 답변 생성은 sloar pro
	// function이 있는 없는 경우 처리
	solarPro := &model.CompletionLLM{
		Provider:  llmProvider,
		ModelName: "solar-pro",
	}

	go func() {
		defer close(completionMultiResultChannel)
		references := make([]model.Reference, 0)

		// functnion이 있는 경우 처리
		if isToolCalls != nil {
			slog.Info("is tool_calls")

		} else {
			slog.Info("is not tool_calls")
			// return nil, fmt.Errorf("not finished")

			slog.Info("try to create response", "provider", solarPro.Provider, "model", solarPro.ModelName)
			predictOpts = setPredictOpts()
			predictOpts = append(predictOpts, chat.WithModel(solarPro.ModelName))

			client, err := llmclient.NewClient(
				s.client,
				solarPro.Provider,
				solarPro.ModelName,
				keyIndex,
				chat.WithStreamEnabled,
			)

			if err != nil {
				completionMultiResultChannel <- &CreateChatCompletionMultiResult{
					Error: err,
				}
				return
			}

			// CreateChatStream
			for {
				predictOpts = append(predictOpts, chat.WithNilTollCall)
				predictOpts = append(predictOpts, chat)
			}

		}

	}()

	return nil, fmt.Errorf("not finished")
}

func (s *CompletionMultiService) CreateChatCompletionMulti(ctx context.Context, param *CreateChatCompletionMultiParameter) (chan *CreateChatCompletionMultiResult, error) {
	payloads := param.Payloads

	// 사용자 메세지와 과거 메세지로 분리
	lastPayloads := []*chat.ChatPayload{}
	lastUserMessage := s.findLastUserMessage(payloads)
	if lastUserMessage == nil {
		return nil, fmt.Errorf("there is no user message")
	}
	lastPayloads = append(lastPayloads, lastUserMessage)
	// historyPayloads := s.findHistoryUserMessage(payloads)

	completionId := uuid.New().String()
	predictOpts := setPredictOpts()
	articleProvider := param.Provider
	currentTime, err := utils.GetCurrentKSTTime()
	if err != nil {
		return nil, err
	}

	payloads, err = s.createInitialPayloads(currentTime, payloads)
	if err != nil {
		return nil, err
	}

	functions := s.FunctionService.ListFunctions(currentTime)
	if len(functions) > 0 {
		functionsRawJson := make([]string, 0, len(functions))
		for _, gptFunction := range functions {
			definition := gptFunction.Definition()
			marshal, err := json.Marshal(definition)
			if err != nil {
				return nil, err
			}
			functionsRawJson = append(functionsRawJson, string(marshal))
		}
		predictOpts = append(predictOpts, chat.WithFunctions(functionsRawJson))
	}

	completionMultiResultChannel := make(chan *CreateChatCompletionMultiResult, 10)

	modelIndex := 0
	keyIndex := 0
	var completionLLM *model.CompletionLLM
	go func() {
		defer close(completionMultiResultChannel)
		fallbackCount := 0

		references := make([]model.Reference, 0)

		var loopError error

		for {
			fallbackCount++

			if errors.Is(loopError, function.ErrSearchFunctionFailed) {
				slog.Error("search function failed")
				completionMultiResultChannel <- &CreateChatCompletionMultiResult{
					Error: loopError,
				}
				return
			} else if loopError != nil {
				if strings.Contains(loopError.Error(), "status code: 429") {
					slog.Error("rate limit exceeded")
					if keyIndex < len(strings.Split(os.Getenv("UPSTAGE_AZURE_ENDPOINT_KEY_MAP"), ",")) && completionLLM.Provider == "azure" {
						keyIndex++
					}
				} else {
					slog.Error("failed to create completion", "error", loopError.Error())
				}
			}

			loopError = nil
			completionLLM, err = GetCompletionLLM(modelIndex)
			if err != nil {
				completionMultiResultChannel <- &CreateChatCompletionMultiResult{
					Error: err,
				}
				return
			}
			if fallbackCount > completionLLM.MaxFallbackCount {
				slog.Info("fallback count exceed")
				modelIndex++
				keyIndex = 0
				fallbackCount = 0
				continue
			}

			slog.Info("try to create response", "provider", completionLLM.Provider, "model", completionLLM.ModelName, "maxFallbackCount", completionLLM.MaxFallbackCount, "fallbackCount", fallbackCount)
			predictOpts = append(predictOpts, chat.WithModel(completionLLM.ModelName))

			client, err := llmclient.NewClient(s.client, completionLLM.Provider, completionLLM.ModelName, keyIndex, chat.WithStreamEnabled)

			if err != nil {
				loopError = err
				continue
			}

			maxFunctionLoop := 0
			var keywordsRelatedQueriesWg sync.WaitGroup
			defer keywordsRelatedQueriesWg.Wait()

			sendKeywordsRelatedQueries := false
			for {
				predictOpts = append(predictOpts, chat.WithNilTollCall)
				predictOpts = append(predictOpts, chat.WithNilTollChoice)

				// 마지막 사용자의 메세지로만 function, tool 판단
				stream, err := client.CreateChatStream(ctx, completionLLM.Provider, lastPayloads, predictOpts...)
				if err != nil {
					loopError = err
					break
				}
				defer func(stream gpt.ChatStream) {
					err := stream.Close()
					if err != nil {
						slog.Error("failed to close stream", "error", err)
					}
				}(stream)

				tokenCount := 0
				for _, payload := range payloads {
					tokenCount += s.tokenCounter.CountTokens(payload.Content)
				}
				for _, gptFunction := range functions {
					tokenCount += s.tokenCounter.CountFunctionInputTokens(gptFunction.Definition())
				}
				completionMultiResultChannel <- &CreateChatCompletionMultiResult{
					Completion: &model.Completion{
						Object:     "chat.completion",
						Id:         completionId,
						Created:    int(time.Now().Unix()),
						TokenUsage: tokenCount,
					},
				}

				done := false

				for {
					if done {
						break
					}
					select {
					case <-ctx.Done():
						completionMultiResultChannel <- &CreateChatCompletionMultiResult{
							Error: ctx.Err(),
						}
						return
					default:
						resp, err := stream.Recv(completionLLM.Provider)
						if err != nil && err != io.EOF {
							loopError = err
							break
						}
						if err == io.EOF {
							done = true
							break
						}

						if (completionLLM.Provider == "upstage" && resp.ToolCalls == nil) || (completionLLM.Provider == "openai" && resp.FunctionCall == nil) {
							completion := &model.Completion{
								Object:  "chat.completion",
								Id:      completionId,
								Created: int(time.Now().Unix()),
								Delta: model.CompletionDelta{
									Content: resp.Payload.Content,
								},
								TokenUsage: s.tokenCounter.CountTokens(resp.Payload.Content),
							}

							completionMultiResultChannel <- &CreateChatCompletionMultiResult{
								Completion: completion,
							}
						}
					}
				}

				if loopError != nil {
					break
				}

				response := stream.ReadUntilNow()
				if len(response.Choices) == 0 {
					loopError = fmt.Errorf("no choices")
					break
				} else if response.Choices[0].FinishReason == "function_call" || response.Choices[0].FinishReason == "tool_calls" {
					callResponse := &gpt.ChatCompletionFunctionCallResp{}
					if response.Choices[0].FinishReason == "function_call" {
						callResponse.Name = response.Choices[0].Message.FunctionCall.Name
						callResponse.Arguments = response.Choices[0].Message.FunctionCall.Arguments
					} else {
						callResponse.Name = response.Choices[0].Message.ToolCalls[0].Function.Name
						callResponse.Arguments = response.Choices[0].Message.ToolCalls[0].Function.Arguments
					}
					tokenCount := s.tokenCounter.CountFunctionOutputTokens(callResponse.Arguments)
					completionMultiResultChannel <- &CreateChatCompletionMultiResult{
						Completion: &model.Completion{
							Object:     "chat.completion",
							Id:         completionId,
							Created:    int(time.Now().Unix()),
							TokenUsage: tokenCount,
						},
					}
					slog.Info("try to call tools ", "name", callResponse.Name, "arguments", callResponse.Arguments, "count", maxFunctionLoop)

					callResponse.Arguments, err = convertArgumentsToUTF8IfNot(callResponse.Arguments, lastUserMessage.Content)
					if err != nil {
						loopError = err
						break
					}
					extraArgs := &function.ExtraArgs{
						RawQuery:       lastUserMessage.Content,
						Provider:       articleProvider,
						Topk:           15,
						MaxChunkSize:   1000,
						MaxChunkNumber: 5,
					}
					callFunctionResponse, err := s.FunctionService.CallFunction(ctx, callResponse.Name, callResponse.Arguments, functions, extraArgs)
					if err != nil {
						loopError = err
						break
					}

					slog.Info("try to generate keywords and relatedQueries", "completionLLM.Provider: ", completionLLM.Provider, "completion.ModelName: ", completionLLM.ModelName)

					if callResponse.Name == "search" && len(callResponse.Arguments) > 0 && !sendKeywordsRelatedQueries {
						keywordsRelatedQueriesWg.Add(1)
						sendKeywordsRelatedQueries = true
						go func() {
							defer keywordsRelatedQueriesWg.Done()
							s.createKeywordsRelatedQueries(ctx, completionMultiResultChannel, completionId, completionLLM.Provider, completionLLM.ModelName, callResponse.Arguments)
						}()
					}

					functionCallMsg := &chat.ChatPayload{
						Role: "assistant",
						FunctionCall: &chat.ChatFunction{
							Name:      callResponse.Name,
							Arguments: callResponse.Arguments,
						},
					}

					payloadContent := string(callFunctionResponse)

					switch callResponse.Name {
					case "search":
						var items struct {
							Items []model.Reference `json:"items"`
						}

						err = json.Unmarshal(callFunctionResponse, &items)
						if err != nil {
							fmt.Printf("error unmarchalling references: %s\n", err.Error())
							break
						}

						contentRemovedReference := make([]model.Reference, len(items.Items))
						copy(contentRemovedReference, items.Items)
						for i := range items.Items {
							contentRemovedReference[i].Attributes.Content = ""
						}

						completionMultiResultChannel <- &CreateChatCompletionMultiResult{
							Completion: &model.Completion{
								Object:  "chat.completion",
								Id:      completionId,
								Created: int(time.Now().Unix()),
								Delta: model.CompletionDelta{
									References: contentRemovedReference,
								},
							},
						}

						references = append(references, items.Items...)

						var tmpItems = make([]struct {
							Title       string    `json:"title"`
							PublishedAt time.Time `json:"published_at"`
							Provider    string    `json:"provider"`
							Byline      string    `json:"byline"`
							Content     string    `json:"content,omitempty"`
							Index       int       `json:"index"`
						}, len(references))

						for i, reference := range references {
							tmpItems[i].Title = reference.Attributes.Title
							tmpItems[i].PublishedAt = reference.Attributes.PublishedAt
							tmpItems[i].Provider = reference.Attributes.Provider
							tmpItems[i].Byline = reference.Attributes.Byline
							tmpItems[i].Content = reference.Attributes.Content
							tmpItems[i].Index = 1 + i
						}

						rawRefsWithItemsIdCleared, err := json.Marshal(tmpItems)
						if err != nil {
							fmt.Printf("error on marshalling reference: %s\n", err.Error())
							break
						}

						// remove past search payloads
						for i := 0; i < len(payloads); i++ {
							payloads[i].FunctionCall = nil
							payloads[i].ToolCalls = nil
							if payloads[i].Name == nil {
								continue
							}
							if (payloads[i].Role == "function" && *payloads[i].Name == "search") || (payloads[i].Role == "assistant" && *payloads[i].Name == "search") {
								payloads = append(payloads[:i], payloads[i+1:]...)
								i--
							}
						}
						payloadContent = string(rawRefsWithItemsIdCleared)

					}
					role := ""
					switch completionLLM.Provider {
					case "openai":
						role = "function"
					case "upstage":
						role = "assistant"
					}
					payloads = append(payloads, functionCallMsg, &chat.ChatPayload{
						Content: payloadContent,
						Role:    role,
						Name:    &callResponse.Name,
					})

					if payloads[0].Role == "system" {
						payloads[0] = &chat.ChatPayload{
							Role:    "system",
							Content: s.PromptService.GetAfterFunctionCallPrompt(currentTime.Time.Format("2006-01-02T15:04:05-07:00")),
						}
					} else {
						loopError = errors.New("first payload must be system")
						break
					}

					predictOpts = append(predictOpts, chat.WithNilTollCall)
					predictOpts = append(predictOpts, chat.WithNilTollChoice)
					predictOpts = append(predictOpts, chat.WithNoFunctions)

				} else {
					keywordsRelatedQueriesWg.Wait()
					completionMultiResultChannel <- &CreateChatCompletionMultiResult{
						Done: true,
					}
					return
				}

			}

		}

	}()

	return completionMultiResultChannel, nil
}
