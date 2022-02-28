package timeutil

import (
	"time"

	"github.com/tkuchiki/go-timezone"
)

var (
	tzInfo = timezone.New()
)

// TimeToOffset calculates returns the time offset for current time zone.
func TimeToOffset(t time.Time) int64 {
	tz, _ := t.Zone()
	infos, _ := tzInfo.GetTzAbbreviationInfo(tz)
	if len(infos) == 0 {
		return 0
	}

	// We can tolerate ambiguities, but only if all timezones have the same offset.
	offset := infos[0].Offset()
	for _, info := range infos[1:] {
		if info.Offset() != offset {
			return 0
		}
	}

	return int64(offset)
}
