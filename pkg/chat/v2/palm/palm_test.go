package palm_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"bigkinds.or.kr/pkg/chat/v2"
	"bigkinds.or.kr/pkg/chat/v2/palm"
	"github.com/stretchr/testify/assert"
)

func TestPalmUnmarshal(t *testing.T) {
	msg := `{
	"predictions": [
		{
			"candidates": [
				{
					"author": "1",
					"content": "I am doing well, thank you for asking!"
				}
			],
			"citationMetadata": [
				{
				"citations": []
				}
			],
			"safetyAttributes": [
				{
				"blocked": false,
				"categories": [],
				"scores": []
				}
			]
		}
	],
	"metadata": {
		"tokenMetadata": {
			"inputTokenCount": {
				"totalTokens": 8,
				"totalBillableCharacters": 15
			},
			"outputTokenCount": {
				"totalTokens": 10,
				"totalBillableCharacters": 31
			}
		}
	}
}`
	var response palm.VertexAIPalmResponse
	err := json.Unmarshal([]byte(msg), &response)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(response.Predections))
	assert.Equal(t, 1, len(response.Predections[0].Candidates))
}

func TestPalmHttpUnmarshal(t *testing.T) {
	const (
		projectID = "test-project-id"
		token     = "test-token"
	)

	cases := []struct {
		TestName string
		Msg      string
	}{
		{
			TestName: "TestPalmHttpUnmarshal",
			Msg: `{
				"predictions": [
					{
						"candidates": [
							{
								"author": "ssong",
								"content": "I am doing well, thank you for asking!"
							}
						],
						"citationMetadata": [
							{
							"citations": []
							}
						],
						"safetyAttributes": [
							{
							"blocked": false,
							"categories": [],
							"scores": []
							}
						]
					}
				],
				"metadata": {
					"tokenMetadata": {
						"inputTokenCount": {
							"totalTokens": 8,
							"totalBillableCharacters": 15
						},
						"outputTokenCount": {
							"totalTokens": 10,
							"totalBillableCharacters": 31
						}
					}
				}
			}`,
		},
	}

	for _, tt := range cases {

		t.Run(tt.TestName, func(t *testing.T) {
			roundTripper := NewTestRoundTripper(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(tt.Msg)),
				}, nil
			})
			testHttpClient := NewTestClient(roundTripper)

			// palmModel, err := palmModel.NewPalmModel(testHttpClient, projectID, token, defaultOption)
			palmModel, err := palm.NewPalmModel(
				testHttpClient,
				palm.WithProjectID(projectID),
				palm.WithToken(token),
			)
			if err != nil {
				t.Fatal(err)
			}

			responseMsg, err := palmModel.Chat(context.Background(), []*chat.ChatPayload{
				{
					Role:    "Jerry",
					Content: "Hello, how are you?",
				},
			}, palm.WithModel("chat-bison@001"))
			if err != nil {
				t.Fatal(err)
			}

			assert.NotNil(t, responseMsg)
			assert.NotNil(t, responseMsg.Payload)
			assert.Equal(t, "I am doing well, thank you for asking!", responseMsg.Payload.Content)
			assert.Equal(t, "ssong", responseMsg.Payload.Role)

		})
	}
}

func TestPalmHttpRequest(t *testing.T) {
	const (
		projectID = "test-project-id"
		token     = "test-token"
		//staticOutput is for avoiding errors
		staticOutput = `{
			"predictions": [
				{
					"candidates": [
						{
							"author": "ssong",
							"content": "I am doing well, thank you for asking!"
						}
					],
					"citationMetadata": [
						{
						"citations": []
						}
					],
					"safetyAttributes": [
						{
						"blocked": false,
						"categories": [],
						"scores": []
						}
					]
				}
			],
			"metadata": {
				"tokenMetadata": {
					"inputTokenCount": {
						"totalTokens": 8,
						"totalBillableCharacters": 15
					},
					"outputTokenCount": {
						"totalTokens": 10,
						"totalBillableCharacters": 31
					}
				}
			}
		}`
	)

	cases := []struct {
		TestName string
		// DefaultOption *palm.PalmOptions
		// Option        *palm.PalmOptions
		PalmOption           []func(*palm.PalmOptions)
		PalmPredictionOption []func(*palm.PalmPredictionOptions)
		ChatMessages         []*chat.ChatPayload
		ExpectedReq          string
		ExpectedErr          error
	}{
		{
			TestName: "TestPalmHttpRequestWithOption",
			PalmOption: []func(*palm.PalmOptions){
				palm.WithProjectID(projectID),
				palm.WithToken(token),
			},
			PalmPredictionOption: []func(*palm.PalmPredictionOptions){
				palm.WithModel("chat-bison@001"),
				palm.WithContext("냥냥체로 대답해줘"),
				palm.WithTemperature(0.2),
				palm.WithMaxOutputTokens(256),
				palm.WithTopP(0.8),
				palm.WithTopK(40),
			},
			ChatMessages: []*chat.ChatPayload{
				{
					Role:    "Jerry",
					Content: "Hello, how are you?",
				},
			},
			ExpectedReq: `{
				"instances": [
					{
						"context": "냥냥체로 대답해줘",
						"messages": [
							{
								"author": "Jerry",
								"content": "Hello, how are you?"
							}
						]
					}
				],
				"parameters": {
					"temperature": 0.2,
					"maxOutputTokens": 256,
					"topP": 0.8,
					"topK": 40
				}
			}`,
			ExpectedErr: nil,
		},
		{
			TestName: "TestPalmHttpRequestNoParameters",
			PalmOption: []func(*palm.PalmOptions){
				palm.WithProjectID(projectID),
				palm.WithToken(token),
			},
			PalmPredictionOption: []func(*palm.PalmPredictionOptions){
				palm.WithModel("chat-bison@001"),
			},
			ChatMessages: []*chat.ChatPayload{
				{
					Role:    "Ssong",
					Content: "냥냥냥",
				},
				{
					Role:    "Jerry",
					Content: "멍멍멍",
				},
			},
			ExpectedReq: `{
				"instances": [
					{
						"messages": [
							{
								"author": "Ssong",
								"content": "냥냥냥"
							},
							{
								"author": "Jerry",
								"content": "멍멍멍"
							}
						]
					}
				],
				"parameters": {}
			}`,
		},
		{
			TestName: "TestPalmHttpRequestNoModelOption",
			ChatMessages: []*chat.ChatPayload{
				{
					Role:    "Junhyun",
					Content: "dududududu",
				},
			},
			PalmOption: []func(*palm.PalmOptions){
				palm.WithProjectID(projectID),
				palm.WithToken(token),
			},
			PalmPredictionOption: nil,
			ExpectedReq:          "",
			ExpectedErr:          errors.New("model is required"),
		},
		{
			TestName: "TestPalmHttpRequestNoChatMessages",
			PalmOption: []func(*palm.PalmOptions){
				palm.WithProjectID(projectID),
				palm.WithToken(token),
			},
			PalmPredictionOption: []func(*palm.PalmPredictionOptions){
				palm.WithModel("chat-bison@001"),
			},
			ChatMessages: []*chat.ChatPayload{},
			ExpectedErr:  errors.New("chat messages are required"),
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
			// palm, err := palm.NewPalmModel(testHttpClient, projectID, token, *tt.DefaultOption)
			palmModel, err := palm.NewPalmModel(
				testHttpClient,
				tt.PalmOption...,
			)
			if err != nil {
				if err != nil && tt.ExpectedErr == nil || err == nil && tt.ExpectedErr != nil {
					t.Fatalf("Expected error %v, got %v", tt.ExpectedErr, err)
				}
				return
			}

			// perform Chat
			_, err = palmModel.Chat(context.Background(), tt.ChatMessages, tt.PalmPredictionOption...)
			if err != nil {
				if err != nil && tt.ExpectedErr == nil || err == nil && tt.ExpectedErr != nil {
					t.Fatalf("Expected error %v, got %v", tt.ExpectedErr, err)
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
