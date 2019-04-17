package timestamp

import (
	"strconv"
	"time"
)

var (
	buildTimestampUnixSecs string

	// BuildTimestamp is the time when this binary was built.
	BuildTimestamp time.Time
	// BuildTimestampParsingErr is the error encountered when parsing the build timestamp (if any).
	BuildTimestampParsingErr error
)

func parseUnixSecsString(str string) (time.Time, error) {
	unixSecs, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(unixSecs, 0), nil
}

func init() {
	// Data might not be available when, e.g., running tests via Goland.
	BuildTimestamp, BuildTimestampParsingErr = parseUnixSecsString(buildTimestampUnixSecs)
}
