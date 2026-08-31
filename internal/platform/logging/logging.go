package logging

import (
	"io"
	"log/slog"
	"strings"
)

func New(output io.Writer, level string) *slog.Logger {
	options := &slog.HandlerOptions{
		Level:       parseLevel(level),
		ReplaceAttr: redactSensitiveAttributes,
	}
	return slog.New(slog.NewJSONHandler(output, options))
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func redactSensitiveAttributes(_ []string, attribute slog.Attr) slog.Attr {
	key := strings.ToLower(attribute.Key)
	for _, sensitive := range []string{"authorization", "cookie", "password", "secret", "token", "credential"} {
		if strings.Contains(key, sensitive) {
			attribute.Value = slog.StringValue("[REDACTED]")
			break
		}
	}
	return attribute
}
