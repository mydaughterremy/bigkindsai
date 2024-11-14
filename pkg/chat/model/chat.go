package model

type ChatPayload struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ChatResponse struct {
	Payload      *ChatPayload
	FinishReason string
	FunctionCall *ChatFunction
}
