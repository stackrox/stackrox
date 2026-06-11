package grpc

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type recordingLogger struct {
	logArgs  []any
	logfFmt  string
	logfArgs []any
}

func (r *recordingLogger) Log(args ...any) {
	r.logArgs = args
}

func (r *recordingLogger) Logf(format string, args ...any) {
	r.logfFmt = format
	r.logfArgs = args
}

func TestDebugLoggerImpl_Log(t *testing.T) {
	rec := &recordingLogger{}
	d := &debugLoggerImpl{log: rec}

	d.Log("hello", "world")

	assert.Equal(t, []any{"hello", "world"}, rec.logArgs)
}

func TestDebugLoggerImpl_Logf(t *testing.T) {
	rec := &recordingLogger{}
	d := &debugLoggerImpl{log: rec}

	d.Logf("count: %d, name: %s", 42, "test")

	assert.Equal(t, "count: %d, name: %s", rec.logfFmt)
	assert.Equal(t, []any{42, "test"}, rec.logfArgs)
	assert.Equal(t, "count: 42, name: test", fmt.Sprintf(rec.logfFmt, rec.logfArgs...))
}

func TestDebugLoggerImpl_NilSafe(t *testing.T) {
	var d *debugLoggerImpl
	d.Log("should not panic")
	d.Logf("should not panic: %d", 1)

	d = &debugLoggerImpl{}
	d.Log("should not panic")
	d.Logf("should not panic: %d", 1)
}
