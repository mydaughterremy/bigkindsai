package log

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	pb "bigkinds.or.kr/pkg/log/internal/proto"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestResponseZapField(t *testing.T) {
	b := []byte("{\"test\":\"test\"}")

	var j interface{}
	_ = json.Unmarshal(b, &j)
	f := &responseLogField{
		Body: j,
	}

	actual := f.ZapField()
	expected := zap.Any("resp", f)
	assert.Equal(t, expected, actual)
}

func TestNewResponseLogFieldFromHTTPResponse(t *testing.T) {
	b := []byte("{\"test\":\"test\"}")
	rc := readCloser(b)
	resp := &http.Response{
		Body:       rc,
		StatusCode: 200,
	}
	ms := 1.0
	f, err := NewResponseLogFieldFromHTTPResponse(resp, ms)
	assert.Nil(t, err)

	var j interface{}
	_ = json.Unmarshal(b, &j)
	expected := &responseLogField{
		Body:   j,
		TimeMS: ms,
		Code:   200,
	}
	assert.Equal(t, expected, f)
}

func TestResponseBodyNotAffected(t *testing.T) {
	b := []byte("{\"test\":\"test\"}")
	rc := readCloser(b)
	resp := &http.Response{
		Body:       rc,
		StatusCode: 200,
	}
	ms := 1.0
	f, err := NewResponseLogFieldFromHTTPResponse(resp, ms)
	assert.Nil(t, err)

	var j interface{}
	_ = json.Unmarshal(b, &j)
	expected := &responseLogField{
		Body:   j,
		TimeMS: ms,
		Code:   200,
	}
	assert.Equal(t, expected, f)

	buf, err := io.ReadAll(rc)
	emptyByte := []byte{}
	assert.Nil(t, err)
	assert.Equal(t, emptyByte, buf)
	// rc should be already read entirely

	buf, err = io.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, b, buf)
}

func TestNewResponseLogFieldFromGrpcResponse(t *testing.T) {
	req := &pb.TestResponse{
		Msg: "test",
	}
	ms := 1.0
	b := []byte("{\"msg\":\"test\"}")
	f, err := NewResponseLogFieldFromGrpcResponse(req, ms, nil)
	assert.Nil(t, err)

	var j interface{}
	_ = json.Unmarshal(b, &j)
	expected := &responseLogField{
		Body:   j,
		TimeMS: ms,
		Code:   int(codes.OK),
	}
	assert.Equal(t, expected, f)

	grpcError := status.Error(codes.Unauthenticated, "unauthenticated")
	f, err = NewResponseLogFieldFromGrpcResponse(req, ms, grpcError)
	assert.Nil(t, err)

	_ = json.Unmarshal(b, &j)
	expected = &responseLogField{
		Body:   j,
		TimeMS: ms,
		Code:   int(codes.Unauthenticated),
		Error:  grpcError.Error(),
	}
	assert.Equal(t, expected, f)

}
