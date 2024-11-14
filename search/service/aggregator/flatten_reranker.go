package aggregator

import (
	"context"

	"bigkinds.or.kr/proto"
	"bigkinds.or.kr/search/service/reranker"
)

type FlattenReranker struct {
	Reranker reranker.Reranker
}

type Passages struct {
	TargetFields []string      `json:"target_fields"`
	Items        []*proto.Item `json:"items"`
}

type FlattenRerankerRequest struct {
	Query    string   `json:"query"`
	Passages Passages `json:"passages"`
	K        int32    `json:"k"`
}

type FlattenRerankerResponse struct {
	Topk []struct {
		Item  *proto.Item `json:"item"`
		Score float64     `json:"score"`
	} `json:"topk"`
}

func (s *FlattenReranker) Aggregate(ctx context.Context, rerankerConfig *proto.Reranker, query string, itemsList []*proto.Items, k int32) ([]*proto.Item, error) {
	if len(itemsList) == 0 {
		return nil, nil
	}
	//flatten items
	flattenedItems := make([]*proto.Item, 0, len(itemsList)*len(itemsList[0].Items))
	for _, items := range itemsList {
		flattenedItems = append(flattenedItems, items.Items...)
	}

	return s.Reranker.Rerank(ctx, rerankerConfig, query, flattenedItems, k)
}
