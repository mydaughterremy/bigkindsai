package compound_strategy

import (
	"fmt"

	"bigkinds.or.kr/proto/searcher"
)

type FunctionScoreCompoundStrategy struct {
}

func (f *FunctionScoreCompoundStrategy) CreateCompoundQuery(query map[string]interface{}, compoundStrategy *searcher.OpenSearchSearcher_CompoundStrategy) (map[string]interface{}, error) {
	functions := compoundStrategy.GetFunctionScore().GetFunctions()
	if len(functions) == 0 {
		return nil, fmt.Errorf("no functions provided")
	}
	query, ok := query["query"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no query provided")
	}
	functionScore := map[string]interface{}{
		"query": map[string]interface{}{
			"function_score": map[string]interface{}{
				"query":     query,
				"functions": functions,
			},
		},
	}
	return functionScore, nil
}

func NewFunctionScoreCompoundStrategy() *FunctionScoreCompoundStrategy {
	return &FunctionScoreCompoundStrategy{}
}
