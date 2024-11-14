package function

import (
	"context"

	"bigkinds.or.kr/conversation/model"
)

type TranslatePlugin struct {
}

func (p *TranslatePlugin) Definition() model.Function {
	return model.Function{
		Name: "translate",
		Parameters: map[string]interface{}{
			"type":        "object",
			"description": "This function is called when translation is needed.",
			"properties": map[string]interface{}{
				"target_language": map[string]interface{}{
					"type":        "string",
					"description": "Desired language for translation",
				},
			},
			"required": []string{"target_language"},
		},
	}
}

func (p *TranslatePlugin) Call(ctx context.Context, arguments map[string]interface{}, extraArgs *ExtraArgs) ([]byte, error) {
	return nil, IndependentCallError
}
