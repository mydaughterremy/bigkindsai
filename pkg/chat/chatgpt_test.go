package chat_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	cm "bigkinds.or.kr/pkg/chat/model"
	"github.com/stretchr/testify/assert"
)

// func TestChatGPT(t *testing.T) {
// 	apiEndpoint := "https://api.openai.com/v1/chat/completions"
// 	apiKey := os.Getenv("OPENAI_API_KEY")
// 	if len(apiKey) == 0 {
// 		t.SkipNow()
// 	}

// 	reqBody := []byte(`{"model": "gpt-3.5-turbo", "messages": [{"role":"user", "content": "What is your name?"}]}`)

// 	// resp, err := http.(apiEndpoint, "application/json", bytes.NewBuffer(reqBody))
// 	req, err := http.NewRequest("POST", apiEndpoint, bytes.NewBuffer(reqBody))
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	req.Header.Set("Authorization", "Bearer "+apiKey)
// 	req.Header.Set("Content-Type", "application/json")

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer resp.Body.Close()

// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	fmt.Printf("Response: %s\n", body)

// }

func TestUnmarshalRawString(t *testing.T) {
	msg := `{"title": "","description": "","author": "","category": "소설","assistant_message": "스릴러 소설을 추천해드릴게요.","exclude": ""}`
	var param cm.ChatCompletionFunctionCallParams

	msg_bytes := []byte(msg)

	err := json.Unmarshal(msg_bytes, &param)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "스릴러 소설을 추천해드릴게요.", param.AssistantMessage)
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

	var param cm.ChatCompletionFunctionCallParams

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

	var param cm.ChatCompletionFunctionCallParams

	err = json.Unmarshal(buffer_to_bytes, &param)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "스릴러 소설을 추천해드릴게요.", param.AssistantMessage)

}
