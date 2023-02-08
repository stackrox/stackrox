package timestamp

import (
	"strconv"
	"time"
)

var (
	buildTimestampUnixSecs string //XDef:BUILD_TIMESTAMP

	// BuildTimestamp is the time when this binary was built.
	// Deprecated: It will be removed in 4.0. Please do not use it.
	// TODO(ROX-14336): delete it
	BuildTimestamp time.Time
	// BuildTimestampParsingErr is the error encountered when parsing the build timestamp (if any).
	// Deprecated: It will be removed in 4.0. Please do not use it.
	// TODO(ROX-14336): delete it
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
