package function

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getMockSummarizePlugin() *SummarizePlugin {
	return &SummarizePlugin{}
}

func TestSummarizePlugin_Call(t *testing.T) {
	result, err := getMockSummarizePlugin().Call(context.Background(), nil, nil)
	assert.Equal(t, IndependentCallError, err)
	assert.Nil(t, result)
}
