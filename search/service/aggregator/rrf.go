package aggregator

import (
	"context"
	"sort"

	"bigkinds.or.kr/proto"
)

type RRF struct {
}

type RankedItem struct {
	ID    string
	Score float32
	Item  *proto.Item
}

const K = 60

func (s *RRF) Aggregate(ctx context.Context, rerankerConfig *proto.Reranker, query string, itemsList []*proto.Items, k int32) ([]*proto.Item, error) {
	resultMap := make(map[string]*RankedItem)
	for _, candidates := range itemsList {
		for rank, candidate := range candidates.Items {
			if _, ok := resultMap[candidate.Id]; !ok {
				resultMap[candidate.Id] = &RankedItem{
					ID:    candidate.Id,
					Score: 0,
					Item:  candidate,
				}
			}
			resultMap[candidate.Id].Score += 1 / float32(K+(rank+1)) // rank starts from 1, not 0
		}
	}

	resultSlice := make([]*RankedItem, 0, len(resultMap))
	for _, v := range resultMap {
		resultSlice = append(resultSlice, v)
	}

	sort.SliceStable(resultSlice, func(i, j int) bool {
		return resultSlice[i].Score > resultSlice[j].Score
	})

	rerankedItems := make([]*proto.Item, 0, k)
	for _, v := range resultSlice[:min(k, int32(len(resultSlice)))] {
		rerankedItems = append(rerankedItems, v.Item)
	}

	return rerankedItems, nil
}
