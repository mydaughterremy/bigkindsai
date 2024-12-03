package request

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"bigkinds.or.kr/backend/model"
)

func ConvIssueTopicSummaryRequest(ctx context.Context, client *http.Client, host string, body []byte) (*model.IssueTopicSummary, error) {
	host = host + "/v2/summary"
	req, err := http.NewRequestWithContext(ctx, "POST", host, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("response status code is not success, status code: %d, body: %s", resp.StatusCode, string(body))
	}
	defer resp.Body.Close()

	var its model.IssueTopicSummary
	err = json.NewDecoder(resp.Body).Decode(&its)
	if err != nil {
		return nil, err
	}

	return &its, nil

}
