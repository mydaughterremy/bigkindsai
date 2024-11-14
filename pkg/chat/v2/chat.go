package chat

import (
	"context"
	"net/http"
	"strings"

	"google.golang.org/grpc/metadata"
)

const (
	ForwardPrefix = "x-forwarded-for-"
	UpstagePrefix = "x-upstage-"

	X_UPSTAGE_ASKUP_PROMPT = "X-Upstage-Askup-Prompt"
	X_UPSTAGE_ASKUP_MODEL  = "X-Upstage-Askup-Model"
	X_UPSTAGE_TENANT_ID    = "X-Upstage-Tenant-Id"
	X_UPSTAGE_GROUP_ID     = "X-Upstage-Group-Id"
)

type ChatPayload struct {
	Role         string        `json:"role"`
	Content      string        `json:"content"`
	Name         *string       `json:"name,omitempty"`
	FunctionCall *ChatFunction `json:"function_call,omitempty"`
	ToolCalls    []*ChatTool   `json:"tool_calls,omitempty"`
	ToolCallID   string        `json:"tool_call_id,omitempty"`
}

type ChatFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ChatTool struct {
	Id       string        `json:"id"`
	Type     string        `json:"type"`
	Function *ChatFunction `json:"function"`
}

type ChatResponse struct {
	Payload      *ChatPayload
	FinishReason string
	FunctionCall *ChatFunction
	ToolCalls    []*ChatTool
}

type LLMOptions interface {
	IsLLMOption()
}

func NewOutgoingContextWithForwardHeaders(ctx context.Context, header *http.Header) context.Context {
	md := metadata.New(map[string]string{})
	for k, v := range *header {
		if strings.HasPrefix(strings.ToLower(k), ForwardPrefix) {
			md[strings.ToLower(k)] = v
		}
		if strings.HasPrefix(strings.ToLower(k), UpstagePrefix) {
			md[strings.ToLower(k)] = v
		}
	}
	return metadata.NewOutgoingContext(ctx, md)
}

func AddForwardHeadersFromIncomingContext(context *context.Context, header *http.Header) {
	if md, ok := metadata.FromIncomingContext(*context); ok {
		for k, v := range md {
			if strings.HasPrefix(k, ForwardPrefix) {
				(*header).Set(strings.TrimPrefix(k, ForwardPrefix), strings.Join(v, ","))
			}
		}
	}
}

func OverrideModelFromHeaders(header *http.Header, modelName *string) {
	newModelName := header.Values(X_UPSTAGE_ASKUP_MODEL)
	if len(newModelName) == 0 {
		return
	}
	*modelName = newModelName[0]
}
