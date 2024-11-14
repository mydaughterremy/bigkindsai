package aggregator

import (
	"context"

	"bigkinds.or.kr/proto"
)

type Aggregator interface {
	Aggregate(ctx context.Context, rerankerConfig *proto.Reranker, query string, itemsList []*proto.Items, k int32) ([]*proto.Item, error)
}
