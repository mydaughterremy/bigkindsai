package reranker

import (
	"context"
	"errors"

	"bigkinds.or.kr/proto"
)

var ErrRerankerTimeout = errors.New("reranker timeout")

type Reranker interface {
	Rerank(ctx context.Context, config *proto.Reranker, query string, items []*proto.Item, k int32) ([]*proto.Item, error)
}
