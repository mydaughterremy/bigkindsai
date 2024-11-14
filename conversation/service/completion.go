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

	"bigkinds.or.kr/pkg/chat/v2"
	"bigkinds.or.kr/pkg/chat/v2/gpt"
	"bigkinds.or.kr/pkg/utils"
	"github.com/google/uuid"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"

	"bigkinds.or.kr/conversation/internal/llmclient"
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
			ResponseHeaderTimeout: 10 * time.Second,
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

func setPredictOpts() []func(*gpt.GptPredictionOptions) {
	predictOpts := make([]func(*gpt.GptPredictionOptions), 0)
	predictOpts = append(predictOpts, gpt.WithStream)

	seed, ok := os.LookupEnv("UPSTAGE_OPENAI_SEED")
	if ok {
		seedInt, err := strconv.ParseInt(seed, 10, 64)
		if err == nil {
			predictOpts = append(predictOpts, gpt.WithSeed(seedInt))
		} else {
			log.Printf("invalid seed: %s", seed)
		}
	}

	var temperature float64 = 0
	temperatureString, ok := os.LookupEnv("UPSTAGE_OPENAI_TEMPERATURE")
	if ok {
		var err error
		temperature, err = strconv.ParseFloat(temperatureString, 32)
		if err != nil {
			log.Printf("invalid temperature from env: %s", temperatureString)
		}
	}
	predictOpts = append(predictOpts, gpt.WithTemperature(float32(temperature)))
	return predictOpts
}

func getModels() []string {
	modelList, ok := os.LookupEnv("UPSTAGE_LLM_MODEL")
	if !ok {
		modelList = "openai/gpt-3.5-turbo-1106/5"
	}
	models := strings.Split(modelList, ",")
	return models
}

func getCompletionLLM(modelIndex int) (*model.CompletionLLM, error) {
	models := getModels()
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

func (s *CompletionService) createInitialPayloads(currentTime utils.CurrentTime, payloads []*chat.ChatPayload) ([]*chat.ChatPayload, error) {
	currentTime, err := utils.GetCurrentKSTTime()
	if err != nil {
		return nil, err
	}
	prompt := s.PromptService.GetPrompt(currentTime.Time.Format("2006-01-02T15:04:05-07:00"))
	systemPayload := &chat.ChatPayload{
		Content: prompt,
		Role:    "system",
	}

	payloads = append([]*chat.ChatPayload{systemPayload}, payloads...)

	return payloads, nil
}

func (s *CompletionService) createKeywordsRelatedQueries(ctx context.Context, ch chan *CreateChatCompletionResult, id, provider, modelName, sargs string) {
	keywordsRelatedQueriesMode := os.Getenv("KEYWORDS_RELATED_QUERIES_MODE")
	switch keywordsRelatedQueriesMode {
	case "llm":
		keywordsRelatedQueriesService := &KeywordsRelatedQueriesService{
			tokenCounter: s.tokenCounter,
		}
		keywordsRelatedQueries, tokens, err := keywordsRelatedQueriesService.GenerateKeywordsRelatedQueries(ctx, provider, modelName, sargs)
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

func (s *CompletionService) CreateChatCompletion(ctx context.Context, param *CreateChatCompletionParameter) (chan *CreateChatCompletionResult, error) {
	payloads := param.Payloads

	completionId := uuid.New().String()
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
		for _, function := range functions {
			definition := function.Definition()
			b, err := json.Marshal(definition)
			if err != nil {
				return nil, err
			}
			functionRawJson = append(functionRawJson, string(b))
		}
		predictOpts = append(predictOpts, gpt.WithFunctions(functionRawJson))
	}

	ch := make(chan *CreateChatCompletionResult, 10)

	modelIndex := 0
	keyIndex := 0
	var completionLLM *model.CompletionLLM
	go func() {
		defer close(ch)

		fallbackCount := 0

		references := make([]model.Reference, 0)

		var loopError error

		for {
			fallbackCount++

			if loopError == function.ErrSearchFunctionFailed {
				slog.Error("search function failed")
				ch <- &CreateChatCompletionResult{
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

			completionLLM, err = getCompletionLLM(modelIndex)
			if err != nil {
				ch <- &CreateChatCompletionResult{
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

			predictOpts = append(predictOpts, gpt.WithModel(completionLLM.ModelName))

			client, err := llmclient.NewClient(
				s.client,
				completionLLM.Provider,
				completionLLM.ModelName,
				keyIndex,
				gpt.WithStreamEnabled,
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
				stream, err := client.CreateChatStream(ctx, payloads, predictOpts...)
				if err != nil {
					loopError = err
					break
				}
				defer stream.Close()

				// count input tokens
				tokenCount := 0
				for _, payload := range payloads {
					tokenCount += s.tokenCounter.CountTokens(payload.Content)
				}
				for _, function := range functions {
					tokenCount += s.tokenCounter.CountFunctionInputTokens(function.Definition())
				}
				ch <- &CreateChatCompletionResult{
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
						ch <- &CreateChatCompletionResult{
							Error: ctx.Err(),
						}
						return
					default:
						resp, err := stream.Recv()
						if err != nil && err != io.EOF {
							loopError = err
							break
						}
						if err == io.EOF {
							done = true
							break
						}

						if resp.FunctionCall == nil {
							completion := &model.Completion{
								Object:  "chat.completion",
								Id:      completionId,
								Created: int(time.Now().Unix()),
								Delta: model.CompletionDelta{
									Content: resp.Payload.Content,
								},
								TokenUsage: s.tokenCounter.CountTokens(resp.Payload.Content),
							}

							ch <- &CreateChatCompletionResult{
								Completion: completion,
							}

						}
					}
				}

				if loopError != nil {
					break
				}
				resp := stream.ReadUntilNow()

				if len(resp.Choices) == 0 {
					loopError = fmt.Errorf("no choices")
					break
				} else if resp.Choices[0].FinishReason == "function_call" {
					f := resp.Choices[0].Message.FunctionCall
					tokenCount := s.tokenCounter.CountFunctionOutputTokens(f.Arguments)
					ch <- &CreateChatCompletionResult{
						Completion: &model.Completion{
							Object:     "chat.completion",
							Id:         completionId,
							Created:    int(time.Now().Unix()),
							TokenUsage: tokenCount,
						},
					}

					slog.Info("try to call function", "name", resp.Choices[0].Message.FunctionCall.Name, "arguments", resp.Choices[0].Message.FunctionCall.Arguments, "count", maxFunctionLoop)
					lastUserMessage := s.findLastUserMessage(payloads)
					if lastUserMessage == nil {
						loopError = fmt.Errorf("there is no user message")
						break
					}

					f.Arguments, err = convertArgumentsToUTF8IfNot(f.Arguments, lastUserMessage.Content)
					if err != nil {
						loopError = err
						break
					}
					extraArgs := &function.ExtraArgs{
						RawQuery: lastUserMessage.Content,
						Provider: articleProvider,
					}
					b, err := s.FunctionService.CallFunction(ctx, f.Name, f.Arguments, functions, extraArgs)
					if err != nil {
						if err == function.IndependentCallError {
							// set payloads
							if payloads[0].Role == "system" {
								payloads = payloads[1:]
							} else {
								loopError = errors.New("first payload should be system")
								break
							}
							predictOpts = append(predictOpts, gpt.WithFunctions([]string{}))
							continue
						} else {
							if maxFunctionLoop < 3 {
								maxFunctionLoop++
								continue
							}
							loopError = err
							break
						}
					}

					// use go routine to call keywords related queries
					slog.Info("try to generate keywords and relatedQueries", "provider", completionLLM.Provider, "model", completionLLM.ModelName)
					if f.Name == "search" && len(f.Arguments) > 0 && !sendKeywordsRelatedQueries {
						keywordsRelatedQueriesWg.Add(1)
						sendKeywordsRelatedQueries = true
						go func() {
							defer keywordsRelatedQueriesWg.Done()
							s.createKeywordsRelatedQueries(ctx, ch, completionId, completionLLM.Provider, completionLLM.ModelName, f.Arguments)
						}()
					}

					functionCallMsg := &chat.ChatPayload{
						Role: "assistant",
						FunctionCall: &chat.ChatFunction{
							Name:      f.Name,
							Arguments: f.Arguments,
						},
					}

					// send result
					payloadContent := string(b)
					switch f.Name {
					case "search":
						var items struct {
							Items []model.Reference `json:"items"`
						}
						err = json.Unmarshal(b, &items)
						if err != nil {
							fmt.Printf("error unmarshalling references: %s\n", err.Error())
							break
						}

						contentRemovedReference := make([]model.Reference, len(items.Items))
						copy(contentRemovedReference, items.Items)
						for i := range items.Items {
							contentRemovedReference[i].Attributes.Content = ""
						}

						ch <- &CreateChatCompletionResult{ // should not be accumulated
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
							if payloads[i].Role == "function" && *payloads[i].Name == "search" {
								payloads = append(payloads[:i], payloads[i+1:]...)
								i--
							}
						}

						payloadContent = string(rawRefsWithItemsIdCleared)
					}

					payloads = append(payloads, functionCallMsg, &chat.ChatPayload{
						Content: payloadContent,
						Role:    "function",
						Name:    &f.Name,
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
					// wait for keywords and relatedQueries before done
					keywordsRelatedQueriesWg.Wait()
					ch <- &CreateChatCompletionResult{
						Done: true,
					}
					return
				}
			}
		}
	}()

	return ch, nil
}
