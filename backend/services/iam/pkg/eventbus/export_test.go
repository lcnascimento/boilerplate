package eventbus

import (
	"testing"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/log/logtest"

	"github.com/lcnascimento/go-kit/o11y/log"
)

func RecordLogs(t *testing.T) *logtest.Recorder {
	t.Helper()

	recorder := logtest.NewRecorder()
	original := logger
	logger = log.MustNewLogger(pkg, log.WithLogHandler(otelslog.NewHandler(pkg, otelslog.WithLoggerProvider(recorder))))

	t.Cleanup(func() { logger = original })

	return recorder
}
