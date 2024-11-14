package log

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type responseLogField struct {
	RequestID string      `json:"request_id"`
	Body      interface{} `json:"body"`
	TimeMS    float64     `json:"time_ms"`
	Code      int         `json:"code"`
	Error     string      `json:"error,omitempty"`
}

func NewResponseLogFieldFromHTTPResponse(resp *http.Response, ms float64) (*responseLogField, error) {
	buf, _ := io.ReadAll(resp.Body)
	resp.Body = io.NopCloser(bytes.NewReader(buf))

	var j interface{}
	err := json.Unmarshal(buf, &j)
	if err != nil {
		j = string(buf)
	}
	return &responseLogField{
		Body:   j,
		TimeMS: ms,
		Code:   resp.StatusCode,
	}, nil
}

func NewResponseLogFieldFromGrpcResponse(resp interface{}, ms float64, grpcError error) (*responseLogField, error) {
	field := &responseLogField{
		TimeMS: ms,
	}

	if grpcError != nil {
		field.Error = grpcError.Error()
		field.Code = int(status.Code(grpcError))
	}

	if resp != nil {
		var j interface{}

		p, ok := resp.(protoreflect.ProtoMessage)
		if !ok {
			return nil, errors.New("response type conversion to protoreflect.ProtoMessage failed")
		}
		buf, err := protojson.Marshal(p)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(buf, &j)
		if err != nil {
			j = string(buf)
		}

		field.Body = j
	}

	return field, nil
}

func (f *responseLogField) ZapField() zapcore.Field {
	return zap.Any("resp", f)
}
