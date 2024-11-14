package token_counter

import (
	"testing"

	"bigkinds.or.kr/conversation/model"
	"github.com/stretchr/testify/assert"
)

type TestCase struct {
	Function model.Function
	Expected string
}

func TestFunctionDefinitionToSystemPrompt(t *testing.T) {
	cases := []TestCase{{
		Function: model.Function{
			Name:        "get_current_weather",
			Description: `Get the current weather in a given location`,
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "The city and state, e.g. San Francisco, CA",
					},
					"unit": map[string]interface{}{
						"type": "string",
						"enum": []string{"celsius", "fahrenheit"},
					},
				},
				"required": []string{"location"},
			},
		},
		Expected: `// Get the current weather in a given location
type get_current_weather = (_: {
// The city and state, e.g. San Francisco, CA
location: string,
unit?: "celsius" | "fahrenheit",
}) => any;`,
	},
		{
			Function: model.Function{
				Name: "search",
				Description: `Use this function to get external knowledge to answer information questions. e.g. "~가 뭐야?", "~에 대해 알려줘", "~는 무엇인가요?".

<example>
question: "최근 애플 근황 뉴스"
{{
	"standalone_query": "애플",
	"published_date_range": {{
	"start_date": "2006-01-02T15:04:05-07:00",
	"end_date": "2006-01-02T15:04:05-07:00"
	}}
}}
</example>
When the user's question pertains to statistics, and the data is typically published after the end of the year, you should set the 'start_date' to the next year's first day and 'end_date' to the current year's last day.
    <example>
    question: "2010년부터 2015년 출산율 알려줘"
    {{
    "standalone_query": "출산율",
    "published_date_range": {{
    "start_date": "2006-01-02T15:04:05-07:00",
    "end_date": "2006-01-02T15:04:05-07:00"
    }}
    }}
    </example>
If the user's query does not specify a time period explicitly, you can leave the 'publish_date_range' field empty.
`,
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"standalone_query": map[string]interface{}{
							"type":        "string",
							"description": "The comprehensive query that considers the context of the conversation history.",
						},
						"published_date_range": map[string]interface{}{
							"type": "object",
							"description": `This field defines the time frame for filtering articles based on their publication dates. 
					Extract time-related expressions from the user's query to determine the appropriate date range.
					The term "recent" implies a date range extending back one month from today's date.
					This field should be filled based on the user's query and has no default value.`,
							"properties": map[string]interface{}{
								"start_date": map[string]interface{}{
									"type":        "string",
									"description": "The start date or minimum publication date for articles to be searched.",
								},
								"end_date": map[string]interface{}{
									"type":        "string",
									"description": "The end date or maximum publication date for articles to be searched.",
								},
							},
						},
					},
					"required": []string{"standalone_query"},
				},
			},
			Expected: `// Use this function to get external knowledge to answer information questions. e.g. "~가 뭐야?", "~에 대해 알려줘", "~는 무엇인가요?".
//
// <example>
// question: "최근 애플 근황 뉴스"
// {{
// "standalone_query": "애플",
// "published_date_range": {{
// "start_date": "2006-01-02T15:04:05-07:00",
// "end_date": "2006-01-02T15:04:05-07:00"
// }}
// }}
// </example>
// When the user's question pertains to statistics, and the data is typically published after the end of the year, you should set the 'start_date' to the next year's first day and 'end_date' to the current year's last day.
// <example>
// question: "2010년부터 2015년 출산율 알려줘"
// {{
// "standalone_query": "출산율",
// "published_date_range": {{
// "start_date": "2006-01-02T15:04:05-07:00",
// "end_date": "2006-01-02T15:04:05-07:00"
// }}
// }}
// </example>
// If the user's query does not specify a time period explicitly, you can leave the 'publish_date_range' field empty.
type search = (_: {
// This field defines the time frame for filtering articles based on their publication dates.
// Extract time-related expressions from the user's query to determine the appropriate date range.
// The term "recent" implies a date range extending back one month from today's date.
// This field should be filled based on the user's query and has no default value.
published_date_range?: {
// The end date or maximum publication date for articles to be searched.
end_date?: string,
// The start date or minimum publication date for articles to be searched.
start_date?: string,
},
// The comprehensive query that considers the context of the conversation history.
standalone_query: string,
}) => any;`,
		}}

	for _, c := range cases {
		got := FunctionDefinitionToSystemPrompt(c.Function)
		assert.Equal(t, c.Expected, got)
	}
}
