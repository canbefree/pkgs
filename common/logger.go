package common

import (
	"context"
	"io"
	"log"
)

type LoggerIFace interface {
	Fatalf(ctx context.Context, format string, v ...any)
}

type Logger struct {
	*log.Logger
}

func NewDefaultLogger() *Logger {
	return &Logger{
		Logger: log.Default(),
	}
}

func NewLogger(out io.Writer, prefix string, flag int) *Logger {
	return &Logger{
		Logger: log.New(out, prefix, flag),
	}
}

func (l *Logger) Fatalf(ctx context.Context, format string, v ...any) {
	l.Logger.Fatalf(format, v...)
}
