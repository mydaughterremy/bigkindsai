package query_strategy

import (
	"bigkinds.or.kr/proto/searcher"
	"google.golang.org/protobuf/types/known/structpb"
)

type QueryIngredients struct {
	Query   map[string]string
	Exclude string
	Filters []*structpb.Struct
	Size    int32
}

type QueryStrategy interface {
	CreateOpenSearchQuery(req *QueryIngredients, osSearcherConfig *searcher.OpenSearchSearcher) (map[string]interface{}, error)
}
