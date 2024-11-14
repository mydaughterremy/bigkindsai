package log

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggingLevel(t *testing.T) {
	var level string

	level = "DEBUG"
	assert.Equal(t, loggingLevel(level), zapcore.DebugLevel)
	level = "INFO"
	assert.Equal(t, loggingLevel(level), zapcore.InfoLevel)
	level = "WARN"
	assert.Equal(t, loggingLevel(level), zapcore.WarnLevel)
	level = "ERROR"
	assert.Equal(t, loggingLevel(level), zapcore.ErrorLevel)
	level = "TEST"
	assert.Equal(t, loggingLevel(level), zapcore.InfoLevel)
}

func TestNewConfigWithLogLevelEnv(t *testing.T) {
	c := newConfig()
	assert.Equal(t, c.Level, zap.NewAtomicLevelAt(zapcore.InfoLevel))

	err := os.Setenv("LOG_LEVEL", "DEBUG")
	assert.Nil(t, err)
	c = newConfig()
	assert.Equal(t, c.Level, zap.NewAtomicLevelAt(zapcore.DebugLevel))
}

func TestNewConfig(t *testing.T) {
	config := newConfig()
	assert.Nil(t, config.Sampling)
	assert.Equal(t, true, config.DisableCaller)
}

func TestAttachServiceField(t *testing.T) {
	zapCore, logs := observer.New(zap.InfoLevel)
	logger := zap.New(zapCore)
	logger = attachServiceField(logger, "test")
	logger.Info("msg")
	allLogs := logs.All()
	assert.Equal(t, 1, len(allLogs))
	assert.Contains(t, allLogs[0].Context, zap.String("service", "test"))
}

func TestWithLogger(t *testing.T) {
	ctx := context.Background()
	logger, _ := NewLogger("test")
	ctx = WithLogger(ctx, logger)
	assert.Equal(t, ctx.Value(loggerKey{}), logger)
}

func TestGetLogger(t *testing.T) {
	ctx := context.Background()
	logger, _ := NewLogger("test")
	ctx = WithLogger(ctx, logger)
	ctxLogger, err := GetLogger(ctx)
	assert.Nil(t, err)
	assert.Equal(t, ctxLogger, logger)
}
