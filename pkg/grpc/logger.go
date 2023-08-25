package grpc

import (
	"bytes"
	"os"

	"github.com/stackrox/rox/pkg/env"
)

var managedCentral = env.ManagedCentral.BooleanSetting()

// httpErrorLogger implements io.Writer interface. It is used to control
// error messages coming from http server which can be logged.
type httpErrorLogger struct {
}

// Write suppresses EOF error messages
func (l httpErrorLogger) Write(p []byte) (n int, err error) {
	if bytes.Contains(p, []byte("EOF")) {
		return len(p), nil
	}
	if managedCentral && bytes.Contains(p, []byte("error reading preface from client")) {
		return len(p), nil
	}
	return os.Stderr.Write(p)
}
