package gpt_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"bigkinds.or.kr/pkg/chat/v2"
	"bigkinds.or.kr/pkg/chat/v2/gpt"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
)

func TestUnmarshalRawString(t *testing.T) {
	msg := `{"title": "","description": "","author": "","category": "소설","assistant_message": "스릴러 소설을 추천해드릴게요.","exclude": ""}`
	var param gpt.ChatCompletionFunctionCallParams

	msg_bytes := []byte(msg)

	err := json.Unmarshal(msg_bytes, &param)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "스릴러 소설을 추천해드릴게요.", param.AssistantMessage)
}

func TestGPTHttpRequest(t *testing.T) {
	const (
		apiKey       = "test-api-key"
		staticOutput = `{
			"id": "chatcmpl-123",
			"object": "chat.completion",
			"created": 1677652288,
			"model": "gpt-3.5-turbo-0613",
			"choices": [{
			  "index": 0,
			  "message": {
				"role": "assistant",
				"content": "\n\nHello there, how may I assist you today?"
			  },
			  "finish_reason": "stop"
			}],
			"usage": {
			  "prompt_tokens": 9,
			  "completion_tokens": 12,
			  "total_tokens": 21
			}
		  }`
	)

	cases := []struct {
		TestName string
		// Option       *gpt.GptPredictionOptions
		Context          context.Context
		GptOption        []func(*chat.GptOptions)
		PredictionOption []func(*chat.GptPredictionOptions)
		ChatMessages     []*chat.ChatPayload
		ExpectedReq      string
		ExpectedErr      error
	}{
		{
			TestName: "TestGPTHttpRequestWithOptions",
			Context:  context.Background(),
			GptOption: []func(*chat.GptOptions){
				chat.WithKey(apiKey),
			},
			PredictionOption: []func(*chat.GptPredictionOptions){
				chat.WithModel("gpt-3.5-turbo-0613"),
				chat.WithFunctions([]string{
					`{"name": "get_current_weather"}`,
				}),
				chat.WithFunctionCall("auto"),
				chat.WithTemperature(0.1),
				chat.WithTopP(0.2),
				chat.WithMaxTokens(500),
				chat.WithPresencePenalty(0.3),
				chat.WithFrequencyPenalty(0.4),
			},
			ChatMessages: []*chat.ChatPayload{
				{
					Role:    "system",
					Content: "System Prompt",
				},
				{
					Role:    "user",
					Content: "Message1",
				},
				{
					Role:    "assistant",
					Content: "Message2",
				},
				{
					Role:    "user",
					Content: "Message3",
				},
			},
			ExpectedReq: `{
				"model": "gpt-3.5-turbo-0613",
				"messages": [
				  {
					"role": "system",
					"content": "System Prompt"
				  },
				  {
					"role": "user",
					"content": "Message1"
				  },
				  {
					"role": "assistant",
					"content": "Message2"
				  },
				  {
					"role": "user",
					"content": "Message3"
				  }
				],
				"functions": [
				  {"name": "get_current_weather"}
				],
				"function_call": "auto",
				"temperature": 0.1,
				"top_p": 0.2,
				"max_tokens": 500,
				"presence_penalty": 0.3,
				"frequency_penalty": 0.4
			}`,
		},
		{
			TestName: "TestGPTHttpRequestCustomEndpoint",
			Context:  metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{"X-Forwarded-For-Foo": "Bar"})),
			GptOption: []func(*chat.GptOptions){
				chat.WithCustomEndpoint("http://upstage.ai"),
				chat.WithModels([]string{
					"up-llm-1",
					"up-llm-2",
				}),
			},
			PredictionOption: []func(*chat.GptPredictionOptions){
				chat.WithModel("up-llm-1"),
				chat.WithFunctions([]string{
					`{"name": "my_function"}`,
				}),
				chat.WithFunctionCall(`{"name": "my_function"}`),
				chat.WithTemperature(0.2),
				chat.WithTopP(0.4),
				chat.WithMaxTokens(1024),
				chat.WithPresencePenalty(0.6),
				chat.WithFrequencyPenalty(0.8),
			},
			ChatMessages: []*chat.ChatPayload{
				{
					Role:    "system",
					Content: "System Prompt",
				},
				{
					Role:    "user",
					Content: "Message1",
				},
				{
					Role:    "assistant",
					Content: "Message2",
				},
				{
					Role:    "user",
					Content: "Message3",
				},
			},
			ExpectedReq: `{
				"model": "up-llm-1",
				"messages": [
				  {
					"role": "system",
					"content": "System Prompt"
				  },
				  {
					"role": "user",
					"content": "Message1"
				  },
				  {
					"role": "assistant",
					"content": "Message2"
				  },
				  {
					"role": "user",
					"content": "Message3"
				  }
				],
				"functions": [
				  {"name": "my_function"}
				],
				"function_call": {"name": "my_function"},
				"temperature": 0.2,
				"top_p": 0.4,
				"max_tokens": 1024,
				"presence_penalty": 0.6,
				"frequency_penalty": 0.8
			}`,
		},
	}

	for _, tt := range cases {
		t.Run(tt.TestName, func(t *testing.T) {
			// Create Http Client for Test
			roundTripper := NewTestRoundTripper(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(staticOutput)),
				}, nil
			})
			testHttpClient := NewTestClient(roundTripper)

			// create Model
			gptModel, err := gpt.NewGPTCompitable(testHttpClient, chat.WithKey(apiKey))
			if err != nil {
				if err != nil && tt.ExpectedErr == nil || err == nil && tt.ExpectedErr != nil {
					t.Fatalf("Expected error %v, got %v", tt.ExpectedErr, err)
				}
				return
			}

			// perform Chat
			_, err = gptModel.Chat(tt.Context, tt.ChatMessages, tt.PredictionOption...)
			if err != nil {
				if err != nil && tt.ExpectedErr == nil || err == nil && tt.ExpectedErr != nil {
					t.Fatalf("Expected error %v, got %v", tt.ExpectedErr, err)
				}
			}

			// chcek header
			header := roundTripper.Req[0].Header
			md, ok := metadata.FromIncomingContext(tt.Context)
			if ok {
				for k, v := range md {
					assert.Equal(t, header.Get(strings.TrimPrefix(k, chat.ForwardPrefix)), v[0])
				}
			}

			// check request
			if len(tt.ExpectedReq) > 0 {
				assert.Equal(t, 1, len(roundTripper.Req))
				reqBytes, err := io.ReadAll(roundTripper.Req[0].Body)
				if err != nil {
					t.Fatal(err)
				}
				reqBody := string(reqBytes)

				// compare compacted json
				reqBodyCompact, err := compactJson(reqBody)
				if err != nil {
					t.Fatal(err)
				}

				expectedReqCompact, err := compactJson(tt.ExpectedReq)
				if err != nil {
					t.Fatal(err)
				}

				assert.Equal(t, expectedReqCompact, reqBodyCompact)
			}
		})
	}
}

func TestUnmarshalEscapeString(t *testing.T) {
	msg := "{\n  \"title\": \"\",\n  \"description\": \"\",\n  \"author\": \"\",\n  \"category\": \"소설\",\n  \"assistant_message\": \"스릴러 소설을 추천해드릴게요.\",\n  \"exclude\": \"\"\n}"

	buffer := new(bytes.Buffer)
	msg_bytes := []byte(msg)
	err := json.Compact(buffer, msg_bytes)
	if err != nil {
		t.Fatal(err)
	}

	buffer_to_bytes := buffer.Bytes()

	var param gpt.ChatCompletionFunctionCallParams

	err = json.Unmarshal(buffer_to_bytes, &param)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "스릴러 소설을 추천해드릴게요.", param.AssistantMessage)

}

func TestUnmarshalEscapeStringRaw(t *testing.T) {
	msg := `"{\n  \"title\": \"\",\n  \"description\": \"\",\n  \"author\": \"\",\n  \"category\": \"소설\",\n  \"assistant_message\": \"스릴러 소설을 추천해드릴게요.\",\n  \"exclude\": \"\"\n}"`
	msg_raw := json.RawMessage(msg)
	fmt.Printf("msg_raw\n%v", string(msg_raw))

	// unquote
	msg_unquote, err := strconv.Unquote(string(msg_raw))
	if err != nil {
		t.Fatal(err)
	}

	// compact
	buffer := new(bytes.Buffer)
	msg_bytes := []byte(msg_unquote)
	err = json.Compact(buffer, msg_bytes)
	if err != nil {
		t.Fatal(err)
	}

	buffer_to_string := buffer.String()
	buffer_to_bytes := buffer.Bytes()
	print(buffer_to_string)

	var param gpt.ChatCompletionFunctionCallParams

	err = json.Unmarshal(buffer_to_bytes, &param)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "스릴러 소설을 추천해드릴게요.", param.AssistantMessage)

}

// HTTP Client For Test
func (f *TestRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	f.Req = append(f.Req, req)
	return f.RoundTripFunc(req)
}

type TestRoundTripper struct {
	Req           []*http.Request
	RoundTripFunc func(req *http.Request) (*http.Response, error)
}

func NewTestRoundTripper(fn func(req *http.Request) (*http.Response, error)) *TestRoundTripper {
	return &TestRoundTripper{
		Req:           []*http.Request{},
		RoundTripFunc: fn,
	}
}

func NewTestClient(roundTripper *TestRoundTripper) *http.Client {
	return &http.Client{
		Transport: roundTripper,
	}
}

// handle strings
func compactJson(jsonStr string) (string, error) {
	compactBuffer := new(bytes.Buffer)
	err := json.Compact(compactBuffer, []byte(jsonStr))
	if err != nil {
		return "", err
	}
	return compactBuffer.String(), nil
}

func TestRawMessage(t *testing.T) {

	type Foo struct {
		ID    string           `json:"id"`
		Name  string           `json:"name"`
		Extra *json.RawMessage `json:"extra"`
	}
	// Initialize a string and a nested JSON
	str := `"some_string"`
	nestedJSON := []byte(`{"key1":"value1","key2":"value2"}`)

	// Marshaling to JSON
	raw1 := json.RawMessage(str)
	foo1 := Foo{
		ID:    "1",
		Name:  "Example 1",
		Extra: &raw1,
	}
	b1, err := json.Marshal(foo1)
	if err != nil {
		t.Fatal(err)
		return
	}
	fmt.Println(string(b1))

	raw2 := json.RawMessage(nestedJSON)
	foo2 := Foo{
		ID:    "2",
		Name:  "Example 2",
		Extra: &raw2,
	}
	b2, err := json.Marshal(foo2)
	if err != nil {
		t.Fatal(err)
		return
	}
	fmt.Println(string(b2))
}
