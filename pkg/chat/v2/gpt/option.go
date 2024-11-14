package gpt

type GPTAPIType string

const GPTAPITypeOpenAI GPTAPIType = "openai"
const GPTAPITypeAzure GPTAPIType = "azure"

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

func (o *GptPredictionOptions) IsLLMOption() {}

func NewGptOptions(opts ...func(*GptOptions)) *GptOptions {
	options := &GptOptions{
		Endpoint: openAIchatCompletion,
		Models:   gptModels(),
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

func WithStream(o *GptPredictionOptions) {
	o.Stream = true
}

func WithSeed(seed int64) func(*GptPredictionOptions) {
	return func(o *GptPredictionOptions) {
		o.Seed = &seed
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
