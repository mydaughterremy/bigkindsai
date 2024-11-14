package query_strategy

import (
	"errors"

	"bigkinds.or.kr/pkg/encoder"
	e "bigkinds.or.kr/proto/encoder"
	"bigkinds.or.kr/proto/searcher"
)

type VectorQueryStrategy struct {
	SentenceTransformers encoder.Encoder
}

func (v *VectorQueryStrategy) CreateOpenSearchQuery(req *QueryIngredients, osSearcherConfig *searcher.OpenSearchSearcher) (map[string]interface{}, error) {
	vectorQueryStrategy := osSearcherConfig.GetVectorQueryStrategy()

	// get raw query
	queryField := getQueryField(vectorQueryStrategy.KnnStrategy)
	rawQuery, ok := req.Query[queryField]
	if !ok {
		return nil, errors.New("query field does not exist in request")
	}

	// get query embedding
	queryEncoder := vectorQueryStrategy.Encoder
	queryEmbedding, err := v.getQueryEmbedding(rawQuery, queryEncoder)
	if err != nil {
		return nil, err
	}
	knnQuery := createKNNQuery(queryField, vectorQueryStrategy.KnnStrategy.K, queryEmbedding)

	// add filter
	return addFilterToKNNQuery(vectorQueryStrategy.FilterStrategy, knnQuery, queryField)
}

func addFilterToKNNQuery(filterStrategy *searcher.OpenSearchSearcher_VectorQueryStrategy_FilterStrategy, knnQuery map[string]interface{}, queryField string) (map[string]interface{}, error) {
	if filterStrategy == nil {
		return map[string]interface{}{
			"query": knnQuery,
		}, nil
	}
	switch filterStrategy.Mode {
	case "efficient-knn":
		knnQuery["knn"].(map[string]interface{})[queryField].(map[string]interface{})["filter"] = filterStrategy.Filter.AsMap()
		return map[string]interface{}{
			"query": knnQuery,
		}, nil
	case "post-filter":
		return map[string]interface{}{
			"query":       knnQuery,
			"post_filter": filterStrategy.Filter.AsMap(),
		}, nil
	default:
		return nil, errors.New("invalid filter mode")
	}
}

func createKNNQuery(queryField string, k int32, queryEmbedding []float32) map[string]interface{} {
	knnQuery := map[string]interface{}{
		"knn": map[string]interface{}{
			queryField: map[string]interface{}{
				"vector": queryEmbedding,
				"k":      k,
			},
		},
	}
	return knnQuery
}

func (v *VectorQueryStrategy) getQueryEmbedding(query string, encoder *e.Encoder) ([]float32, error) {
	// create request
	switch encoder.Encoder.(type) {
	case *e.Encoder_SentenceTransformer_:
		return v.SentenceTransformers.Encode(query)
	default:
		return nil, errors.New("invalid encoder type")
	}
}

func getQueryField(strategy *searcher.OpenSearchSearcher_VectorQueryStrategy_KNNStrategy) string {
	if strategy.SourceField == "" {
		return strategy.Field
	}
	return strategy.SourceField
}

func NewVectorQueryStrategy(sentenceTransformers encoder.Encoder) *VectorQueryStrategy {
	return &VectorQueryStrategy{
		SentenceTransformers: sentenceTransformers,
	}
}
