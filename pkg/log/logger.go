// Each package creates its own logger via log.New("packagename").
// Call Init() once at startup before any other package logs.
package log

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

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

// New returns a logger scoped to a package name.
func New(name string) *slog.Logger {
	return slog.Default().With("pkg", name)
}
