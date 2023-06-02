package logger

import (
	"fmt"

	"go.uber.org/zap"
)

// Тут мне не нравится инициализация синглтона, я бы использовал once.Do, если бы писал сам, но это я стырил с лекции и не стал переделывать
var log *zap.Logger = zap.NewNop()

func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return fmt.Errorf("cannot parse log level: %w", err)
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl
	zl, err := cfg.Build()
	if err != nil {
		return fmt.Errorf("cannot build log: %w", err)
	}

	log = zl
	return nil
}

func Info(msg string, opt ...any) {
	log.Info(fmt.Sprintf(msg, opt...))
}

func Error(msg string, opt ...any) {
	log.Error(fmt.Sprintf(msg, opt...))
}
