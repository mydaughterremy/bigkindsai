package llmclient

import (
	"bigkinds.or.kr/pkg/chat/v2"
	"fmt"
	"net/http"
	"os"
	"strings"

	"bigkinds.or.kr/pkg/chat/v2/gpt"
)

func NewOpenAIClient(client *http.Client, model string, opts ...func(*chat.GptOptions)) (*gpt.GPT, error) {
	key := os.Getenv("UPSTAGE_OPENAI_KEY")

	opts = append(
		opts,
		chat.WithCustomEndpoint("https://api.openai.com/v1/chat/completions"),
		chat.WithKey(key),
		chat.WithModels([]string{model}),
		chat.WithAPIType(chat.GPTAPITypeOpenAI),
	)

	options := chat.NewGptOptions(opts...)

	return &gpt.GPT{
		Client: client,
		Option: options,
	}, nil
}

func NewAzureClient(client *http.Client, model string, keyIndex int, opts ...func(*chat.GptOptions)) (*gpt.GPT, error) {
	endpointKeyMap := os.Getenv("UPSTAGE_AZURE_ENDPOINT_KEY_MAP")
	if endpointKeyMap == "" {
		return nil, fmt.Errorf("endpoint key map is not in env")
	}
	endpointKeyMapList := strings.Split(endpointKeyMap, ",")
	if keyIndex >= len(endpointKeyMapList) {
		return nil, fmt.Errorf("azure key index is out of range")
	}

	splited := strings.Split(endpointKeyMapList[keyIndex], ";")
	endpoint, key := splited[0], splited[1]
	opts = append(
		opts,
		chat.WithCustomEndpoint(endpoint),
		chat.WithKey(key),
		chat.WithModels([]string{model}),
		chat.WithAPIType(chat.GPTAPITypeAzure),
	)

	options := chat.NewGptOptions(opts...)

	return &gpt.GPT{
		Client: client,
		Option: options,
	}, nil
}

func NewSolarClient(client *http.Client, model string, opts ...func(options *chat.GptOptions)) (*gpt.GPT, error) {
	key := os.Getenv("UPSTAGE_SOLAR_KEY")

	opts = append(
		opts,
		chat.WithCustomEndpoint("https://api.upstage.ai/v1/solar/chat/completions"),
		chat.WithKey(key),
		chat.WithModels([]string{model}),
	)

	options := chat.NewGptOptions(opts...)

	return &gpt.GPT{
		Client: client,
		Option: options,
	}, nil
}

func NewClient(client *http.Client, provider string, model string, keyIndex int, opts func(*chat.GptOptions)) (*gpt.GPT, error) {
	switch provider {
	case "openai":
		return NewOpenAIClient(client, model, opts)
	case "azure":
		return NewAzureClient(client, model, keyIndex, opts)
	case "upstage":
		return NewSolarClient(client, model, opts)
	default:
		return nil, fmt.Errorf("invalid provider: %s", provider)
	}
}
