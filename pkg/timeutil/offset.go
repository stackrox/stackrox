package timeutil

import "time"

// TimeToOffset returns the UTC offset in seconds for the given time's zone.
func TimeToOffset(t time.Time) int64 {
	_, offset := t.Zone()
	return int64(offset)
}
