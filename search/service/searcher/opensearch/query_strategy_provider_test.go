package opensearch

import (
	"testing"

	"bigkinds.or.kr/search/service/searcher/opensearch/query_strategy"
)

func TestNewQueryStrategyProvider(t *testing.T) {
	cases := []struct {
		TestName              string
		Key                   string
		ExpectedQueryStrategy query_strategy.QueryStrategy
		Error                 error
	}{
		{
			TestName:              "TestTextQueryStrategy",
			Key:                   "text",
			ExpectedQueryStrategy: query_strategy.NewTextQueryStrategy(),
		},
		{
			TestName:              "TestInvalidQueryStrategy",
			Key:                   "invalid",
			ExpectedQueryStrategy: nil,
		},
	}

	// init query strategy provider
	queryStrategyProvider, err := NewQueryStrategyProvider()
	if err != nil {
		t.Errorf("error creating query strategy provider: %v", err)
	}
	for _, tt := range cases {
		t.Run(tt.TestName, func(t *testing.T) {
			queryStrategy := queryStrategyProvider.Get(tt.Key)
			if queryStrategy != tt.ExpectedQueryStrategy {
				t.Errorf("query strategy is not as expected. expected: %v, actual: %v", tt.ExpectedQueryStrategy, queryStrategy)
			}
		})
	}

}
