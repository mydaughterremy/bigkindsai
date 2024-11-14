package chat

import (
	"context"

	"bigkinds.or.kr/proto"

	"bigkinds.or.kr/pkg/chat/model"
)

type ChatModel interface {
	Chat(ctx context.Context, request *proto.ChatRequest) (response *proto.ChatResponse, err error)
	CreateChatStream(ctx context.Context, request *proto.ChatRequest) (ChatStream, error)
}

type ChatStream interface {
	ReadUntilNow() *model.ChatCompletionResponse
	Recv() (*proto.ChatResponse, error)
}
