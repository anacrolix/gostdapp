package app

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"syscall"
	"testing"
)

// Collects records so tests can assert what OnMainReturned logged, and through which logger.
type capturingHandler struct {
	records []slog.Record
}

func (me *capturingHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (me *capturingHandler) Handle(_ context.Context, record slog.Record) error {
	me.records = append(me.records, record)
	return nil
}

func (me *capturingHandler) WithAttrs([]slog.Attr) slog.Handler {
	return me
}

func (me *capturingHandler) WithGroup(string) slog.Handler {
	return me
}

// Installs a capturing handler as the default slog logger for the duration of the test.
func captureDefaultLogger(t *testing.T) *capturingHandler {
	t.Helper()
	handler := &capturingHandler{}
	previous := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})
	return handler
}

// An error out of main goes to the default slog logger at Error level. Anything routing slog
// somewhere else, which is the whole reason for using the default, should see it there.
func TestOnMainReturnedLogsToDefaultSlogLogger(t *testing.T) {
	handler := captureDefaultLogger(t)
	code := OnMainReturned(context.Background(), errors.New("it broke"))
	if code != 1 {
		t.Errorf("got exit code %v, want 1", code)
	}
	if len(handler.records) != 1 {
		t.Fatalf("got %v records, want 1", len(handler.records))
	}
	record := handler.records[0]
	if record.Level != slog.LevelError {
		t.Errorf("got level %v, want %v", record.Level, slog.LevelError)
	}
	if !strings.Contains(record.Message, "it broke") {
		t.Errorf("message %q doesn't contain the error", record.Message)
	}
}

// Returning nil isn't worth a word.
func TestOnMainReturnedNilLogsNothing(t *testing.T) {
	handler := captureDefaultLogger(t)
	code := OnMainReturned(context.Background(), nil)
	if code != 0 {
		t.Errorf("got exit code %v, want 0", code)
	}
	if len(handler.records) != 0 {
		t.Errorf("got %v records, want none", len(handler.records))
	}
}

// Neither is returning the reason the context was cancelled.
func TestOnMainReturnedContextCauseLogsNothing(t *testing.T) {
	handler := captureDefaultLogger(t)
	cause := errors.New("interrupted")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	code := OnMainReturned(ctx, cause)
	if code != 0 {
		t.Errorf("got exit code %v, want 0", code)
	}
	if len(handler.records) != 0 {
		t.Errorf("got %v records, want none", len(handler.records))
	}
}

func TestOnMainReturnedSignalExitCode(t *testing.T) {
	captureDefaultLogger(t)
	code := OnMainReturned(context.Background(), SignalReceivedError{syscall.SIGINT})
	if want := 128 + int(syscall.SIGINT); code != want {
		t.Errorf("got exit code %v, want %v", code, want)
	}
}
