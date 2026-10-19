package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger = *zap.SugaredLogger

func New(level string) (Logger, error) {
	zapLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		return nil, fmt.Errorf("log level %q: %w", level, err)
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			CallerKey:      "caller",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}),
		zapcore.AddSync(os.Stdout),
		zap.NewAtomicLevelAt(zapLevel),
	)

	// SugaredLogger adds a frame; skip it so caller is the application line.
	log := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	return log.Sugar(), nil
}
