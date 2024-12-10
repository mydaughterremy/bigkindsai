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
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"bigkinds.or.kr/conversation/internal/llmclient"
	"bigkinds.or.kr/pkg/chat/v2/gpt"

	"bigkinds.or.kr/pkg/chat/v2"
	"bigkinds.or.kr/pkg/utils"
	"github.com/google/uuid"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"

	"bigkinds.or.kr/conversation/internal/token_counter"
	model "bigkinds.or.kr/conversation/model"
	"bigkinds.or.kr/conversation/service/function"
)

type CompletionService struct {
	PromptService   *PromptService
	FunctionService *function.FunctionService

	client       *http.Client
	tokenCounter *token_counter.TokenCounter
}

type CreateChatCompletionParameter struct {
	Payloads []*chat.ChatPayload `json:"payloads"`
	Provider string              `json:"provider"`
}

type CreateChatCompletionResult struct {
	Completion *model.Completion `json:"completion"`
	Done       bool              `json:"done"`
	Error      error             `json:"error"`
}

func NewCompletionService(
	functionService *function.FunctionService,
	tokenCounter *token_counter.TokenCounter,
) *CompletionService {

	client := &http.Client{
		Transport: &http.Transport{
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}

	service := &CompletionService{
		FunctionService: functionService,
		client:          client,
		tokenCounter:    tokenCounter,
	}

	return service
}

func (s *CompletionService) findLastUserMessage(payloads []*chat.ChatPayload) *chat.ChatPayload {
	for i := len(payloads) - 1; i >= 0; i-- {
		if payloads[i].Role == "user" {
			return payloads[i]
		}
	}
	return nil
}

// 수정
func setPredictOpts() []func(*chat.GptPredictionOptions) {
	predictOpts := make([]func(*chat.GptPredictionOptions), 0)
	predictOpts = append(predictOpts, chat.WithStream)

	seed, ok := os.LookupEnv("UPSTAGE_LLM_SEED")
	if ok {
		seedInt, err := strconv.ParseInt(seed, 10, 64)
		if err == nil {
			predictOpts = append(predictOpts, chat.WithSeed(seedInt))
		} else {
			log.Printf("invalid seed: %s", seed)
		}
	}

	var temperature float64 = 0
	temperatureString, ok := os.LookupEnv("UPSTAGE_LLM_TEMPERATURE")
	if ok {
		var err error
		temperature, err = strconv.ParseFloat(temperatureString, 32)
		if err != nil {
			log.Printf("invalid temperature from env: %s", temperatureString)
		}
	}
	predictOpts = append(predictOpts, chat.WithTemperature(float32(temperature)))
	return predictOpts
}

// GetModels 수정
//
//	func GetModels() []string {
//		modelList, ok := os.LookupEnv("UPSTAGE_LLM_MODEL")
//		if !ok {
//			modelList = "openai/gpt-3.5-turbo-1106/5"
//		}
//		models := strings.Split(modelList, ",")
//		return models
//	}
//

// 수정
// provider | upstage
// model | solar-mini
func GetCompletionLLM(modelIndex int) (*model.CompletionLLM, error) {
	models := chat.GetModels()
	if modelIndex >= len(models) {
		return nil, fmt.Errorf("all fallback failed")
	}
	splited := strings.Split(models[modelIndex], "/")
	provider := splited[0]
	modelName := splited[1]

	maxFallbackCount := 3
	if len(splited) > 2 {
		var err error
		maxFallbackCount, err = strconv.Atoi(splited[2])
		if err != nil {
			return nil, fmt.Errorf("error parsing maxFallbackCount: %s", err.Error())
		}
	}

	return &model.CompletionLLM{
		Provider:         provider,
		ModelName:        modelName,
		MaxFallbackCount: maxFallbackCount,
	}, nil
}

// 수정
func (s *CompletionService) createInitialPayloads(currentTime utils.CurrentTime, payloads []*chat.ChatPayload) ([]*chat.ChatPayload, error) {
	currentTime, err := utils.GetCurrentKSTTime()
	if err != nil {
		return nil, err
	}
	prompt := s.PromptService.GetChatPrompt(currentTime.Time.Format("2006-01-02T15:04:05-07:00"))

	models := chat.GetModels()
	provider := models[0]
	systemPayload := &chat.ChatPayload{
		Content: prompt,
		Role:    "system",
	}
	if provider == "upstage" {
		systemPayload.Name = new(string)
		systemPayload.FunctionCall = &chat.ChatFunction{}
		systemPayload.ToolCalls = make([]*chat.ChatTool, 0)
	}
	payloads = append([]*chat.ChatPayload{systemPayload}, payloads...)

	return payloads, nil
}

// 수정
func (s *CompletionService) createKeywordsRelatedQueries(context context.Context, ch chan *CreateChatCompletionResult, id, provider, modelName, sargs string) {
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
			keywordsRelatedQueries, tokens, err = keywordsRelatedQueriesService.GenerateKeywordsRelatedQueriesSolar(context, modelName, sargs)
		case "openai":
			keywordsRelatedQueries, tokens, err = keywordsRelatedQueriesService.GenerateKeywordsRelatedQueriesGpt(context, provider, modelName, sargs)
		}

		if err != nil {
			slog.Error("error getting keywords related queries", "error", err.Error())
			keywordsRelatedQueries = &model.KeywordsRelatedQueries{
				Keywords:       []string{},
				RelatedQueries: []string{},
			}
		}
		concatenatedKeywords := strings.Join(keywordsRelatedQueries.Keywords, " ")
		ch <- &CreateChatCompletionResult{
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
		ch <- &CreateChatCompletionResult{
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
		ch <- &CreateChatCompletionResult{
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

func setNewStandaloneQuery(sargs string, newStandaloneQuery string) (string, error) {
	var margs map[string]interface{}
	err := json.Unmarshal([]byte(sargs), &margs)
	if err != nil {
		return "", err
	}
	_, ok := margs["standalone_query"]
	if ok {
		margs["standalone_query"] = newStandaloneQuery
		arguments, err := json.Marshal(margs)
		if err != nil {
			return "", err
		}
		return string(arguments), nil
	} else {
		return "", fmt.Errorf("there is no standalone_query in arguments")
	}
}

func convertArgumentsToUTF8IfNot(sargs string, newStandaloneQuery string) (string, error) {
	if utf8.ValidString(sargs) {
		return sargs, nil
	} else {
		convertedRunes := make([]rune, 0, len(sargs))

		invalidSargs := sargs
		for len(invalidSargs) > 0 {
			r, size := utf8.DecodeRuneInString(invalidSargs)
			if r == utf8.RuneError && size == 1 {
				convertedRunes = append(convertedRunes, rune(invalidSargs[0]))
			} else {
				convertedRunes = append(convertedRunes, r)
			}
			invalidSargs = invalidSargs[size:]
		}
		latin1Encoder := charmap.ISO8859_1.NewEncoder()
		latinEncodedBytes, err := latin1Encoder.Bytes([]byte(string(convertedRunes)))
		if err != nil {
			return setNewStandaloneQuery(sargs, newStandaloneQuery)
		}
		utf8Decoder := unicode.UTF8.NewDecoder()
		utf8DecodedString, err := utf8Decoder.String(string(latinEncodedBytes))
		if err != nil {
			return setNewStandaloneQuery(sargs, newStandaloneQuery)
		}
		return string(utf8DecodedString), nil
	}
}

func (s *CompletionService) CreateChatCompletion(context context.Context, param *CreateChatCompletionParameter) (chan *CreateChatCompletionResult, error) {
	payloads := param.Payloads

	completionId := uuid.New().String()
	// 수정 predict 옵션
	predictOpts := setPredictOpts()
	articleProvider := param.Provider

	currentTime, err := utils.GetCurrentKSTTime()
	if err != nil {
		return nil, err
	}

	// set initial payloads
	payloads, err = s.createInitialPayloads(currentTime, payloads)
	if err != nil {
		return nil, err
	}

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
	completionResultChannel := make(chan *CreateChatCompletionResult, 10)

	modelIndex := 0
	keyIndex := 0
	var completionLLM *model.CompletionLLM
	go func() {
		defer close(completionResultChannel)

		fallbackCount := 0

		references := make([]model.Reference, 0)

		var loopError error

		for {
			fallbackCount++

			if errors.Is(loopError, function.ErrSearchFunctionFailed) {
				slog.Error("search function failed")
				completionResultChannel <- &CreateChatCompletionResult{
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
				completionResultChannel <- &CreateChatCompletionResult{
					Error: err,
				}
				return
			}
			if fallbackCount > completionLLM.MaxFallbackCount {
				slog.Info("fallback count exceeded")
				modelIndex++
				keyIndex = 0
				fallbackCount = 0
				continue
			}

			slog.Info("try to create response", "provider", completionLLM.Provider, "model", completionLLM.ModelName, "maxFallbackCount", completionLLM.MaxFallbackCount, "fallbackCount", fallbackCount)

			predictOpts = append(predictOpts, chat.WithModel(completionLLM.ModelName))

			// 클라이언트 수정
			client, err := llmclient.NewClient(
				s.client,
				completionLLM.Provider,
				completionLLM.ModelName,
				keyIndex,
				chat.WithStreamEnabled,
			)

			if err != nil {
				loopError = err
				continue
			}

			maxFunctionLoop := 0
			var keywordsRelatedQueriesWg sync.WaitGroup
			defer keywordsRelatedQueriesWg.Wait()

			sendKeywordsRelatedQueries := false
			for {
				// 수정시 볼 곳
				stream, err := client.CreateChatStream(context, completionLLM.Provider, payloads, predictOpts...)
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

				// count input tokens
				tokenCount := 0
				for _, payload := range payloads {
					tokenCount += s.tokenCounter.CountTokens(payload.Content)
				}
				for _, gptFunction := range functions {
					tokenCount += s.tokenCounter.CountFunctionInputTokens(gptFunction.Definition())
				}
				completionResultChannel <- &CreateChatCompletionResult{
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
					case <-context.Done():
						completionResultChannel <- &CreateChatCompletionResult{
							Error: context.Err(),
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
					// chat gpt 의 경우 function call
					// solar 의 경우 tool call 만 동작함
						if (completionLLM.Provider == "upstage" && resp.ToolCalls == nil) ||
							(completionLLM.Provider == "openai" && resp.FunctionCall == nil) {
							completion := &model.Completion{
								Object:  "chat.completion",
								Id:      completionId,
								Created: int(time.Now().Unix()),
								Delta: model.CompletionDelta{
									Content: resp.Payload.Content,
								},
								TokenUsage: s.tokenCounter.CountTokens(resp.Payload.Content),
							}

							completionResultChannel <- &CreateChatCompletionResult{
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
					// chat gpt 의 경우 function call
					// solar 의 경우 tool call 만 동작함
					callResponse := &gpt.ChatCompletionFunctionCallResp{}
					if response.Choices[0].FinishReason == "function_call" {
						callResponse.Name = response.Choices[0].Message.FunctionCall.Name
						callResponse.Arguments = response.Choices[0].Message.FunctionCall.Arguments
					} else {
						callResponse.Name = response.Choices[0].Message.ToolCalls[0].Function.Name
						callResponse.Arguments = response.Choices[0].Message.ToolCalls[0].Function.Arguments
					}
					tokenCount := s.tokenCounter.CountFunctionOutputTokens(callResponse.Arguments)
					completionResultChannel <- &CreateChatCompletionResult{
						Completion: &model.Completion{
							Object:     "chat.completion",
							Id:         completionId,
							Created:    int(time.Now().Unix()),
							TokenUsage: tokenCount,
						},
					}
					slog.Info("try to call tools ", "name", callResponse.Name, "arguments", callResponse.Arguments, "count", maxFunctionLoop)
					lastUserMessage := s.findLastUserMessage(payloads)
					if lastUserMessage == nil {
						loopError = fmt.Errorf("there is no user message")
						break
					}

					callResponse.Arguments, err = convertArgumentsToUTF8IfNot(callResponse.Arguments, lastUserMessage.Content)
					if err != nil {
						loopError = err
						break
					}
					extraArgs := &function.ExtraArgs{
						RawQuery: lastUserMessage.Content,
						Provider: articleProvider,
						Topk: 15,
						MaxChunkSize: 1000,
						MaxChunkNumber: 5,
					}
					callFunctionResponse, err := s.FunctionService.CallFunction(context, callResponse.Name, callResponse.Arguments, functions, extraArgs)

					slog.Info("try to generate keywords and relatedQueries", "provider", completionLLM.Provider, "model", completionLLM.ModelName)

					// 수정 query 가져오기
					if callResponse.Name == "search" && len(callResponse.Arguments) > 0 && !sendKeywordsRelatedQueries {
						keywordsRelatedQueriesWg.Add(1)
						sendKeywordsRelatedQueries = true
						go func() {
							defer keywordsRelatedQueriesWg.Done()
							// 수정
							s.createKeywordsRelatedQueries(context, completionResultChannel, completionId, completionLLM.Provider, completionLLM.ModelName, callResponse.Arguments)
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
					// send result
					switch callResponse.Name {
					case "search":
						var items struct {
							Items []model.Reference `json:"items"`
						}
						err = json.Unmarshal(callFunctionResponse, &items)
						if err != nil {
							fmt.Printf("error unmarshalling references: %s\n", err.Error())
							break
						}

						contentRemovedReference := make([]model.Reference, len(items.Items))
						copy(contentRemovedReference, items.Items)
						for i := range items.Items {
							contentRemovedReference[i].Attributes.Content = ""
						}

						completionResultChannel <- &CreateChatCompletionResult{ // should not be accumulated
							Completion: &model.Completion{
								Object:  "chat.completion",
								Id:      completionId,
								Created: int(time.Now().Unix()),
								Delta: model.CompletionDelta{
									References: contentRemovedReference,
								},
							},
						}

						references = append(references, items.Items...) // should be accumulated

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
						loopError = errors.New("first payload should be system")
						break
					}
				} else {
					keywordsRelatedQueriesWg.Wait()
					completionResultChannel <- &CreateChatCompletionResult{
						Done: true,
					}
					return
				}
			}
		}
	}()

	return completionResultChannel, nil
}
