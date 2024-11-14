package model

type Completion struct {
	Object     string          `json:"object"`
	Id         string          `json:"id"`
	Created    int             `json:"created"`
	Delta      CompletionDelta `json:"delta"`
	TokenUsage int             `json:"token_usage"`
}

type CompletionDelta struct {
	Content        string      `json:"content"`
	References     []Reference `json:"references"`
	Keywords       []string    `json:"keywords"`
	RelatedQueries []string    `json:"related_queries"`
}

type CompletionLLM struct {
	Provider         string `json:"provider"`
	ModelName        string `json:"model_name"`
	MaxFallbackCount int    `json:"max_fallback_count"`
	KeyIndex         int    `json:"key_index"`
}
