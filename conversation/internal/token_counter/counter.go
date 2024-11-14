package token_counter

import (
	"bigkinds.or.kr/conversation/model"
	tiktoken "github.com/pkoukk/tiktoken-go"
)

type TokenCounter struct {
	tokenizer *tiktoken.Tiktoken
}

func NewTokenCounter(tokenizer *tiktoken.Tiktoken) *TokenCounter {
	return &TokenCounter{
		tokenizer: tokenizer,
	}
}

func (c *TokenCounter) CountTokens(text string) int {
	return len(c.tokenizer.Encode(text, nil, nil))
}

func (c *TokenCounter) CountFunctionInputTokens(function model.Function) int {
	const FUNCTION_OVERHEAD = 12
	return FUNCTION_OVERHEAD + len(c.tokenizer.Encode(FunctionDefinitionToSystemPrompt(function), nil, nil))
}

func (c *TokenCounter) CountFunctionOutputTokens(output string) int {
	return len(c.tokenizer.Encode(output, nil, nil))
}
