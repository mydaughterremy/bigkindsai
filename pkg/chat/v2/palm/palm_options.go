package palm

type PalmOptions struct {
	Endpoint  string
	ProjectID string
	Token     string
	Models    []string
}

type PalmPredictionOptions struct {
	Model           string   `json:"model"`
	Context         *string  `json:"context"`
	Temperature     *float32 `json:"temperature"`
	MaxOutputTokens *int32   `json:"maxOutputTokens"`
	TopP            *float32 `json:"topP"`
	TopK            *int32   `json:"topK"`
}

func (o *PalmOptions) IsLLMOption() {}

const gcpEndpoint = "us-central1-aiplatform.googleapis.com"

func NewPalmOptions(opts ...func(*PalmOptions)) *PalmOptions {
	options := &PalmOptions{
		Endpoint: gcpEndpoint,
		Models:   palmModels(),
	}

	for _, o := range opts {
		o(options)
	}

	return options
}

func NewPalmPredictionOptions(opts ...func(*PalmPredictionOptions)) *PalmPredictionOptions {
	options := &PalmPredictionOptions{}

	for _, o := range opts {
		o(options)
	}

	return options
}

// PALM OPTIONS

func WithEndpoint(url string) func(*PalmOptions) {
	return func(o *PalmOptions) {
		o.Endpoint = url
	}
}

func WithProjectID(id string) func(*PalmOptions) {
	return func(o *PalmOptions) {
		o.ProjectID = id
	}
}

func WithToken(token string) func(*PalmOptions) {
	return func(o *PalmOptions) {
		o.Token = token
	}
}

func WithModels(models []string) func(*PalmOptions) {
	return func(o *PalmOptions) {
		o.Models = models
	}
}

// PALM PREDICTION OPTIONS

func WithModel(model string) func(*PalmPredictionOptions) {
	return func(o *PalmPredictionOptions) {
		o.Model = model
	}
}

func WithContext(context string) func(*PalmPredictionOptions) {
	return func(o *PalmPredictionOptions) {
		o.Context = &context
	}
}

func WithTemperature(temperature float32) func(*PalmPredictionOptions) {
	return func(o *PalmPredictionOptions) {
		o.Temperature = &temperature
	}
}

func WithMaxOutputTokens(maxOutputTokens int32) func(*PalmPredictionOptions) {
	return func(o *PalmPredictionOptions) {
		o.MaxOutputTokens = &maxOutputTokens
	}
}

func WithTopP(topP float32) func(*PalmPredictionOptions) {
	return func(o *PalmPredictionOptions) {
		o.TopP = &topP
	}
}

func WithTopK(topK int32) func(*PalmPredictionOptions) {
	return func(o *PalmPredictionOptions) {
		o.TopK = &topK
	}
}

// helpers

func palmModels() []string {
	return []string{
		"chat-bison@001",
		"chat-bison@latest",
	}
}
