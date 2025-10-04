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

type LoggerInterface interface {
	ErrorWithID(ctx context.Context, args ...interface{})
	DebugWithID(ctx context.Context, args ...interface{})
	InfoWithID(ctx context.Context, args ...interface{})
	WarnWithID(ctx context.Context, args ...interface{})
}

type Logger struct {
	*zap.SugaredLogger
}

func Initialize(appEnv string) *Logger {
	once.Do(func() {
		var baseLogger *zap.Logger
		var err error

		switch appEnv {
		case "test":
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

func Sync() {
	if loggerInstance != nil {
		_ = loggerInstance.Sync()
	}
}

func GetLogger() *Logger {
	if loggerInstance == nil {
		panic("logger is not initialized. Call Initialize() first.")
	}
	return loggerInstance
}

func (l *Logger) ErrorWithID(ctx context.Context, args ...interface{}) {
	loggerWithSkip := l.SugaredLogger.Desugar().WithOptions(zap.AddCallerSkip(1)).Sugar()
	loc, _ := time.LoadLocation("Asia/Bangkok")
	timestamp := time.Now().In(loc).Format(time.RFC3339)
	loggerWithSkip.Errorf("%s | TimeStamp: %s", fmt.Sprint(args...), timestamp)
}

func (l *Logger) DebugWithID(ctx context.Context, args ...interface{}) {
	loggerWithSkip := l.SugaredLogger.Desugar().WithOptions(zap.AddCallerSkip(1)).Sugar()
	loc, _ := time.LoadLocation("Asia/Bangkok")
	timestamp := time.Now().In(loc).Format(time.RFC3339)
	loggerWithSkip.Debugf("%s | TimeStamp: %s", fmt.Sprint(args...), timestamp)
}

func (l *Logger) WarnWithID(ctx context.Context, args ...interface{}) {
	// loggerWithSkip := l.SugaredLogger.Desugar().WithOptions(zap.AddCallerSkip(1)).Sugar()
	// loc, _ := time.LoadLocation("Asia/Bangkok")
	// timestamp := time.Now().In(loc).Format(time.RFC3339)
	// loggerWithSkip.Warnf("%s | TimeStamp: %s", fmt.Sprint(args...), timestamp)
}

func (l *Logger) InfoWithID(ctx context.Context, args ...interface{}) {
	loggerWithSkip := l.SugaredLogger.Desugar().WithOptions(zap.AddCallerSkip(1)).Sugar()
	loc, _ := time.LoadLocation("Asia/Bangkok")
	timestamp := time.Now().In(loc).Format(time.RFC3339)
	loggerWithSkip.Infof("%s | TimeStamp: %s", fmt.Sprint(args...), timestamp)
}
