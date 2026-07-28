package timeutil

import "time"

// TimeToOffset returns the UTC offset in seconds for the timezone of t.
func TimeToOffset(t time.Time) int64 {
	_, offset := t.Zone()
	return int64(offset)
}
