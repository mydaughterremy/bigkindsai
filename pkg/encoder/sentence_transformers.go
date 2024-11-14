package encoder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

var once sync.Once
var stClient *http.Client

type SentenceTransformers struct {
	Client *http.Client
}

type STEncodeResult struct {
	Embeddings [][]float32 `json:"embeddings"`
}

func (s *SentenceTransformers) Encode(query string) ([]float32, error) {
	req, err := createEncodeRequest(query)
	if err != nil {
		return nil, err
	}
	// parse response
	return s.getVectorFromEncodeResponse(req)
}

func createEncodeRequest(query string) (*http.Request, error) {
	req := map[string]interface{}{
		"sentences": []string{query},
	}
	queryBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshalling encode request: %s", err)
	}
	requestURL := "http://sentence-transformers.askupspace-prod.gangnam2.serving.instage.ai/v1/models/sentence-transformer:predict" // TODO: replace this
	encodeRequest, err := http.NewRequest("POST", requestURL, bytes.NewReader(queryBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %s", err)
	}
	encodeRequest.Header.Set("Content-Type", "application/json")
	return encodeRequest, nil
}

func (s *SentenceTransformers) getVectorFromEncodeResponse(req *http.Request) ([]float32, error) {
	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %s", err)
	}
	defer resp.Body.Close()

	// check response status code
	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading response body: %s", err)
		}
		body_string := string(body)
		return nil, fmt.Errorf("response status code is not 200: (%d), %v", resp.StatusCode, body_string)
	}
	var r STEncodeResult

	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("Error parsing the response body: %s", err)
	}
	return r.Embeddings[0], nil
}

func NewSentenceTransformers() *SentenceTransformers {
	once.Do(func() {
		stClient = &http.Client{}
	})
	return &SentenceTransformers{
		Client: stClient,
	}
}
