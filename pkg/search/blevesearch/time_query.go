package blevesearch

import (
	"strconv"
	"strings"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/timeutil"
)

const (
	dayDuration = 24 * time.Hour
)

func newTimeQuery(_ v1.SearchCategory, field string, value string, modifiers ...queryModifier) (query.Query, error) {
	prefix, trimmedValue := parseNumericPrefix(value)
	var seconds int64
	if t, ok := parseTimeString(trimmedValue); ok {
		// Adjust for the timezone offset when comparing
		seconds = t.Unix() - timeutil.TimeToOffset(t)

		// If the date query is a singular date with no prefix, then need to create a numeric query with the min = date. max = date + 1
		if prefix == "" {
			q := bleve.NewNumericRangeInclusiveQuery(floatPtr(float64(seconds)), floatPtr(float64(seconds)+dayDuration.Seconds()), boolPtr(true), boolPtr(true))
			q.SetField(field)
			return q, nil
		}
	} else if d, ok := parseDuration(trimmedValue); ok {
		seconds = time.Now().Add(-d).Unix()
		// Invert the prefix in a duration query, since if someone queries for >=3d
		// they mean more than 3 days ago, which means the timestamp should be
		// < the timestamp of 3 days ago.
		prefix = invertNumericPrefix(prefix)
	} else {
		return nil, errors.New("Invalid time query. Must be of the format (01/02/2006 or 1d)")
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
	layouts := []string{"01/02/2006 3:04:05 PM MST", "01/02/2006 MST", "01/02/2006"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
