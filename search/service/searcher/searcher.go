package searcher

import (
	"context"

	"bigkinds.or.kr/proto"
)

const groupIDSearchField = "upstage_group_id"

type Searcher interface {
	Search(ctx context.Context, req *proto.SearchRequest, config *proto.SearchConfig) ([]*proto.Item, error)
}
