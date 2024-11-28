package request

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"bigkinds.or.kr/backend/model"
	"github.com/sirupsen/logrus"
)

type SSEStream struct {
	reader *bufio.Reader
	body   io.ReadCloser
	merged *model.Completion
}

func CreateChatStream(ctx context.Context, client *http.Client, host string, body []byte) (*SSEStream, error) {
	request, err := http.NewRequestWithContext(ctx, "POST", host, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "text/event-stream")

	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		body, err := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("response status code is not in 200-299, status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return &SSEStream{
		reader: bufio.NewReader(resp.Body),
		body:   resp.Body,
	}, nil
}

type SSEToken string

const (
	SSEDataStartToken  SSEToken = "data: "
	SSEDoneToken       SSEToken = "[DONE]"
	SSEErrorStartToken SSEToken = "{\"error\""
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func (c *SSEStream) Close() error {
	return c.body.Close()
}

func (c *SSEStream) Recv() (*model.Completion, error) {
	for {
		resp, err := c.reader.ReadBytes('\n')
		logrus.Debugf("resp: %s", string(resp))
		if errors.Is(err, io.EOF) {
			return nil, errors.New("EOF comes before [DONE]")
		}
		if err != nil {
			return nil, err
		}

		resp = bytes.TrimSpace(resp)

		if !bytes.HasPrefix(resp, []byte(SSEDataStartToken)) {
			continue
		}

		data := bytes.TrimPrefix(resp, []byte(SSEDataStartToken))

		if strings.HasPrefix(string(data), string(SSEErrorStartToken)) {
			var errresp ErrorResponse
			err = json.Unmarshal(data, &errresp)
			if err != nil {
				return nil, errors.New("failed to unmarshal error response")
			}

			return nil, errors.New(errresp.Error)
		}

		if bytes.Equal(data, []byte(SSEDoneToken)) {
			return nil, io.EOF
		}

		var chunk model.Completion
		err = json.Unmarshal(data, &chunk)
		if err != nil {
			return nil, err
		}

		if c.merged != nil {
			err := c.merged.Merge(&chunk)

			if err != nil {
				return nil, err
			}
		} else {
			c.merged = &chunk
		}

		return &chunk, err
	}
}

func (c *SSEStream) ReadUntilNow() *model.Completion {
	return c.merged
}
