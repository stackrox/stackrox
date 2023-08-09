package pgsearch

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/stringutils"
)

const (
	dayDuration = 24 * time.Hour

	sqlTimeStampFormat = "2006-01-02 15:04:05 -07:00"
)

func newTimeQuery(ctx *queryAndFieldContext) (*QueryEntry, error) {
	if len(ctx.queryModifiers) > 0 {
		return nil, errors.New("modifiers not supported for time query")
	}
	from, to, ok, err := parseTimeRange(ctx.value)
	if err != nil {
		return nil, errors.Wrap(err, "parsing time range")
	}
	if ok {
		return qeWithSelectFieldIfNeeded(ctx, &WhereClause{
			Query:  fmt.Sprintf("%s >= $$ and %s < $$", ctx.qualifiedColumnName, ctx.qualifiedColumnName),
			Values: []interface{}{from.Format(sqlTimeStampFormat), to.Format(sqlTimeStampFormat)},
		}, nil), nil
	}

	prefix, trimmedValue := parseNumericPrefix(ctx.value)
	if t, ok := parseTimeString(trimmedValue); ok {
		// If the date query is a singular datetime with no prefix, then need to create a numeric query with the min = date. max = date + 1
		if prefix == "" {
			return qeWithSelectFieldIfNeeded(ctx, &WhereClause{
				Query:  fmt.Sprintf("%s >= $$ and %s < $$", ctx.qualifiedColumnName, ctx.qualifiedColumnName),
				Values: []interface{}{t.Format(sqlTimeStampFormat), t.Add(dayDuration).Format(sqlTimeStampFormat)},
			}, nil), nil
		}
		return timeQueryEntry(ctx, prefix, t.Format(sqlTimeStampFormat)), nil
	}

	lower, upper, err := maybeParseDurationAsRange(trimmedValue)
	if err == nil {
		return qeWithSelectFieldIfNeeded(ctx, &WhereClause{
			Query:  fmt.Sprintf("%s > $$ and %s < $$", ctx.qualifiedColumnName, ctx.qualifiedColumnName),
			Values: []interface{}{ctx.now.Add(-upper).Format(sqlTimeStampFormat), ctx.now.Add(-lower).Format(sqlTimeStampFormat)},
		}, nil), nil
	} else if err != errNotARange {
		return nil, fmt.Errorf("tried to parse %s as a range, but it was not valid: %w", ctx.value, err)
	}

	if d, ok := parseDuration(trimmedValue); ok {
		// Invert the prefix in a duration query, since if someone queries for >=3d
		// they mean more than 3 days ago, which means the timestamp should be
		// < the timestamp of 3 days ago.
		if prefix == "" || prefix == "==" {
			prefix = "="
		} else {
			prefix = invertNumericPrefix(prefix)
		}
		return timeQueryEntry(ctx, prefix, ctx.now.Add(-d).Format(sqlTimeStampFormat)), nil
	}

	return nil, fmt.Errorf("invalid time query (prefix: %s, value: %s). Must be of the format (01/02/2006 or 1d)", prefix, trimmedValue)
}

func timeQueryEntry(ctx *queryAndFieldContext, prefix, formattedTime string) *QueryEntry {
	return qeWithSelectFieldIfNeeded(ctx, &WhereClause{
		Query:  fmt.Sprintf("%s %s $$", ctx.qualifiedColumnName, prefix),
		Values: []interface{}{formattedTime},
	}, nil)
}

func parseTimeRange(value string) (from, to time.Time, ok bool, err error) {
	if !strings.HasPrefix(value, search.TimeRangePrefix) {
		return
	}
	value = value[len(search.TimeRangePrefix):]

	fromStr, toStr := stringutils.Split2(value, "-")
	if fromStr == "" || toStr == "" {
		err = errors.Errorf("malformed time range query string: %s", value)
		return
	}

	fromMillis, err := strconv.ParseInt(fromStr, 10, 64)
	if err != nil {
		return
	}
	toMillis, err := strconv.ParseInt(toStr, 10, 64)
	if err != nil {
		return
	}
	return time.UnixMilli(fromMillis).UTC(), time.UnixMilli(toMillis).UTC(), true, nil
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
				return 0, 0, fmt.Errorf("invalid range %s (first value must be strictly less than the second)", value)
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
