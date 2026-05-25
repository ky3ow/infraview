// Each package creates its own logger via log.New("packagename").
// Call Init() once at startup before any other package logs.
// TODO figure out how to properly set configuration and have loggers per package
// with context for tracing
package log

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Debug(msg string, args ...any)
	Log(ctx context.Context, level slog.Level, msg string, args ...any)
}

type packageLogger struct {
	pkg string
}

func Init(level, format string) error {
	lvl, err := parseLevel(level)
	if err != nil {
		return err
	}
	jsonFormat, err := parseFormat(format)
	if err != nil {
		return err
	}

	opts := &slog.HandlerOptions{Level: lvl}

	var handler slog.Handler
	if jsonFormat {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
	return nil
}

func New(name string) Logger {
	return &packageLogger{pkg: name}
}

func (p *packageLogger) Info(msg string, args ...any) {
	slog.Default().With("pkg", p.pkg).Info(msg, args...)
}

func (p *packageLogger) Warn(msg string, args ...any) {
	slog.Default().With("pkg", p.pkg).Warn(msg, args...)
}

func (p *packageLogger) Error(msg string, args ...any) {
	slog.Default().With("pkg", p.pkg).Error(msg, args...)
}

func (p *packageLogger) Debug(msg string, args ...any) {
	slog.Default().With("pkg", p.pkg).Debug(msg, args...)
}

func (p *packageLogger) Log(ctx context.Context, level slog.Level, msg string, args ...any) {
	slog.Default().With("pkg", p.pkg).Log(ctx, level, msg, args...)
}

func parseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelWarn, fmt.Errorf("unknown log level: %q", s)
	}
}

func parseFormat(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "json":
		return true, nil
	case "text":
		return false, nil
	default:
		return false, fmt.Errorf("unknown log format: %q", s)
	}
}

