package chat

import (
	"os"
	"strings"
)

type GPTAPIType string

const GPTAPITypeOpenAI GPTAPIType = "openai"
const GPTAPITypeAzure GPTAPIType = "azure"

const OpenAICompletionEndpoint = "https://api.openai.com/v1/chat/completions"
const SolarCompletionEndpoint = "https://api.upstage.ai/v1/solar/chat/completions"

type GptOptions struct {
	Endpoint   string
	ApiKey     string
	Models     []string
	Streamable bool
	ApiType    GPTAPIType
}

type GptPredictionOptions struct {
	Model            string   `json:"model"`
	Functions        []string `json:"functions"`
	FunctionCall     *string  `json:"function_call"`
	Tools            []string `json:"tools"`
	ToolChoice       *string  `json:"tool_choice"`
	Temperature      *float32 `json:"temperature"`
	TopP             *float32 `json:"top_p"`
	MaxTokens        *int32   `json:"max_tokens"`
	PresencePenalty  *float32 `json:"presence_penalty"`
	FrequencyPenalty *float32 `json:"frequency_penalty"`
	ResponseFormat   *string  `json:"response_format"`
	Stream           bool     `json:"stream"`
	Seed             *int64   `json:"seed"`
}

// 정보 필요함, 수정 필요함
func GetModels() []string {
	modelList, ok := os.LookupEnv("UPSTAGE_LLM_MODEL")
	if !ok {
		modelList = "upstage,2"
	}
	models := strings.Split(modelList, ",")
	return models
}
func GetLLMOptions() []string {
	modelList := os.Getenv("UPSTAGE_LLM_MODEL")
	if modelList == "" {
		modelList = "upstage/solar-mini"
	}
	models := strings.Split(modelList, "/")
	return models
}

func (o *GptPredictionOptions) IsLLMOption() {}

func NewGptOptions(opts ...func(*GptOptions)) *GptOptions {
	options := &GptOptions{}
	models := GetLLMOptions()
	switch models[0] {
	case "openai":
		options.Endpoint = OpenAICompletionEndpoint
		options.Models = gptModels()
	case "upstage":
		options.Endpoint = SolarCompletionEndpoint
		options.Models = solarModels()
	}

	for _, opt := range opts {
		opt(options)
	}

	return options
}

func NewGptPredictionOptions(opts ...func(*GptPredictionOptions)) *GptPredictionOptions {
	options := defaultGptPredictionOptions()

	for _, opt := range opts {
		opt(options)
	}

	return options
}

func defaultGptPredictionOptions() *GptPredictionOptions {
	return &GptPredictionOptions{}
}

// GPT OPTIONS

func WithCustomEndpoint(url string) func(*GptOptions) {
	return func(o *GptOptions) {
		o.Endpoint = url
	}
}

func WithKey(key string) func(*GptOptions) {
	return func(o *GptOptions) {
		o.ApiKey = key
	}
}

func WithModels(models []string) func(*GptOptions) {
	return func(o *GptOptions) {
		o.Models = models
	}
}

func WithStreamEnabled(o *GptOptions) {
	o.Streamable = true
}
func WithStreamDisabled(o *GptOptions) {
	o.Streamable = false
}

func WithAPIType(apiType GPTAPIType) func(*GptOptions) {
	return func(o *GptOptions) {
		o.ApiType = apiType
	}
}

// GPT PREDICTION OPTIONS

func WithModel(model string) func(*GptPredictionOptions) {
	return func(o *GptPredictionOptions) {
		o.Model = model
	}
}

func WithFunctions(functions []string) func(*GptPredictionOptions) {
	return func(o *GptPredictionOptions) {
		o.Functions = functions
	}
}
func WithNoFunctions(options *GptPredictionOptions) {
	options.Functions = nil
}

func WithFunctionCall(functionCall string) func(*GptPredictionOptions) {
	return func(o *GptPredictionOptions) {
		o.FunctionCall = &functionCall
	}
}

func WithTools(tools []string) func(*GptPredictionOptions) {
	return func(o *GptPredictionOptions) {
		o.Tools = tools
	}
}

func WithToolChoice(toolChoice string) func(*GptPredictionOptions) {
	return func(o *GptPredictionOptions) {
		o.ToolChoice = &toolChoice
	}
}

func WithTemperature(temperature float32) func(*GptPredictionOptions) {
	return func(o *GptPredictionOptions) {
		o.Temperature = &temperature
	}
}

func WithTopP(topP float32) func(*GptPredictionOptions) {
	return func(o *GptPredictionOptions) {
		o.TopP = &topP
	}
}

func WithMaxTokens(maxTokens int32) func(*GptPredictionOptions) {
	return func(o *GptPredictionOptions) {
		o.MaxTokens = &maxTokens
	}
}

func WithPresencePenalty(presencePenalty float32) func(*GptPredictionOptions) {
	return func(o *GptPredictionOptions) {
		o.PresencePenalty = &presencePenalty
	}
}

func WithFrequencyPenalty(frequencyPenalty float32) func(*GptPredictionOptions) {
	return func(o *GptPredictionOptions) {
		o.FrequencyPenalty = &frequencyPenalty
	}
}

func WithResponseFormat(responseFormat string) func(*GptPredictionOptions) {
	return func(o *GptPredictionOptions) {
		o.ResponseFormat = &responseFormat
	}
}

func WithNilTollCall(opts *GptPredictionOptions) {
	opts.Tools = nil
}
func WithNilTollChoice(opts *GptPredictionOptions) {
	opts.ToolChoice = nil
}

func WithStream(o *GptPredictionOptions) {
	o.Stream = true
}

func WithoutStream(o *GptPredictionOptions) {
	o.Stream = false
}

func WithSeed(seed int64) func(*GptPredictionOptions) {
	return func(o *GptPredictionOptions) {
		o.Seed = &seed
	}
}

func solarModels() []string {
	return []string{
		"solar-mini",
	}
}

func gptModels() []string {
	return []string{
		"gpt-4",
		"gpt-4-0314",
		"gpt-4-32k",
		"gpt-4-32k-0314",
		"gpt-4-0613",
		"gpt-3.5-turbo",
		"gpt-3.5-turbo-0301",
		"gpt-3.5-turbo-0613",
		"gpt-3.5-turbo-1106",
	}
}
