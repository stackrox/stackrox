package basematchers

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/timeutil"
	"github.com/stackrox/rox/pkg/transitional/protocompat/types"
)

const (
	dayDuration = 24 * time.Hour
)

func parseTimestamp(value string) (*types.Timestamp, *time.Duration, error) {
	if t, ok := parseTimeString(value); ok {
		// Adjust for the timezone offset when comparing
		seconds := t.Unix() - timeutil.TimeToOffset(t)
		return &types.Timestamp{
			Seconds: seconds,
		}, nil, nil
	}
	if d, ok := parseDuration(value); ok {
		return nil, &d, nil
	}
	return nil, nil, errors.New("Invalid time query. Must be of the format (01/02/2006 or 1d)")
}

func timestampComparator(cmp string) (func(instance, value *types.Timestamp) bool, error) {
	switch cmp {
	case LessThanOrEqualTo:
		return func(instance, value *types.Timestamp) bool {
			return !value.AsTime().Before(instance.AsTime())
		}, nil
	case GreaterThanOrEqualTo:
		return func(instance, value *types.Timestamp) bool {
			return !value.AsTime().After(instance.AsTime())
		}, nil
	case LessThan:
		return func(instance, value *types.Timestamp) bool {
			return value.AsTime().After(instance.AsTime())
		}, nil
	case GreaterThan:
		return func(instance, value *types.Timestamp) bool {
			return value.AsTime().Before(instance.AsTime())
		}, nil
	case "":
		return func(instance, value *types.Timestamp) bool {
			return value.Compare(instance) == 0
		}, nil
	default:
		return nil, fmt.Errorf("unrecognized comparator: %s", cmp)
	}
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
	return time.Time{}, false
}
