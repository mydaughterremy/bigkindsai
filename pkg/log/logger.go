package log

import (
	"context"
	"errors"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type loggerKey struct{}

func loggingLevel(level string) zapcore.Level {
	switch level {
	case "DEBUG":
		return zapcore.DebugLevel
	case "INFO":
		return zapcore.InfoLevel
	case "WARN":
		return zapcore.WarnLevel
	case "ERROR":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func newConfig() zap.Config {
	config := zap.NewProductionConfig()
	level := loggingLevel(os.Getenv("LOG_LEVEL"))
	config.Level.SetLevel(level)
	config.Sampling = nil
	config.DisableCaller = true

	return config
}

func NewLogger(service string, options ...zap.Option) (*zap.Logger, error) {
	logger, err := newConfig().Build(options...)
	if err != nil {
		return nil, err
	}
	logger = attachServiceField(logger, service)
	return logger, nil
}

func attachServiceField(logger *zap.Logger, service string) *zap.Logger {
	return logger.With(
		zap.String("service", service),
	)
}

func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func GetLogger(ctx context.Context) (*zap.Logger, error) {
	logger, ok := ctx.Value(loggerKey{}).(*zap.Logger)
	if !ok {
		return nil, errors.New("no logger in context")
	}
	return logger, nil
}
