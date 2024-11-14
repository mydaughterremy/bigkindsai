package log

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	pb "bigkinds.or.kr/pkg/log/internal/proto"
)

func readCloser(b []byte) io.ReadCloser {
	reader := bytes.NewReader(b)
	return io.NopCloser(reader)
}

func TestRequestZapField(t *testing.T) {
	b := []byte("{\"test\":\"test\"}")

	var j interface{}
	_ = json.Unmarshal(b, &j)
	f := &requestLogField{
		Body: &j,
	}

	actual := f.ZapField()
	expected := zap.Any("req", f)
	assert.Equal(t, expected, actual)
}

func TestNewRequestLogFieldFromValidJSONHTTPRequest(t *testing.T) {
	b := []byte("{\"test\":\"test\"}")
	rc := readCloser(b)
	req := &http.Request{
		Body: rc,
	}
	f, err := NewRequestLogFieldFromHTTPRequest(req)
	assert.Nil(t, err)

	var j interface{}
	_ = json.Unmarshal(b, &j)
	expected := &requestLogField{
		Body: j,
	}
	assert.Equal(t, expected, f)
}

func TestNewRequestLogFieldFromInvalidJSONHTTPRequest(t *testing.T) {
	b := []byte("{\"test\":\"test}")
	rc := readCloser(b)
	req := &http.Request{
		Body: rc,
	}
	f, err := NewRequestLogFieldFromHTTPRequest(req)
	assert.Nil(t, err)

	expected := &requestLogField{
		Body: string(b),
	}
	assert.Equal(t, expected, f)
}

func TestRequestBodyNotAffected(t *testing.T) {
	b := []byte("{\"test\":\"test\"}")
	rc := readCloser(b)
	req := &http.Request{
		Body: rc,
	}
	f, err := NewRequestLogFieldFromHTTPRequest(req)
	assert.Nil(t, err)

	var j interface{}
	_ = json.Unmarshal(b, &j)
	expected := &requestLogField{
		Body: j,
	}
	assert.Equal(t, expected, f)

	buf, err := io.ReadAll(rc)
	emptyByte := []byte{}
	assert.Nil(t, err)
	assert.Equal(t, emptyByte, buf)
	// rc should be already read entirely

	buf, err = io.ReadAll(req.Body)
	assert.Nil(t, err)
	assert.Equal(t, b, buf)
}

func TestNewRequestLogFieldFromGrpcRequest(t *testing.T) {
	req := &pb.TestRequest{
		Msg: "test",
	}
	b := []byte("{\"msg\":\"test\"}")
	f, err := NewRequestLogFieldFromGrpcRequest(req)
	assert.Nil(t, err)

	var j interface{}
	_ = json.Unmarshal(b, &j)
	expected := &requestLogField{
		Body: j,
	}
	assert.Equal(t, expected, f)
}
