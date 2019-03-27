package blevesearch

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/tkuchiki/go-timezone"
)

const (
	dayDuration = 24 * time.Hour
)

func newTimeQuery(_ v1.SearchCategory, field string, value string) (query.Query, error) {
	prefix, trimmedValue := parseNumericPrefix(value)
	var seconds int64
	if t, ok := parseTimeString(trimmedValue); ok {
		// Adjust for the timezone offset when comparing
		seconds := t.Unix() - timeToOffset(t)

		// If the date query is a singular date with no prefix, then need to create a numeric query with the min = date. max = date + 1
		if prefix == "" {
			q := bleve.NewNumericRangeInclusiveQuery(floatPtr(float64(seconds)), floatPtr(float64(seconds)+dayDuration.Seconds()), boolPtr(true), boolPtr(true))
			q.SetField(field)
			return q, nil
		}
	} else if d, ok := parseDuration(trimmedValue); ok {
		seconds = time.Now().Add(-d).Unix()
	} else {
		return nil, fmt.Errorf("Invalid time query. Must be of the format (01/02/2006 or 1d)")
	}
	floatSeconds := float64(seconds)
	return createNumericQuery(field, prefix, &floatSeconds), nil
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
	if t, err := time.Parse("01/02/2006 MST", value); err == nil {
		return t, true
	}
	if t, err := time.Parse("01/02/2006", value); err == nil {
		return t, true
	}
	return time.Now(), false
}

func timeToOffset(t time.Time) int64 {
	tz, _ := t.Zone()
	offset, err := timezone.GetOffset(tz, false)
	if err != nil {
		return 0
	}
	return int64(offset)
}
