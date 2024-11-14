package palm

import "encoding/json"

// this model implements Google VertexAI's
// projects.locations.publishers.models.predict API
// to use ChatBison model a.k.a. "palm2"
// api ref.
// https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.publishers.models/predict

type VertexAIPalm struct {
	Instances  []VertexAIPalmInstace  `json:"instances"`
	Parameters VertexAIPalmParameters `json:"parameters"`
}

type VertexAIPalmInstace struct {
	Context  string                       `json:"context,omitempty"`
	Examples []VertexAIPalmExample        `json:"examples,omitempty"`
	Messages []VertexAIPalmMessagePayload `json:"messages"`
}

type VertexAIPalmExample struct {
	Input  VertexAIPalmMessagePayload `json:"input"`
	Output VertexAIPalmMessagePayload `json:"output"`
}

type VertexAIPalmMessagePayload struct {
	Author  string `json:"author"`
	Content string `json:"content"`
	// CitationMetadata *VertexAIPalmCitations `json:"citationMetadata,omitempty"`
}

type VertexAIPalmCitations struct {
	Citations []string `json:"citations"`
}

type VertexAIPalmParameters struct {
	Temparature     *float32 `json:"temperature,omitempty"`
	MaxOutputTokens *int32   `json:"maxOutputTokens,omitempty"`
	TopP            *float32 `json:"topP,omitempty"`
	TopK            *int32   `json:"topK,omitempty"`
}

type VertexAIPalmResponse struct {
	Predections []VertexAIPalmPredictions `json:"predictions"`
	Metadata    json.RawMessage           `json:"metadata"` // unimplemented
}

type VertexAIPalmPredictions struct {
	CitationMetadata json.RawMessage              `json:"citationMetadata"` // unimplemented
	SafetyAttributes json.RawMessage              `json:"safetyAttributes"` // unimplemented
	Candidates       []VertexAIPalmMessagePayload `json:"candidates"`
}

type JerryIsWatching struct {
}
