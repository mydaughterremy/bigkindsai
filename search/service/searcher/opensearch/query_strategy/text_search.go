package query_strategy

import (
	"errors"
	"fmt"

	"bigkinds.or.kr/proto/searcher"
)

type TextQueryStrategy struct {
}

type TextQueryInput struct {
	Field *searcher.OpenSearchSearcher_Field
	Value string
}

// createShouldAndFilterQuery processes the query arguments from the request and returns the "should" and "filter" queries
func (k *TextQueryStrategy) createShouldAndFilterQuery(textQueryInputs []*TextQueryInput) ([]map[string]interface{}, []map[string]interface{}, error) {
	shouldQuery := make([]map[string]interface{}, 0)          // queryType == "should"
	filterQuery := make([]map[string]interface{}, 0)          // queryType == "filter"
	secondaryShouldQuery := make([]map[string]interface{}, 0) // queryType == "filter" and secondary_should = true

	for _, textQueryInput := range textQueryInputs {
		field := textQueryInput.Field
		queryType := field.QueryType     // queryType is the query type of the field in the request. The value is one of "should", "filter"
		fieldVal := textQueryInput.Value // fieldVal is the value of the field in the request ex) req.Title, req.Exclude

		// if fieldVal is empty, skip
		if fieldVal == "" {
			continue
		}

		// create match query for 'shouldQuery' and 'filterQuery'
		matchQuery := map[string]interface{}{
			"match": map[string]interface{}{
				field.SourceField: fieldVal,
			},
		}

		// create 'shouldQuery' and 'filterQuery' query, and 'secondaryShouldQuery' if it exists
		switch queryType {
		case "should":
			shouldQuery = append(shouldQuery, matchQuery)
		case "filter":
			if field.SecondaryShould {
				secondaryShouldQuery = append(secondaryShouldQuery, matchQuery)
			} else {
				filterQuery = append(filterQuery, matchQuery)
			}
		default:
			return nil, nil, fmt.Errorf("invalid queryType: %s", queryType)
		}
	}

	// process 'secondaryShouldQuery'
	if len(shouldQuery) == 0 { // if length of 'shouldQuery' is 0, 'secondaryShouldQuery' should be added to 'shouldQuery'
		shouldQuery = append(shouldQuery, secondaryShouldQuery...)
	} else { // if length of 'shouldQuery' is not 0, 'secondaryShouldQuery' should be added to 'filterQuery'
		filterQuery = append(filterQuery, secondaryShouldQuery...)
	}

	return shouldQuery, filterQuery, nil
}

// createMustNotQuery processes the exclude argument from the request and returns the "must_not" query
func (k *TextQueryStrategy) createMustNotQuery(req *QueryIngredients, textQueryInputs []*TextQueryInput) []map[string]interface{} {
	mustNotQuery := make([]map[string]interface{}, 0)
	if req.Exclude != "" {
		for _, textQueryInput := range textQueryInputs {
			field := textQueryInput.Field
			mustNotQuery = append(mustNotQuery, map[string]interface{}{
				"match": map[string]interface{}{
					field.SourceField: req.Exclude,
				},
			})
		}
	}
	return mustNotQuery
}

func createRelevanceOrSoftmaxQuery(shouldQuery []map[string]interface{}, filterQuery []map[string]interface{}, mustNotQuery []map[string]interface{}) map[string]interface{} {
	// TODO: deal with len(shouldQuery) == 0 (all field is empty)
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"should": shouldQuery,
			},
		},
	}
	if len(filterQuery) > 0 {
		query["query"].(map[string]interface{})["bool"].(map[string]interface{})["filter"] = filterQuery
	}

	if len(mustNotQuery) > 0 {
		query["query"].(map[string]interface{})["bool"].(map[string]interface{})["must_not"] = mustNotQuery
	}
	return query
}

func createHarmonicQuery(shouldQuery []map[string]interface{}, filterQuery []map[string]interface{}, mustNotQuery []map[string]interface{}) map[string]interface{} {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"function_score": map[string]interface{}{
				"query": map[string]interface{}{
					"bool": map[string]interface{}{
						"should": shouldQuery,
					},
				},
				"script_score": map[string]interface{}{
					"script": map[string]interface{}{
						"source": "2 * ((_score + 1) * doc['REVW_SCORE'].value) / ((_score + 1) + doc['REVW_SCORE'].value)",
					},
				},
			},
		},
	}
	filterQuery = append(filterQuery, map[string]interface{}{
		"range": map[string]interface{}{
			"REVW_CNT": map[string]interface{}{
				"gte": 10,
			},
		},
	})
	if len(filterQuery) > 0 {
		query["query"].(map[string]interface{})["function_score"].(map[string]interface{})["query"].(map[string]interface{})["bool"].(map[string]interface{})["filter"] = filterQuery
	}

	if len(mustNotQuery) > 0 {
		query["query"].(map[string]interface{})["function_score"].(map[string]interface{})["query"].(map[string]interface{})["bool"].(map[string]interface{})["must_not"] = mustNotQuery
	}
	return query
}

// createOpenSearchQuery creates the OpenSearch query from the search request and search config
func (k *TextQueryStrategy) CreateOpenSearchQuery(req *QueryIngredients, osSearcherConfig *searcher.OpenSearchSearcher) (map[string]interface{}, error) {
	textQueryStrategy := osSearcherConfig.GetTextQueryStrategy()
	textQueryInputs, err := createTextQueryInputs(req, textQueryStrategy)
	if err != nil {
		return nil, err
	}

	// create "should" and "filter" queries
	shouldQuery, filterQuery, err := k.createShouldAndFilterQuery(textQueryInputs)
	if err != nil {
		return nil, err
	}

	// create "must_not" query
	mustNotQuery := k.createMustNotQuery(req, textQueryInputs)

	// create opensearch query
	var opensearchQuery map[string]interface{}

	switch textQueryStrategy.ScoringMode { // create opensearch query based on scoring mode
	case "relevance", "softmax":
		opensearchQuery = createRelevanceOrSoftmaxQuery(shouldQuery, filterQuery, mustNotQuery)
	case "harmonic":
		opensearchQuery = createHarmonicQuery(shouldQuery, filterQuery, mustNotQuery)
	default:
		return nil, fmt.Errorf("Invalid scoring mode: %s", textQueryStrategy.ScoringMode)
	}

	return opensearchQuery, nil
}

// createTextQueryInput creates a map of search fields from the search config
func createTextQueryInputs(req *QueryIngredients, textQueryStrategy *searcher.OpenSearchSearcher_TextQueryStrategy) ([]*TextQueryInput, error) {
	textQueryInputs := make([]*TextQueryInput, 0)
	for key, field := range textQueryStrategy.Fields {
		// if source field is not specified, set it to the key
		if field.SourceField == "" {
			field.SourceField = key
		}

		if field.QueryType == "should" && field.SecondaryShould {
			return nil, errors.New("'secondary_should' cannot be true for query_type should")
		}
		textQueryInputs = append(textQueryInputs, &TextQueryInput{
			Field: field,
			Value: req.Query[key],
		})
	}
	return textQueryInputs, nil
}

func NewTextQueryStrategy() *TextQueryStrategy {
	return &TextQueryStrategy{}
}
