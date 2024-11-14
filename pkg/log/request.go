package log

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type requestLogField struct {
	RequestID string      `json:"request_id"`
	Body      interface{} `json:"body"`
	Method    string      `json:"method"`
	Path      string      `json:"path"`
}

type requestIDKey struct{}

func NewRequestLogFieldFromHTTPRequest(r *http.Request) (*requestLogField, error) {
	buf, _ := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(buf))

	method := r.Method
	var path string
	if r.URL != nil {
		path = r.URL.Path
	}
	switch r.Header.Get("Content-Type") {
	default:
		var j interface{}
		err := json.Unmarshal(buf, &j)
		if err != nil {
			j = string(buf)
		}
		return &requestLogField{
			Body:   j,
			Method: method,
			Path:   path,
		}, nil
	}
}
func NewRequestLogFieldFromGrpcRequest(req interface{}) (*requestLogField, error) {
	p, ok := req.(protoreflect.ProtoMessage)
	if !ok {
		return nil, errors.New("request type conversion to protoreflect.ProtoMessage failed")
	}
	buf, err := protojson.Marshal(p)
	if err != nil {
		return nil, err
	}

	var j interface{}
	err = json.Unmarshal(buf, &j)
	if err != nil {
		j = string(buf)
	}
	return &requestLogField{
		Body: j,
	}, nil
}

func (f *requestLogField) ZapField() zapcore.Field {
	return zap.Any("req", f)
}

func GetRequestIDFromHTTPRequest(r *http.Request) string {
	// if request id is started with "recsys-", it is a requested to recsys traefik
	reqID := r.Header.Get("X-Upstage-Request-Id")
	if len(reqID) == 0 {
		reqID = "llmapps-" + uuid.New().String()
	}

	return reqID
}

func GetRequestIDFromContext(ctx context.Context) string {
	reqID, ok := ctx.Value(requestIDKey{}).(string)
	if !ok {
		return ""
	}
	return reqID
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}
