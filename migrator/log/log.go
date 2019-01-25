package log

import (
	"fmt"
	"os"
	"time"
)

// WriteToStderr is a helper function to write to stderr.
func WriteToStderr(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, fmt.Sprintf("Migrator: %s: %s\n", time.Now().Format("2006/01/02 15:04:05"), fmt.Sprintf(format, args...)))
}
