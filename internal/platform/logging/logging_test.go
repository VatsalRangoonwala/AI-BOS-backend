package logging

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNewRedactsSensitiveAttributes(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	logger := New(&output, "info")
	logger.Info("configuration", slog.String("database_password", "top-secret"), slog.String("component", "api"))

	logged := output.String()
	if strings.Contains(logged, "top-secret") {
		t.Fatalf("log output leaked secret: %s", logged)
	}
	if !strings.Contains(logged, "[REDACTED]") {
		t.Fatalf("log output did not contain redaction marker: %s", logged)
	}
}
