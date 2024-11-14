package model

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

type CreateChatCompletionRequest struct {
	Messages []*Message `json:"messages"`
	Provider string     `json:"provider"`
}

type Completion struct {
	Object     string           `json:"object"`
	Id         string           `json:"id"`
	Created    int              `json:"created"`
	Delta      *CompletionDelta `json:"delta"`
	TokenUsage int              `json:"token_usage"`
}

type CompletionDelta struct {
	Content        string      `json:"content"`
	References     []Reference `json:"references"`
	Keywords       []string    `json:"keywords"`
	RelatedQueries []string    `json:"related_queries"`
}

func (c *Completion) Merge(chunk *Completion) error {
	if chunk == nil || chunk.Delta == nil {
		return nil
	}

	c.Delta.Content += chunk.Delta.Content
	c.Delta.References = append(c.Delta.References, chunk.Delta.References...)
	c.Delta.Keywords = append(c.Delta.Keywords, chunk.Delta.Keywords...)
	c.Delta.RelatedQueries = append(c.Delta.RelatedQueries, chunk.Delta.RelatedQueries...)

	return nil
}
