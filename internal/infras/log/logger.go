package log

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	loggerInstance *Logger
	once           sync.Once
)

// LoggerInterface defines the methods for custom logger
type LoggerInterface interface {
	ErrorWithID(ctx context.Context, args ...interface{})
	DebugWithID(ctx context.Context, args ...interface{})
	InfoWithID(ctx context.Context, args ...interface{})
	WarnWithID(ctx context.Context, args ...interface{})
}

// Logger wraps zap.SugaredLogger and implements LoggerInterface
type Logger struct {
	*zap.SugaredLogger
}

// Initialize sets up the logger based on the application environment
func Initialize(appEnv string) *Logger {
	once.Do(func() {
		var baseLogger *zap.Logger
		var err error

		switch appEnv {
		case "test":
			// Use no-op logger that disables all logs
			baseLogger = zap.NewNop()
		case "dev", "local":
			config := zap.NewDevelopmentConfig()
			config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
			config.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
			config.EncoderConfig.TimeKey = ""
			config.EncoderConfig.CallerKey = "caller"
			config.EncoderConfig.MessageKey = "msg"
			config.EncoderConfig.LevelKey = "level"
			config.EncoderConfig.ConsoleSeparator = " | "
			baseLogger, err = config.Build(zap.AddCaller())
			if err != nil {
				panic("failed to initialize zap logger: " + err.Error())
			}
		default:
			config := zap.NewProductionConfig()
			config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
			baseLogger, err = config.Build(zap.AddCaller())
			if err != nil {
				panic("failed to initialize zap logger: " + err.Error())
			}
		}

		loggerInstance = &Logger{baseLogger.Sugar()}
	})

	return loggerInstance
}

// Sync flushes any buffered log entries
func Sync() {
	if loggerInstance != nil {
		_ = loggerInstance.Sync()
	}
}

// GetLogger returns the singleton logger instance
func GetLogger() *Logger {
	if loggerInstance == nil {
		panic("logger is not initialized. Call Initialize() first.")
	}
	return loggerInstance
}

// ErrorWithID logs an error with custom context information, adjusting caller skip
func (l *Logger) ErrorWithID(ctx context.Context, args ...interface{}) {
	// Create a new logger instance with caller skip set to 1 to point to the handler
	loggerWithSkip := l.SugaredLogger.Desugar().WithOptions(zap.AddCallerSkip(1)).Sugar()
	loc, _ := time.LoadLocation("Asia/Bangkok")
	timestamp := time.Now().In(loc).Format(time.RFC3339)
	loggerWithSkip.Errorf("%s | TimeStamp: %s", fmt.Sprint(args...), timestamp)
}

// DebugWithID logs a debug message with custom context information, adjusting caller skip
func (l *Logger) DebugWithID(ctx context.Context, args ...interface{}) {
	// Create a new logger instance with caller skip set to 1 to point to the handler
	loggerWithSkip := l.SugaredLogger.Desugar().WithOptions(zap.AddCallerSkip(1)).Sugar()
	loc, _ := time.LoadLocation("Asia/Bangkok")
	timestamp := time.Now().In(loc).Format(time.RFC3339)
	loggerWithSkip.Debugf("%s | TimeStamp: %s", fmt.Sprint(args...), timestamp)
}

func (l *Logger) WarnWithID(ctx context.Context, args ...interface{}) {
	loggerWithSkip := l.SugaredLogger.Desugar().WithOptions(zap.AddCallerSkip(1)).Sugar()
	loc, _ := time.LoadLocation("Asia/Bangkok")
	timestamp := time.Now().In(loc).Format(time.RFC3339)
	loggerWithSkip.Warnf("%s | TimeStamp: %s", fmt.Sprint(args...), timestamp)
}

// InfoWithID logs an info message with custom context information, adjusting caller skip
func (l *Logger) InfoWithID(ctx context.Context, args ...interface{}) {
	loggerWithSkip := l.SugaredLogger.Desugar().WithOptions(zap.AddCallerSkip(1)).Sugar()
	loc, _ := time.LoadLocation("Asia/Bangkok")
	timestamp := time.Now().In(loc).Format(time.RFC3339)
	loggerWithSkip.Infof("%s | TimeStamp: %s", fmt.Sprint(args...), timestamp)
}
