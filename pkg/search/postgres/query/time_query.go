package pgsearch

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/search"
)

const (
	dayDuration = 24 * time.Hour

	sqlTimeStampFormat = "2006-01-02 15:04:05 -07:00"
)

func newTimeQuery(column string, _ *search.Field, value string, modifiers ...search.QueryModifier) (*QueryEntry, error) {
	if len(modifiers) > 0 {
		return nil, errors.New("modifiers not supported for time query")
	}
	prefix, trimmedValue := parseNumericPrefix(value)
	var formattedTime string
	if t, ok := parseTimeString(trimmedValue); ok {
		// If the date query is a singular datetime with no prefix, then need to create a numeric query with the min = date. max = date + 1
		if prefix == "" {
			return &QueryEntry{
				Query:  fmt.Sprintf("%s >= $$ and %s < $$", column, column),
				Values: []interface{}{t.Format(sqlTimeStampFormat), t.Add(dayDuration).Format(sqlTimeStampFormat)},
			}, nil
		}
		formattedTime = t.Format(sqlTimeStampFormat)
	} else if d, ok := parseDuration(trimmedValue); ok {
		formattedTime = time.Now().Add(-d).Format(sqlTimeStampFormat)
		// Invert the prefix in a duration query, since if someone queries for >=3d
		// they mean more than 3 days ago, which means the timestamp should be
		// < the timestamp of 3 days ago.
		prefix = invertNumericPrefix(prefix)
	} else {
		return nil, fmt.Errorf("invalid time query (prefix: %s, value: %s). Must be of the format (01/02/2006 or 1d)", prefix, trimmedValue)
	}

	return &QueryEntry{
		Query:  fmt.Sprintf("%s %s $$", column, prefix),
		Values: []interface{}{formattedTime},
	}, nil
}

func parseDuration(d string) (time.Duration, bool) {
	d = strings.TrimSuffix(d, "d")
	days, err := strconv.ParseInt(d, 10, 32)
	if err != nil {
		return time.Second, false
	}
	return time.Duration(days) * dayDuration, true
}

func parseTimeString(value string) (time.Time, bool) {
	layouts := []string{"01/02/2006 3:04:05 PM MST", "01/02/2006 3:04 PM MST", "01/02/2006 3:04:05 PM", "01/02/2006 3:04 PM", "01/02/2006 MST", "01/02/2006"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
