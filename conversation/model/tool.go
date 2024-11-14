package model

type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type ToolCallParameter struct {
	Id        string `json:"id"`
	Arguments string `json:"arguments"`
	Name      string `json:"name"`
}

type ToolCallRequest struct {
	ToolName   string               `json:"tool_name"`
	Parameters []*ToolCallParameter `json:"parameters"`
}
