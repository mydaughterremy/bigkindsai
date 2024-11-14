package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"

	"bigkinds.or.kr/conversation/internal/token_counter"
	"bigkinds.or.kr/conversation/model"
	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
)

const maxRetries = 3

type KeywordsRelatedQueriesService struct {
	tokenCounter *token_counter.TokenCounter
}

func getKeywordQuery(sargs string) (string, error) {
	var margs map[string]interface{}
	err := json.Unmarshal([]byte(sargs), &margs)
	if err != nil {
		return "", err
	}
	keywordQuery, ok := margs["standalone_query"].(string)
	if !ok {
		return "", fmt.Errorf("invalid arguments: %s", sargs)
	}
	return keywordQuery, nil
}

func (k *KeywordsRelatedQueriesService) GenerateKeywordsRelatedQueries(ctx context.Context, provider, modelName, arguments string) (*model.KeywordsRelatedQueries, int, error) {
	client, err := NewLLMClient(provider, modelName)
	if err != nil {
		return nil, 0, err
	}
	chatCompletionOptions := newChatCompletionOptions()

	// set seed
	seed, ok := os.LookupEnv("UPSTAGE_OPENAI_SEED")
	if ok {
		seedInt, err := strconv.ParseInt(seed, 10, 64)
		if err == nil {
			chatCompletionOptions.SetSeed(seedInt)
		} else {
			return nil, 0, fmt.Errorf("invalid seed: %s", seed)
		}
	}

	// set temperature
	var temperature float32 = 0
	temperatureString, ok := os.LookupEnv("UPSTAGE_OPENAI_TEMPERATURE")
	if ok {
		var err error
		temperatureFloat64, err := strconv.ParseFloat(temperatureString, 32)
		if err != nil {
			log.Printf("invalid temperature from env: %s", temperatureString)
		}
		temperature = float32(temperatureFloat64)
	}
	chatCompletionOptions.SetTemperature(temperature)

	// set response format
	chatCompletionOptions.SetJsonResponseFormat()

	// set deployment name
	chatCompletionOptions.SetDeploymentName(modelName)

	// get keyword query
	keywordQuery, err := getKeywordQuery(arguments)
	if err != nil {
		return nil, 0, err
	}
	// set messages
	messages := []azopenai.ChatRequestMessageClassification{
		&azopenai.ChatRequestUserMessage{Content: azopenai.NewChatRequestUserMessageContent(getPrompt(keywordQuery))},
	}
	chatCompletionOptions.SetMessages(messages)
	tokenCount := 0
	asd, err := json.Marshal(chatCompletionOptions)
	fmt.Println(string(asd))

	for i := 0; i < maxRetries; i++ {
		slog.Info("try to generate keywords and relatedQueries", "keywordQuery", keywordQuery, "count", i)
		var keywordsRelatedQueries model.KeywordsRelatedQueries
		resp, err := client.GetChatCompletions(ctx, chatCompletionOptions.options, nil)
		if err != nil {
			return nil, tokenCount, err
		}
		tokenCount += k.tokenCounter.CountTokens(getPrompt(keywordQuery))
		if len(resp.Choices) == 0 {
			return nil, tokenCount, errors.New("no choices in response")
		}

		for _, choice := range resp.Choices {
			if choice.Message == nil || choice.Message.Content == nil {
				continue
			}

			tokenCount += k.tokenCounter.CountTokens(*choice.Message.Content)
		}

		if *resp.Choices[0].FinishReason == azopenai.CompletionsFinishReason("stop") {
			if resp.Choices[0].Message != nil || resp.Choices[0].Message.Content != nil {
				content := *resp.Choices[0].Message.Content
				err = json.Unmarshal([]byte(content), &keywordsRelatedQueries)
				if err != nil {
					return nil, tokenCount, err
				}
			}
		}
		if len(keywordsRelatedQueries.Keywords) > 0 && len(keywordsRelatedQueries.RelatedQueries) > 0 {
			return &keywordsRelatedQueries, tokenCount, nil
		}
	}
	return nil, tokenCount, fmt.Errorf("failed to get keywords and related queries, max retries: %d", maxRetries)
}

func getPrompt(keywordQuery string) string {
	return fmt.Sprintf(`Please follow below instructions.
1) Generate <keywords> to represent <question: %s>.
    - One word that best represents the question.
    - Please generate it with two or fewer.
2) Generate <related_queries> for each of the <keywords>.
    - The meaning of each generated sentence must differ from the question, and the meaning between each sentence must also be distinct.

<example>
question: "윤석열 대통령 베네딕토 16세 전 교황 평가"
{
    "keywords": ["윤석열", "베네딕토 16세"],
    "related_queries": ["윤석열 대통령의 최근 행보에 대해서 말해줄래?", "베네딕토 16세 전 교황의 업적은 무엇이 있어?"]
}
Please provide the results in JSON format with the following two keys: 'keywords' and 'related_queries'`, keywordQuery)
}
