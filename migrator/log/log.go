package log

import (
	"fmt"
	"os"
	"time"
)

// WriteToStderr is a helper function to write to stderr.
func WriteToStderr(s string) {
	_, _ = fmt.Fprint(os.Stderr, fmt.Sprintf("Migrator: %s: %s\n", time.Now().Format("2006/01/02 15:04:05"), s))
}

// WriteToStderrf writes to stderr with a format string.
func WriteToStderrf(format string, args ...interface{}) {
	WriteToStderr(fmt.Sprintf(format, args...))
}
