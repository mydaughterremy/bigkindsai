package function

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getMockSpellCorrectionPlugin() *SpellingCorrectionPlugin {
	return &SpellingCorrectionPlugin{}
}

func TestSpellingCorrectionService_Call(t *testing.T) {
	result, err := getMockSpellCorrectionPlugin().Call(context.Background(), nil, nil)
	assert.Equal(t, IndependentCallError, err)
	assert.Nil(t, result)
}
