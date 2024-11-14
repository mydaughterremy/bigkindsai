package query_strategy

import (
	"errors"
	"reflect"
	"testing"

	e "bigkinds.or.kr/proto/encoder"
	"bigkinds.or.kr/proto/searcher"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

type MockEncoder struct {
}

func (m *MockEncoder) Encode(text string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3}, nil
}

func getValidVectorSearchConfig() *searcher.OpenSearchSearcher {
	filter, err := structpb.NewStruct(map[string]interface{}{
		"bool": map[string]interface{}{
			"must": []interface{}{
				map[string]interface{}{
					"range": map[string]interface{}{
						"price": map[string]interface{}{
							"gte": 0,
							"lte": 15,
						},
					},
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	return &searcher.OpenSearchSearcher{
		QueryStrategy: &searcher.OpenSearchSearcher_VectorQueryStrategy_{
			VectorQueryStrategy: &searcher.OpenSearchSearcher_VectorQueryStrategy{
				KnnStrategy: &searcher.OpenSearchSearcher_VectorQueryStrategy_KNNStrategy{
					Field:       "knn-test-field",
					SourceField: "",
					K:           10,
				},
				FilterStrategy: &searcher.OpenSearchSearcher_VectorQueryStrategy_FilterStrategy{
					Mode:   "efficient-knn",
					Filter: filter,
				},
				Encoder: &e.Encoder{
					Encoder: &e.Encoder_SentenceTransformer_{
						SentenceTransformer: &e.Encoder_SentenceTransformer{},
					},
				},
			},
		},
	}
}

func getValidVectorSearchReq() *QueryIngredients {
	return &QueryIngredients{
		Query: map[string]string{
			"knn-test-field": "test-knn-query",
		},
	}
}

func getTestVectorQueryStrategy() *VectorQueryStrategy {
	return NewVectorQueryStrategy(&MockEncoder{})
}

func TestCreateKNNQuery(t *testing.T) {
	tests := []struct {
		name           string
		queryField     string
		k              int32
		queryEmbedding []float32
		expectedQuery  map[string]interface{}
	}{
		{
			name:           "Valid KNN query",
			queryField:     "vectorField",
			k:              10,
			queryEmbedding: []float32{0.1, 0.2, 0.3},
			expectedQuery: map[string]interface{}{
				"knn": map[string]interface{}{
					"vectorField": map[string]interface{}{
						"vector": []float32{0.1, 0.2, 0.3},
						"k":      int32(10),
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			knnQuery := createKNNQuery(test.queryField, test.k, test.queryEmbedding)
			assert.Equal(t, test.expectedQuery, knnQuery)
		})
	}
}

func TestGetQueryEmbedding(t *testing.T) {
	strategy := getTestVectorQueryStrategy()
	tests := []struct {
		name          string
		query         string
		encoder       *e.Encoder
		expectedEmbed []float32
		expectedError error
	}{
		{
			name:  "Valid Global Encoder - Sentence Transformers",
			query: "test query",
			encoder: &e.Encoder{
				Encoder: &e.Encoder_SentenceTransformer_{
					SentenceTransformer: &e.Encoder_SentenceTransformer{},
				},
			},
			expectedEmbed: []float32{0.1, 0.2, 0.3},
			expectedError: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			embed, err := strategy.getQueryEmbedding(test.query, test.encoder)
			assert.Equal(t, test.expectedEmbed, embed)
			assert.Equal(t, test.expectedError, err)
		})
	}
}

func TestGetQueryField(t *testing.T) {
	tests := []struct {
		name          string
		strategy      *searcher.OpenSearchSearcher_VectorQueryStrategy_KNNStrategy
		expectedField string
	}{
		{
			name: "Field only",
			strategy: &searcher.OpenSearchSearcher_VectorQueryStrategy_KNNStrategy{
				Field:       "testField",
				SourceField: "",
			},
			expectedField: "testField",
		},
		{
			name: "Both Field and SourceField",
			strategy: &searcher.OpenSearchSearcher_VectorQueryStrategy_KNNStrategy{
				Field:       "testField",
				SourceField: "testSourceField",
			},
			expectedField: "testSourceField",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := getQueryField(test.strategy)
			assert.Equal(t, test.expectedField, result)
		})
	}
}

func TestAddFilterToKNNQuery(t *testing.T) {
	queryField := "queryField"
	dummyFilter, err := structpb.NewStruct(map[string]interface{}{
		"dummy_filter": "value",
	})
	if err != nil {
		panic(err)
	}

	tests := []struct {
		name           string
		filterStrategy *searcher.OpenSearchSearcher_VectorQueryStrategy_FilterStrategy
		knnQuery       map[string]interface{}
		expectedQuery  map[string]interface{}
		expectedError  error
	}{
		{
			name: "Valid efficient-knn filter mode",
			knnQuery: map[string]interface{}{
				"knn": map[string]interface{}{
					"queryField": map[string]interface{}{
						"vector": []float32{0.1, 0.2, 0.3},
						"k":      10,
					},
				},
			},
			filterStrategy: &searcher.OpenSearchSearcher_VectorQueryStrategy_FilterStrategy{
				Mode:   "efficient-knn",
				Filter: dummyFilter,
			},
			expectedQuery: map[string]interface{}{
				"query": map[string]interface{}{
					"knn": map[string]interface{}{
						"queryField": map[string]interface{}{
							"vector": []float32{0.1, 0.2, 0.3},
							"k":      10,
							"filter": dummyFilter.AsMap(),
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "Valid post-filter filter mode",
			knnQuery: map[string]interface{}{
				"knn": map[string]interface{}{
					"queryField": map[string]interface{}{
						"vector": []float32{0.1, 0.2, 0.3},
						"k":      10,
					},
				},
			},
			filterStrategy: &searcher.OpenSearchSearcher_VectorQueryStrategy_FilterStrategy{
				Mode:   "post-filter",
				Filter: dummyFilter,
			},
			expectedQuery: map[string]interface{}{
				"query": map[string]interface{}{
					"knn": map[string]interface{}{
						"queryField": map[string]interface{}{
							"vector": []float32{0.1, 0.2, 0.3},
							"k":      10,
						},
					},
				},
				"post_filter": dummyFilter.AsMap(),
			},
			expectedError: nil,
		},
		{
			name: "No filter mode",
			knnQuery: map[string]interface{}{
				"knn": map[string]interface{}{
					"queryField": map[string]interface{}{
						"vector": []float32{0.1, 0.2, 0.3},
						"k":      10,
					},
				},
			},
			filterStrategy: nil,
			expectedQuery: map[string]interface{}{
				"query": map[string]interface{}{
					"knn": map[string]interface{}{
						"queryField": map[string]interface{}{
							"vector": []float32{0.1, 0.2, 0.3},
							"k":      10,
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "Invalid filter mode",
			knnQuery: map[string]interface{}{
				"knn": map[string]interface{}{
					"queryField": map[string]interface{}{
						"vector": []float32{0.1, 0.2, 0.3},
						"k":      10,
					},
				},
			},
			filterStrategy: &searcher.OpenSearchSearcher_VectorQueryStrategy_FilterStrategy{
				Mode:   "invalid",
				Filter: dummyFilter,
			},
			expectedQuery: nil,
			expectedError: errors.New("invalid filter mode"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			modifiedQuery, err := addFilterToKNNQuery(test.filterStrategy, test.knnQuery, queryField)
			if !reflect.DeepEqual(test.expectedQuery, modifiedQuery) {
				t.Errorf("expected query: %v, got: %v", test.expectedQuery, modifiedQuery)
			}
			assert.Equal(t, test.expectedError, err)
		})
	}
}

func TestCreateOpenSearchVectorQuery(t *testing.T) {
	strategy := getTestVectorQueryStrategy()
	tests := []struct {
		name           string
		req            *QueryIngredients
		osSearcherConf *searcher.OpenSearchSearcher
		expectedQuery  map[string]interface{}
		expectedError  error
	}{
		{
			name:           "Valid Vector Query",
			req:            getValidVectorSearchReq(),
			osSearcherConf: getValidVectorSearchConfig(),
			expectedQuery: map[string]interface{}{
				"query": map[string]interface{}{
					"knn": map[string]interface{}{
						"knn-test-field": map[string]interface{}{
							"vector": []float32{0.1, 0.2, 0.3},
							"k":      int32(10),
							"filter": map[string]interface{}{
								"bool": map[string]interface{}{
									"must": []interface{}{
										map[string]interface{}{
											"range": map[string]interface{}{
												"price": map[string]interface{}{
													"gte": float64(0),
													"lte": float64(15),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedError: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			query, err := strategy.CreateOpenSearchQuery(test.req, test.osSearcherConf)
			if !reflect.DeepEqual(test.expectedQuery, query) {
				t.Errorf("expected query: %v, got: %v", test.expectedQuery, query)
			}
			assert.Equal(t, test.expectedError, err)
		})
	}
}
