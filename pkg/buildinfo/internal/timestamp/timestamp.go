package timestamp

import (
	"time"
)

var (
	buildTimestampRFC3339 string

	// BuildTimestamp is the time when this binary was built.
	BuildTimestamp time.Time
	// BuildTimestampParsingErr is the error encountered when parsing the build timestamp (if any).
	BuildTimestampParsingErr error
)

func init() {
	// Data might not be available when, e.g., running tests via Goland.
	BuildTimestamp, BuildTimestampParsingErr = time.Parse(time.RFC3339, buildTimestampRFC3339)
}
