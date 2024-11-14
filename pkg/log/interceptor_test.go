package log

import (
	"context"
	"testing"

	"bigkinds.or.kr/pkg/log/internal/proto"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
)

func TestContextLoggerUnaryServerInterceptor(t *testing.T) {
	logger, _ := zap.NewProduction()

	interceptor := ContextLoggerUnaryServerInterceptor(logger)
	_, _ = interceptor(context.Background(), nil, nil, func(ctx context.Context, req interface{}) (interface{}, error) {
		ctxLogger, err := GetLogger(ctx)
		assert.Nil(t, err)
		assert.Equal(t, logger, ctxLogger)
		return nil, nil
	})
}

func TestRequestLogUnaryServerInterceptor(t *testing.T) {
	observer, logs := observer.New(zap.InfoLevel)
	logger := zap.New(observer)

	interceptor := RequestLogUnaryServerInterceptor()
	ctx := context.Background()
	ctx = WithLogger(ctx, logger)

	req := &proto.TestRequest{
		Msg: "test",
	}
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test",
	}
	_, _ = interceptor(ctx, req, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	})

	fields, err := NewRequestLogFieldFromGrpcRequest(req)
	assert.Nil(t, err)

	entries := logs.All()
	assert.Len(t, entries, 1)

	ctxFields := entries[0].Context
	assert.Len(t, ctxFields, 1)
	expectedZapField := fields.ZapField()
	assert.Equal(t, expectedZapField, ctxFields[0])
}

func TestResponseLogUnaryServerInterceptor(t *testing.T) {
	observer, logs := observer.New(zap.InfoLevel)
	logger := zap.New(observer)

	interceptor := ResponseLogUnaryServerInterceptor()
	ctx := context.Background()
	ctx = WithLogger(ctx, logger)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test",
	}
	resp, err := interceptor(ctx, nil, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		resp := &proto.TestResponse{
			Msg: "test",
		}
		return resp, nil
	})
	assert.Nil(t, err)

	ms := 1.0
	fields, err := NewResponseLogFieldFromGrpcResponse(resp, ms, nil)
	expectedZapField := fields.ZapField()
	assert.Nil(t, err)

	entries := logs.All()
	assert.Len(t, entries, 1)

	ctxFields := entries[0].Context
	assert.Len(t, ctxFields, 1)

	// hack to ignore response time checking
	_interface, _ := ctxFields[0].Interface.(*responseLogField)
	_interface.TimeMS = ms
	ctxFields[0].Interface = _interface

	assert.Equal(t, expectedZapField, ctxFields[0])
}
