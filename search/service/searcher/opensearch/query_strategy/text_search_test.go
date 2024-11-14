package query_strategy

import (
	"fmt"
	"reflect"
	"testing"

	"bigkinds.or.kr/proto/searcher"
	"github.com/stretchr/testify/assert"
)

func getValidTextSearchConfig() *searcher.OpenSearchSearcher {
	return &searcher.OpenSearchSearcher{
		QueryStrategy: &searcher.OpenSearchSearcher_TextQueryStrategy_{
			TextQueryStrategy: &searcher.OpenSearchSearcher_TextQueryStrategy{
				ScoringMode: "relevance",
				Fields: map[string]*searcher.OpenSearchSearcher_Field{
					"title": {
						ValueType:   "text",
						QueryType:   "should",
						SourceField: "TITLE",
					},
					"description": {
						ValueType:   "text",
						QueryType:   "should",
						SourceField: "DESCRIPTION",
					},
					"author": {
						ValueType:       "text",
						QueryType:       "filter",
						SourceField:     "AUTHOR",
						SecondaryShould: false,
					},
					"category": {
						ValueType:       "text",
						QueryType:       "filter",
						SourceField:     "CATEGORY",
						SecondaryShould: true,
					},
				},
			},
		},
	}
}

func getValidTextQueryInputs() []*TextQueryInput {
	return []*TextQueryInput{
		{
			Field: &searcher.OpenSearchSearcher_Field{
				ValueType:   "text",
				QueryType:   "should",
				SourceField: "TITLE",
			},
			Value: "test-title",
		},
		{
			Field: &searcher.OpenSearchSearcher_Field{
				ValueType:   "text",
				QueryType:   "should",
				SourceField: "DESCRIPTION",
			},
			Value: "test-description",
		},
		{
			Field: &searcher.OpenSearchSearcher_Field{
				ValueType:       "text",
				QueryType:       "filter",
				SourceField:     "AUTHOR",
				SecondaryShould: false,
			},
			Value: "test-author",
		},
		{
			Field: &searcher.OpenSearchSearcher_Field{
				ValueType:       "text",
				QueryType:       "filter",
				SourceField:     "CATEGORY",
				SecondaryShould: true,
			},
			Value: "test-category",
		},
	}
}

func getValidTextSearchReq() *QueryIngredients {
	return &QueryIngredients{
		Query: map[string]string{
			"title":       "test-title",
			"description": "test-description",
			"author":      "test-author",
			"category":    "test-category",
		},
		Exclude: "test-exclude",
	}
}

func TestCreateShouldAndFilterQuery(t *testing.T) {
	cases := []struct {
		TestName            string
		TextQueryInputs     []*TextQueryInput
		ExpectedShouldQuery []map[string]interface{}
		ExpectedFilterQuery []map[string]interface{}
		Error               error
	}{
		{
			TestName:        "TestCreateShouldAndFilterQueryValidCase: not use shouldOrFilterQuery",
			TextQueryInputs: getValidTextQueryInputs(),
			ExpectedShouldQuery: []map[string]interface{}{
				{
					"match": map[string]interface{}{
						"TITLE": "test-title",
					},
				},
				{
					"match": map[string]interface{}{
						"DESCRIPTION": "test-description",
					},
				},
			},
			ExpectedFilterQuery: []map[string]interface{}{
				{
					"match": map[string]interface{}{
						"AUTHOR": "test-author",
					},
				},
				{
					"match": map[string]interface{}{
						"CATEGORY": "test-category",
					},
				},
			},
			Error: nil,
		},
		{
			TestName: "TestCreateShouldAndFilterQueryValidCase: use shouldOrFilterQuery",
			TextQueryInputs: []*TextQueryInput{
				{
					Field: &searcher.OpenSearchSearcher_Field{
						ValueType:   "text",
						QueryType:   "filter",
						SourceField: "AUTHOR",
					},
					Value: "test-author",
				},
				{
					Field: &searcher.OpenSearchSearcher_Field{
						ValueType:       "text",
						QueryType:       "filter",
						SourceField:     "CATEGORY",
						SecondaryShould: true,
					},
					Value: "test-category",
				},
			},
			ExpectedShouldQuery: []map[string]interface{}{
				{
					"match": map[string]interface{}{
						"CATEGORY": "test-category",
					},
				},
			},
			ExpectedFilterQuery: []map[string]interface{}{
				{
					"match": map[string]interface{}{
						"AUTHOR": "test-author",
					},
				},
			},
			Error: nil,
		},
		{
			TestName: "TestCreateShouldAndFilterQueryInvalidCase: invalid queryType",
			TextQueryInputs: []*TextQueryInput{
				{
					Field: &searcher.OpenSearchSearcher_Field{
						ValueType:   "text",
						QueryType:   "filters",
						SourceField: "AUTHOR",
					},
					Value: "test-author",
				},
			},
			ExpectedShouldQuery: nil,
			ExpectedFilterQuery: nil,
			Error:               fmt.Errorf("invalid queryType: filters"),
		},
	}
	for _, tt := range cases {
		t.Run(tt.TestName, func(t *testing.T) {
			// use empty search request because it is not used in createShouldAndFilterQuery
			openSearchQueryGenerator := NewTextQueryStrategy()
			shouldQuery, filterQuery, err := openSearchQueryGenerator.createShouldAndFilterQuery(tt.TextQueryInputs)
			assert.Equal(t, tt.ExpectedShouldQuery, shouldQuery)
			assert.Equal(t, tt.ExpectedFilterQuery, filterQuery)
			assert.Equal(t, tt.Error, err)
		})
	}
}

func TestCreateMustNotQuery(t *testing.T) {
	textQueryInputs := getValidTextQueryInputs()
	cases := []struct {
		TestName        string
		Req             *QueryIngredients
		TextQueryInputs []*TextQueryInput
		Expected        []map[string]interface{}
	}{
		{
			TestName:        "TestCreateMustNotQuery: request has exclude",
			Req:             getValidTextSearchReq(),
			TextQueryInputs: textQueryInputs,
			Expected: []map[string]interface{}{
				{
					"match": map[string]interface{}{
						"TITLE": "test-exclude",
					},
				},
				{
					"match": map[string]interface{}{
						"DESCRIPTION": "test-exclude",
					},
				},
				{
					"match": map[string]interface{}{
						"AUTHOR": "test-exclude",
					},
				},
				{
					"match": map[string]interface{}{
						"CATEGORY": "test-exclude",
					},
				},
			},
		},
		{
			TestName: "TestCreateMustNotQuery: request does not have exclude",
			Req: &QueryIngredients{ // request does not have exclude
				Query: map[string]string{
					"title":       "test-title",
					"description": "test-description",
					"author":      "test-author",
					"category":    "test-category",
				},
			},
			TextQueryInputs: textQueryInputs,
			Expected:        []map[string]interface{}{},
		},
	}

	for _, tt := range cases {
		t.Run(tt.TestName, func(t *testing.T) {
			openSearchQueryGenerator := NewTextQueryStrategy()
			actual := openSearchQueryGenerator.createMustNotQuery(tt.Req, tt.TextQueryInputs)
			assert.Equal(t, tt.Expected, actual)
		})
	}
}

func TestCreateTextQueryInputs(t *testing.T) {

	cases := []struct {
		TestName      string
		Req           *QueryIngredients
		Config        *searcher.OpenSearchSearcher
		Expected      []*TextQueryInput
		ExpectedError error
	}{
		{
			TestName:      "TestCreateOpenSearchQueryInputs: request has title, description, and attribute query",
			Req:           getValidTextSearchReq(),
			Config:        getValidTextSearchConfig(),
			Expected:      getValidTextQueryInputs(), // For scalability, use pre-initialized TextQueryInputs
			ExpectedError: nil,
		},
	}
	for _, tt := range cases {
		t.Run(tt.TestName, func(t *testing.T) {
			actual, err := createTextQueryInputs(tt.Req, tt.Config.GetTextQueryStrategy())
			assert.NoError(t, err)
			if !assert.ElementsMatch(t, tt.Expected, actual) {
				t.Errorf("expected: %#v, actual: %v", tt.Expected, actual)
			}
		})
	}
}

func TestCreateRelevanceOrSoftmaxQuery(t *testing.T) {
	testCases := []struct {
		TestName     string
		ShouldQuery  []map[string]interface{}
		FilterQuery  []map[string]interface{}
		MustNotQuery []map[string]interface{}
		Expected     map[string]interface{}
	}{
		{
			TestName: "TestCreateRelevanceOrSoftmaxQuery: with filter and must not query",
			ShouldQuery: []map[string]interface{}{
				{
					"match": map[string]interface{}{
						"field1": "value1",
					},
				},
			},
			FilterQuery: []map[string]interface{}{
				{
					"range": map[string]interface{}{
						"REVW_CNT": map[string]interface{}{
							"gte": 10,
						},
					},
				},
			},
			MustNotQuery: []map[string]interface{}{
				{
					"match": map[string]interface{}{
						"field3": "value3",
					},
				},
			},
			Expected: map[string]interface{}{
				"query": map[string]interface{}{
					"bool": map[string]interface{}{
						"should": []map[string]interface{}{
							{
								"match": map[string]interface{}{
									"field1": "value1",
								},
							},
						},
						"filter": []map[string]interface{}{
							{
								"range": map[string]interface{}{
									"REVW_CNT": map[string]interface{}{
										"gte": 10,
									},
								},
							},
						},
						"must_not": []map[string]interface{}{
							{
								"match": map[string]interface{}{
									"field3": "value3",
								},
							},
						},
					},
				},
			},
		},
		{
			TestName: "TestCreateRelevanceOrSoftmaxQuery: without filter and must not query",
			ShouldQuery: []map[string]interface{}{
				{
					"match": map[string]interface{}{
						"field1": "value1",
					},
				},
			},
			FilterQuery:  []map[string]interface{}{},
			MustNotQuery: []map[string]interface{}{},
			Expected: map[string]interface{}{
				"query": map[string]interface{}{
					"bool": map[string]interface{}{
						"should": []map[string]interface{}{
							{
								"match": map[string]interface{}{
									"field1": "value1",
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.TestName, func(t *testing.T) {
			result := createRelevanceOrSoftmaxQuery(tc.ShouldQuery, tc.FilterQuery, tc.MustNotQuery)
			if !reflect.DeepEqual(result, tc.Expected) {
				t.Errorf("Expected result: %v, got: %v", tc.Expected, result)
			}
		})
	}
}

func TestCreateHarmonicQuery(t *testing.T) {
	testCases := []struct {
		TestName     string
		ShouldQuery  []map[string]interface{}
		FilterQuery  []map[string]interface{}
		MustNotQuery []map[string]interface{}
		Expected     map[string]interface{}
	}{
		{
			TestName: "TestCreateHarmonicQuery: with filter and must not query",
			ShouldQuery: []map[string]interface{}{
				{
					"match": map[string]interface{}{
						"field1": "value1",
					},
				},
			},
			FilterQuery: []map[string]interface{}{
				{
					"match": map[string]interface{}{
						"field2": "value2",
					},
				},
			},
			MustNotQuery: []map[string]interface{}{
				{
					"match": map[string]interface{}{
						"field3": "value3",
					},
				},
			},
			Expected: map[string]interface{}{
				"query": map[string]interface{}{
					"function_score": map[string]interface{}{
						"query": map[string]interface{}{
							"bool": map[string]interface{}{
								"should": []map[string]interface{}{
									{
										"match": map[string]interface{}{
											"field1": "value1",
										},
									},
								},
								"filter": []map[string]interface{}{
									{
										"match": map[string]interface{}{
											"field2": "value2",
										},
									},
									{
										"range": map[string]interface{}{
											"REVW_CNT": map[string]interface{}{
												"gte": 10,
											},
										},
									},
								},
								"must_not": []map[string]interface{}{
									{
										"match": map[string]interface{}{
											"field3": "value3",
										},
									},
								},
							},
						},
						"script_score": map[string]interface{}{
							"script": map[string]interface{}{
								"source": "2 * ((_score + 1) * doc['REVW_SCORE'].value) / ((_score + 1) + doc['REVW_SCORE'].value)",
							},
						},
					},
				},
			},
		},
		{
			TestName: "TestCreateHarmonicQuery: without filter and must not query",
			ShouldQuery: []map[string]interface{}{
				{
					"match": map[string]interface{}{
						"field1": "value1",
					},
				},
			},
			FilterQuery:  []map[string]interface{}{},
			MustNotQuery: []map[string]interface{}{},
			Expected: map[string]interface{}{
				"query": map[string]interface{}{
					"function_score": map[string]interface{}{
						"query": map[string]interface{}{
							"bool": map[string]interface{}{
								"should": []map[string]interface{}{
									{
										"match": map[string]interface{}{
											"field1": "value1",
										},
									},
								},
								"filter": []map[string]interface{}{
									{
										"range": map[string]interface{}{
											"REVW_CNT": map[string]interface{}{
												"gte": 10,
											},
										},
									},
								},
							},
						},
						"script_score": map[string]interface{}{
							"script": map[string]interface{}{
								"source": "2 * ((_score + 1) * doc['REVW_SCORE'].value) / ((_score + 1) + doc['REVW_SCORE'].value)",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.TestName, func(t *testing.T) {
			result := createHarmonicQuery(tc.ShouldQuery, tc.FilterQuery, tc.MustNotQuery)
			if !reflect.DeepEqual(result, tc.Expected) {
				t.Errorf("Expected result: %v, got: %v", tc.Expected, result)
			}
		})
	}
}

func TestCreateOpenSearchTextQuery(t *testing.T) {
	// create search config with invalid scoring mode
	invalidSearchConfig := getValidTextSearchConfig()
	invalidSearchConfig.GetTextQueryStrategy().ScoringMode = "invalid-scoring-mode"

	// create search request no exclude
	noExcludeReq := getValidTextSearchReq()
	noExcludeReq.Exclude = ""

	// create search request with no title field
	noTitleReq := getValidTextSearchReq()
	noTitleReq.Query["title"] = ""

	// create empty should config
	emptyShouldConfig := getValidTextSearchConfig()
	emptyShouldConfig.GetTextQueryStrategy().Fields["title"].QueryType = "filter"
	emptyShouldConfig.GetTextQueryStrategy().Fields["description"].QueryType = "filter"

	testCases := []struct {
		TestName      string
		Req           *QueryIngredients
		Config        *searcher.OpenSearchSearcher
		Expected      map[string]interface{}
		ExpectedError error
	}{
		{
			TestName: "TestCreateOpenSearchQueryValidCase",
			Req:      getValidTextSearchReq(),
			Config:   getValidTextSearchConfig(),
			Expected: map[string]interface{}{
				"query": map[string]interface{}{
					"bool": map[string]interface{}{
						"should": []map[string]interface{}{
							{
								"match": map[string]interface{}{
									"TITLE": "test-title",
								},
							},
							{
								"match": map[string]interface{}{
									"DESCRIPTION": "test-description",
								},
							},
						},
						"filter": []map[string]interface{}{
							{
								"match": map[string]interface{}{
									"AUTHOR": "test-author",
								},
							},
							{
								"match": map[string]interface{}{
									"CATEGORY": "test-category",
								},
							},
						},
						"must_not": []map[string]interface{}{
							{
								"match": map[string]interface{}{
									"TITLE": "test-exclude",
								},
							},
							{
								"match": map[string]interface{}{
									"DESCRIPTION": "test-exclude",
								},
							},
							{
								"match": map[string]interface{}{
									"AUTHOR": "test-exclude",
								},
							},
							{
								"match": map[string]interface{}{
									"CATEGORY": "test-exclude",
								},
							},
						},
					},
				},
			},
			ExpectedError: nil,
		},
		{
			TestName: "TestCreateOpenSearchQueryValidCase: no exclude",
			Req:      noExcludeReq,
			Config:   getValidTextSearchConfig(),
			Expected: map[string]interface{}{
				"query": map[string]interface{}{
					"bool": map[string]interface{}{
						"should": []map[string]interface{}{
							{
								"match": map[string]interface{}{
									"TITLE": "test-title",
								},
							},
							{
								"match": map[string]interface{}{
									"DESCRIPTION": "test-description",
								},
							},
						},
						"filter": []map[string]interface{}{
							{
								"match": map[string]interface{}{
									"AUTHOR": "test-author",
								},
							},
							{
								"match": map[string]interface{}{
									"CATEGORY": "test-category",
								},
							},
						},
					},
				},
			},
			ExpectedError: nil,
		},
		{
			TestName: "TestCreateOpenSearchQueryValidCase: no title",
			Req:      noTitleReq,
			Config:   getValidTextSearchConfig(),
			Expected: map[string]interface{}{
				"query": map[string]interface{}{
					"bool": map[string]interface{}{
						"should": []map[string]interface{}{
							{
								"match": map[string]interface{}{
									"DESCRIPTION": "test-description",
								},
							},
						},
						"filter": []map[string]interface{}{
							{
								"match": map[string]interface{}{
									"AUTHOR": "test-author",
								},
							},
							{
								"match": map[string]interface{}{
									"CATEGORY": "test-category",
								},
							},
						},
						"must_not": []map[string]interface{}{
							{
								"match": map[string]interface{}{
									"TITLE": "test-exclude",
								},
							},
							{
								"match": map[string]interface{}{
									"DESCRIPTION": "test-exclude",
								},
							},
							{
								"match": map[string]interface{}{
									"AUTHOR": "test-exclude",
								},
							},
							{
								"match": map[string]interface{}{
									"CATEGORY": "test-exclude",
								},
							},
						},
					},
				},
			},
			ExpectedError: nil,
		},
		{
			TestName: "TestCreateOpenSearchQueryValidCase: empty request",
			Req:      &QueryIngredients{},
			Config:   getValidTextSearchConfig(),
			Expected: map[string]interface{}{
				"query": map[string]interface{}{
					"bool": map[string]interface{}{
						"should": []map[string]interface{}{},
					},
				},
			},
			ExpectedError: nil,
		},
		{
			TestName: "TestCreateOpenSearchQueryValidCase: empty should in search config",
			Req:      getValidTextSearchReq(),
			Config:   emptyShouldConfig,
			Expected: map[string]interface{}{
				"query": map[string]interface{}{
					"bool": map[string]interface{}{
						"should": []map[string]interface{}{
							{
								"match": map[string]interface{}{
									"CATEGORY": "test-category",
								},
							},
						},
						"filter": []map[string]interface{}{
							{
								"match": map[string]interface{}{
									"TITLE": "test-title",
								},
							},
							{
								"match": map[string]interface{}{
									"DESCRIPTION": "test-description",
								},
							},
							{
								"match": map[string]interface{}{
									"AUTHOR": "test-author",
								},
							},
						},
						"must_not": []map[string]interface{}{
							{
								"match": map[string]interface{}{
									"TITLE": "test-exclude",
								},
							},
							{
								"match": map[string]interface{}{
									"DESCRIPTION": "test-exclude",
								},
							},
							{
								"match": map[string]interface{}{
									"AUTHOR": "test-exclude",
								},
							},
							{
								"match": map[string]interface{}{
									"CATEGORY": "test-exclude",
								},
							},
						},
					},
				},
			},
			ExpectedError: nil,
		},
		{
			TestName:      "TestCreateOpenSearchQueryInvalidCase: invalid scoring mode",
			Req:           getValidTextSearchReq(),
			Config:        invalidSearchConfig,
			Expected:      nil,
			ExpectedError: fmt.Errorf("Invalid scoring mode: invalid-scoring-mode"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.TestName, func(t *testing.T) {
			queryStrategy := NewTextQueryStrategy()
			result, err := queryStrategy.CreateOpenSearchQuery(tc.Req, tc.Config)
			if query, ok := tc.Expected["query"].(map[string]interface{}); ok {
				if boolQuery, ok := query["bool"].(map[string]interface{}); ok {
					if should, ok := boolQuery["should"].([]map[string]interface{}); ok {
						if !assert.ElementsMatch(t, should, result["query"].(map[string]interface{})["bool"].(map[string]interface{})["should"].([]map[string]interface{})) {
							t.Errorf("Expected: %v, got: %v", should, result["query"].(map[string]interface{})["bool"].(map[string]interface{})["should"].([]map[string]interface{}))
						}
					} else if filter, ok := boolQuery["filter"].([]map[string]interface{}); ok {
						if !assert.ElementsMatch(t, filter, result["query"].(map[string]interface{})["bool"].(map[string]interface{})["filter"].([]map[string]interface{})) {
							t.Errorf("Expected: %v, got: %v", filter, result["query"].(map[string]interface{})["bool"].(map[string]interface{})["filter"].([]map[string]interface{}))
						}
					} else if mustNot, ok := boolQuery["must_not"].([]map[string]interface{}); ok {
						if !assert.ElementsMatch(t, mustNot, result["query"].(map[string]interface{})["bool"].(map[string]interface{})["must_not"].([]map[string]interface{})) {
							t.Errorf("Expected: %v, got: %v", mustNot, result["query"].(map[string]interface{})["bool"].(map[string]interface{})["must_not"].([]map[string]interface{}))
						}
					}
				}
			}

			if (err == nil && tc.ExpectedError != nil) || (err != nil && tc.ExpectedError == nil) {
				t.Errorf("Expected error: %v, got: %v", tc.ExpectedError, err)
			}
		})
	}
}
