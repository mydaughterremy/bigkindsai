package reranker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"bigkinds.or.kr/pkg/log"
	singleton_e5 "bigkinds.or.kr/pkg/reranker/e5"
	"bigkinds.or.kr/proto"
)

type E5Reranker struct {
	Url    string
	Client *http.Client
}

type Passages struct {
	DedupField   string        `json:"dedup_field"`
	TargetFields []string      `json:"target_fields"`
	Items        []*proto.Item `json:"items"`
}

type E5RerankRequest struct {
	Query    string   `json:"query"`
	Passages Passages `json:"passages"`
	K        int32    `json:"k"`
}

type E5RerankResponse struct {
	Topk []struct {
		Item  *proto.Item `json:"item"`
		Score float64     `json:"score"`
	} `json:"topk"`
}

func e5RerankerResponseToItems(resp *http.Response) ([]*proto.Item, error) {
	var r E5RerankResponse

	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("error parsing the response body: %s", err)
	}

	topk := r.Topk
	items := make([]*proto.Item, len(topk))
	for i, k := range topk {
		items[i] = k.Item
		items[i].Score = float32(k.Score)
	}

	return items, nil
}

func (s *E5Reranker) newE5RerankRequest(req *E5RerankRequest) (*http.Request, error) {
	// marshal query
	queryBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshalling rerank request: %s", err)
	}

	// create rerankRequest
	rerankRequest, err := http.NewRequest("POST", s.Url, bytes.NewReader(queryBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %s", err)
	}
	rerankRequest.Header.Set("Content-Type", "application/json")
	return rerankRequest, nil
}

func (s *E5Reranker) Rerank(ctx context.Context, config *proto.Reranker, query string, items []*proto.Item, k int32) ([]*proto.Item, error) {
	// get logger
	logger, err := log.GetLogger(ctx)
	if err != nil {
		logger, _ = log.NewLogger("rerank")
	}

	// create rerankRequest
	req := &E5RerankRequest{
		Query: query,
		Passages: Passages{
			DedupField:   config.DedupField,
			TargetFields: config.PassageFields,
			Items:        items,
		},
		K: k,
	}
	e5RerankerRequest, err := s.newE5RerankRequest(req)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	// send request
	resp, err := s.Client.Do(e5RerankerRequest)
	if err != nil {
		if os.IsTimeout(err) {
			return nil, ErrRerankerTimeout
		} else {
			return nil, fmt.Errorf("error sending request: %s", err)
		}
	}
	defer resp.Body.Close()

	// check response status code
	if resp.StatusCode != 200 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			errMsg := fmt.Sprintf("error reading response body: %s", err)
			logger.Error(errMsg)
			return nil, errors.New(errMsg)
		}
		body_string := string(body)
		errMsg := fmt.Sprintf("error response status code: %d, body: %s", resp.StatusCode, body_string)
		logger.Error(errMsg)
		return nil, errors.New(errMsg)
	}

	return e5RerankerResponseToItems(resp)
}

func NewE5Reranker() (*E5Reranker, error) {
	url := os.Getenv("UPSTAGE_E5_RERANKER_URL")
	client, err := singleton_e5.GetE5RerankerClient()
	if err != nil {
		return nil, err
	}
	return &E5Reranker{
		Url:    url,
		Client: client,
	}, nil
}
