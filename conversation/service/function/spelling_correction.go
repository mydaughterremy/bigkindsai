package function

import (
	"context"

	"bigkinds.or.kr/conversation/model"
)

type SpellingCorrectionPlugin struct {
}

func (p *SpellingCorrectionPlugin) Definition() model.Function {
	return model.Function{
		Name: "spelling_correction",
		Parameters: map[string]interface{}{
			"type":        "object",
			"description": "This function is called when spelling correction is needed.",
			"properties":  map[string]interface{}{},
		},
	}
}

func (p *SpellingCorrectionPlugin) Call(ctx context.Context, arguments map[string]interface{}, extraArgs *ExtraArgs) ([]byte, error) {
	return nil, IndependentCallError
}
