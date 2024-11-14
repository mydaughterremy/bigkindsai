package function

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func getMockTranslatePlugin() *TranslatePlugin {
	return &TranslatePlugin{}
}

func TestTranslatePlugin_Call(t *testing.T) {
	result, err := getMockTranslatePlugin().Call(context.Background(), nil, nil)
	assert.Equal(t, IndependentCallError, err)
	assert.Nil(t, result)
}
