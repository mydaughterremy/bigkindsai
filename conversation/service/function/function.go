package function

import (
	"bigkinds.or.kr/conversation/model"
	"bigkinds.or.kr/pkg/utils"
	"context"
	"encoding/json"
	"errors"
)

var IndependentCallError = errors.New("this function should be called independently not as a part of a pipeline")

type GPTFunction interface {
	Definition() model.Function
	Call(ctx context.Context, arguments map[string]interface{}, extraArgs *ExtraArgs) ([]byte, error)
}

type ExtraArgs struct {
	RawQuery string `json:"raw_query"`
	Provider string `json:"provider"`
}

type FunctionService struct {
}

func parseFunctionArguments(sargs string) (map[string]interface{}, error) {
	var margs map[string]interface{}
	err := json.Unmarshal([]byte(sargs), &margs)

	return margs, err
}

func (s *FunctionService) CallFunction(ctx context.Context, name string, sargs string, functions []GPTFunction, extraArgs *ExtraArgs) ([]byte, error) {
	for _, function := range functions {
		def := function.Definition()
		if def.Name == name {
			margs, err := parseFunctionArguments(sargs)
			if err != nil {
				return nil, err
			}
			return function.Call(ctx, margs, extraArgs)
		}
	}
	return nil, errors.New("function not found")
}

func (s *FunctionService) ListFunctions(currentTime utils.CurrentTime) []GPTFunction {
	return []GPTFunction{
		&SearchPlugin{
			currentTime: currentTime,
		},
		&TranslatePlugin{},
		&SummarizePlugin{},
		&SpellingCorrectionPlugin{},
	}
}
