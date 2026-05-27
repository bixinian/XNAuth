package logx

import (
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"xnauth/internal/config"
)

func New(cfg config.LogConfig) (*zap.Logger, func(), error) {
	level := zapcore.InfoLevel
	if strings.TrimSpace(cfg.Level) != "" {
		if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
			return nil, nil, err
		}
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	cores := []zapcore.Core{
		zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.AddSync(os.Stdout),
			level,
		),
	}

	var file *os.File
	if strings.TrimSpace(cfg.Path) != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.Path), 0o755); err != nil {
			return nil, nil, err
		}
		opened, err := os.OpenFile(cfg.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, nil, err
		}
		file = opened
		cores = append(cores, zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.AddSync(file),
			level,
		))
	}

	logger := zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	cleanup := func() {
		_ = logger.Sync()
		if file != nil {
			_ = file.Close()
		}
	}
	return logger, cleanup, nil
}
