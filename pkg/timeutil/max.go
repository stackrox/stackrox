package timeutil

import "time"

var (
	// Max is the maximum value that can be represented in a `time.Time` object.
	Max = time.Unix(1<<63-62135596801, 999999999)

	// MaxProtoValid is the maximum value that can be represented in a `time.Time` object and converted to a proto
	// timestamp without errors.
	MaxProtoValid = time.Date(9999, 12, 31, 23, 59, 59, 999999999, time.UTC)
)
