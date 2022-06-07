package maincommand

import (
	"bytes"

	"github.com/stackrox/rox/roxctl/common"
)

// errorWriter implements io.Writer that could be passed to Cobra to handle colorful printing for error messages.
// It replaces Cobra error prefix with our own defined in Logger.
type errorWriter struct {
	logger common.Logger
}

func (e errorWriter) Write(p []byte) (n int, err error) {
	e.logger.ErrfLn("%s", bytes.TrimRight(bytes.TrimPrefix(p, []byte("Error: ")), "\n"))
	return len(p), nil
}
