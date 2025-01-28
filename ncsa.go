package simpleton

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

type NCSAHandler struct {
	writer *os.File
	mutex  sync.Mutex
}

// NewNCSAHandler creates a new handler for NCSA-formatted log output.
func NewNCSAHandler(outputFile *os.File) *NCSAHandler {
	return &NCSAHandler{
		writer: outputFile,
	}
}

func (h *NCSAHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// Enable all log levels. You can customize this if needed.
	return true
}

func (h *NCSAHandler) Handle(ctx context.Context, record slog.Record) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Extract attributes from the log record
	attrs := make(map[string]any)
	record.Attrs(func(attr slog.Attr) bool {
		attrs[attr.Key] = attr.Value.Any()
		return true
	})

	// Retrieve fields required for the NCSA format
	host := attrs["host"]
	if host == nil {
		host = "-"
	}

	user := attrs["user"]
	if user == nil {
		user = "-"
	}

	timestamp := time.Now().Format("02/Jan/2006:15:04:05 -0700")

	method := attrs["method"]
	if method == nil {
		method = "-"
	}

	path := attrs["path"]
	if path == nil {
		path = "-"
	}

	protocol := attrs["protocol"]
	if protocol == nil {
		protocol = "-"
	}

	status := attrs["status"]
	if status == nil {
		status = "-"
	}

	size := attrs["size"]
	if size == nil {
		size = "-"
	}

	// Format the log entry in NCSA Common Access Log format
	logLine := fmt.Sprintf("%s - %s [%s] \"%s %s %s\" %v %v\n",
		host, user, timestamp, method, path, protocol, status, size)

	// Write to the log output
	_, err := h.writer.WriteString(logLine)
	return err
}

func (h *NCSAHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *NCSAHandler) WithGroup(name string) slog.Handler {
	return h
}
