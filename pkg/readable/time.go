package readable

import (
	"time"

	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

const (
	// ISO-8601 format.
	layout = "2006-01-02 15:04:05"
)

// Time takes a golang time type and converts it to a human readable string down to seconds
// It always print the UTC time.
func Time(t time.Time) string {
	return t.UTC().Format(layout)
}
