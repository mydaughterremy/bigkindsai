package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertArgumentsToUTF8IfNot(t *testing.T) {
	testCases := []struct {
		name               string
		sargs              string
		newStandaloneQuery string
		expected           string
		expectError        bool
	}{
		{
			name:        "valid utf8",
			sargs:       "{\"standalone_query\":\"최근 손흥민 기사\"}",
			expected:    "{\"standalone_query\":\"최근 손흥민 기사\"}",
			expectError: false,
		},
		{
			name:               "invalid utf8",
			sargs:              "{\"standalone_query\":\"ì\x9c¤ì\x84\x9dì\x97´ ì\xa0\x95ì±\x85\"}",
			newStandaloneQuery: "안녕",
			expected:           "{\"standalone_query\":\"윤석열 정책\"}",
			expectError:        false,
		},
		{
			name:               "invalid latin-1 string",
			sargs:              "{\"standalone_query\":\"ì\xbd\xb2\x3d\xbc\x20\xe2\x8c\x98\"}",
			newStandaloneQuery: "최근 손흥민 기사 알려줄래?",
			expected:           "{\"standalone_query\":\"최근 손흥민 기사 알려줄래?\"}",
			expectError:        false,
		},
		{
			name:        "error string",
			sargs:       "{\"standalone_query\":\"ì\xbd\xb2\x3d\xbc\x20\xe2\x8c\x98\"",
			expected:    "",
			expectError: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			convertedString, err := convertArgumentsToUTF8IfNot(test.sargs, test.newStandaloneQuery)
			assert.Equal(t, test.expected, convertedString)
			if test.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}

}
