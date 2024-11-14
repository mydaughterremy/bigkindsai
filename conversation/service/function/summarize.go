package function

import (
	"context"

	"bigkinds.or.kr/conversation/model"
)

type SummarizePlugin struct {
}

func (p *SummarizePlugin) Definition() model.Function {
	return model.Function{
		Name: "summarize",
		Parameters: map[string]interface{}{
			"type":        "object",
			"description": "This function is called when summarization is needed.",
			"properties":  map[string]interface{}{},
		},
	}
}

func (p *SummarizePlugin) Call(ctx context.Context, arguments map[string]interface{}, extraArgs *ExtraArgs) ([]byte, error) {
	return nil, IndependentCallError
}
