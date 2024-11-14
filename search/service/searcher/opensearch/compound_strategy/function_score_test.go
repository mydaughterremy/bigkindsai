package compound_strategy

import (
	"fmt"
	"testing"

	"bigkinds.or.kr/proto/searcher"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestCreateCompoundQuery(t *testing.T) {
	cases := []struct {
		TestName         string
		Query            map[string]interface{}
		CompoundStrategy *searcher.OpenSearchSearcher_CompoundStrategy
		Error            error
		Expected         map[string]interface{}
	}{
		{
			TestName: "with functions",
			Query: map[string]interface{}{
				"query": map[string]interface{}{
					"match": map[string]interface{}{
						"message": "this is a test",
					},
				},
			},
			CompoundStrategy: &searcher.OpenSearchSearcher_CompoundStrategy{
				CompoundStrategy: &searcher.OpenSearchSearcher_CompoundStrategy_FunctionScore_{
					FunctionScore: &searcher.OpenSearchSearcher_CompoundStrategy_FunctionScore{
						Functions: []*structpb.Struct{
							{
								Fields: map[string]*structpb.Value{
									"weight": {
										Kind: &structpb.Value_NumberValue{
											NumberValue: 1,
										},
									},
								},
							},
						},
					},
				},
			},
			Error: nil,
			Expected: map[string]interface{}{
				"query": map[string]interface{}{
					"function_score": map[string]interface{}{
						"query": map[string]interface{}{
							"match": map[string]interface{}{
								"message": "this is a test",
							},
						},
						"functions": []*structpb.Struct{
							{
								Fields: map[string]*structpb.Value{
									"weight": {
										Kind: &structpb.Value_NumberValue{
											NumberValue: 1,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			TestName: "without functions",
			Query: map[string]interface{}{
				"match": map[string]interface{}{
					"message": "this is a test",
				},
			},
			CompoundStrategy: &searcher.OpenSearchSearcher_CompoundStrategy{
				CompoundStrategy: &searcher.OpenSearchSearcher_CompoundStrategy_FunctionScore_{
					FunctionScore: &searcher.OpenSearchSearcher_CompoundStrategy_FunctionScore{
						Functions: []*structpb.Struct{},
					},
				},
			},
			Error:    fmt.Errorf("no functions provided"),
			Expected: nil,
		},
	}

	functionScore := NewFunctionScoreCompoundStrategy()

	for _, tt := range cases {
		t.Run(tt.TestName, func(t *testing.T) {
			actual, err := functionScore.CreateCompoundQuery(tt.Query, tt.CompoundStrategy)
			assert.Equal(t, tt.Error, err)
			assert.Equal(t, tt.Expected, actual)
		})
	}
}
