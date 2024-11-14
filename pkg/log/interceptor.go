package log

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func ContextLoggerUnaryServerInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		ctx = WithLogger(ctx, logger)
		return handler(ctx, req)
	}
}

func RequestLogUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		logger, err := GetLogger(ctx)
		if err != nil {
			logger, _ = zap.NewProduction()
			ctx = WithLogger(ctx, logger)
		}
		fields, err := NewRequestLogFieldFromGrpcRequest(req)
		if err != nil {
			logger.Error("Error while parsing request log field", zap.Any("error", err))
		}
		if info != nil {
			if info.FullMethod != "/grpc.health.v1.Health/Check" {
				logger.Info("grpc request", fields.ZapField())
			}
		} else {
			logger.Info("grpc request", fields.ZapField())
		}
		return handler(ctx, req)
	}
}

func ResponseLogUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		t := time.Now()
		logger, err := GetLogger(ctx)
		if err != nil {
			logger, _ = zap.NewProduction()
			ctx = WithLogger(ctx, logger)
		}
		ms := calculateMS(time.Since(t).Nanoseconds())

		grpcResp, grpcError := handler(ctx, req)
		fields, err := NewResponseLogFieldFromGrpcResponse(grpcResp, ms, grpcError)
		if err != nil {
			logger.Error("Error while parsing response log field", zap.Any("error", err))
		}
		if info != nil {
			if info.FullMethod != "/grpc.health.v1.Health/Check" {
				logger.Info("grpc response", fields.ZapField())
			}
		} else {
			logger.Info("grpc response", fields.ZapField())
		}
		return grpcResp, grpcError
	}
}
