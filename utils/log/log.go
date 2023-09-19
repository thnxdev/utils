package log

import (
	"context"

	"github.com/sirupsen/logrus"
)

var loggerKey = struct{}{}

// Logger is the interface that wraps the basic logging methods.
type Logger = logrus.Ext1FieldLogger

// LoggerContext returns a new context with a logger attached.
func LoggerContext(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext returns the logger from the context. If no logger is found this will panic.
func FromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value(loggerKey).(Logger); ok {
		return logger
	}
	return logrus.StandardLogger()
}
