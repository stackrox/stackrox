package blevesearch

import (
	"strconv"
	"strings"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/search/query"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/timeutil"
)

const (
	dayDuration = 24 * time.Hour
)

func newTimeQuery(_ v1.SearchCategory, field string, value string, _ ...queryModifier) (query.Query, error) {
	return newTimeQueryHelper(time.Now(), field, value)
}

func newTimeQueryHelper(now time.Time, field, value string) (query.Query, error) {
	prefix, trimmedValue := parseNumericPrefix(value)
	if t, ok := parseTimeString(trimmedValue); ok {
		// Adjust for the timezone offset when comparing
		seconds := t.Unix() - timeutil.TimeToOffset(t)

		// If the date query is a singular date with no prefix, then need to create a numeric query with the min = date. max = date + 1
		if prefix == "" {
			q := bleve.NewNumericRangeInclusiveQuery(floatPtr(float64(seconds)), floatPtr(float64(seconds)+dayDuration.Seconds()), boolPtr(true), boolPtr(true))
			q.SetField(field)
			return q, nil
		}
		return timeQueryWithSeconds(field, prefix, seconds), nil
	}

	lower, upper, err := maybeParseDurationAsRange(trimmedValue)
	if err == nil {
		q := bleve.NewNumericRangeInclusiveQuery(
			floatPtr(float64(now.Add(-upper).Unix())), floatPtr(float64(now.Add(-lower).Unix())), boolPtr(false), boolPtr(false),
		)
		q.SetField(field)
		return q, nil
	} else if err != errNotARange {
		return nil, errors.Wrapf(err, "tried to parse %s as a range, but it was not valid", value)
	}

	if d, ok := parseDuration(trimmedValue); ok {
		// Invert the prefix in a duration query, since if someone queries for >=3d
		// they mean more than 3 days ago, which means the timestamp should be
		// <= the timestamp of 3 days ago.
		return timeQueryWithSeconds(field, invertNumericPrefix(prefix), now.Add(-d).Unix()), nil
	}
	return nil, errors.New("Invalid time query. Must be of the format (01/02/2006 or 1d)")
}

func timeQueryWithSeconds(field, prefix string, seconds int64) query.Query {
	return createNumericQuery(field, prefix, floatPtr(float64(seconds)))
}

func maybeParseDurationAsRange(value string) (lower, upper time.Duration, err error) {
	// Split the value into two parts, separated by a hyphen.
	// We need to be careful to ensure that we don't mistake
	// hyphens for minus signs.
	for i, char := range value {
		if i == 0 {
			continue
		}
		if char == '-' {
			var valid bool
			lower, valid = parseDuration(value[:i])
			if !valid {
				return 0, 0, errNotARange
			}
			upper, valid = parseDuration(value[i+1:])
			if !valid {
				return 0, 0, errNotARange
			}
			if lower >= upper {
				return 0, 0, errors.Errorf("invalid range %s (first value must be strictly less than the second)", value)
			}
			return lower, upper, nil
		}
	}
	return 0, 0, errNotARange
}

func parseDuration(d string) (time.Duration, bool) {
	d = strings.TrimSuffix(d, "d")
	days, err := strconv.ParseInt(d, 10, 32)
	if err != nil {
		return 0, false
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
