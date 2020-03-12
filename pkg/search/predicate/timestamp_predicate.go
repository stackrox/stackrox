package predicate

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/pkg/search"
	"github.com/tkuchiki/go-timezone"
)

const (
	dayDuration = 24 * time.Hour
)

func parseTimestamp(value string) (*types.Timestamp, *time.Duration, error) {
	if t, ok := parseTimeString(value); ok {
		// Adjust for the timezone offset when comparing
		seconds := t.Unix() - timeToOffset(t)
		return &types.Timestamp{
			Seconds: seconds,
		}, nil, nil
	}
	if d, ok := parseDuration(value); ok {
		return nil, &d, nil
	}
	return nil, nil, errors.New("Invalid time query. Must be of the format (01/02/2006 or 1d)")
}

func timestampComparator(cmp string) (func(interface{}, *types.Timestamp) bool, error) {
	switch cmp {
	case lessThanOrEqualTo:
		return func(instance interface{}, value *types.Timestamp) bool {
			return value.Compare(instance) >= 0
		}, nil
	case greaterThanOrEqualTo:
		return func(instance interface{}, value *types.Timestamp) bool {
			return value.Compare(instance) <= 0
		}, nil
	case lessThan:
		return func(instance interface{}, value *types.Timestamp) bool {
			return value.Compare(instance) > 0
		}, nil
	case greaterThan:
		return func(instance interface{}, value *types.Timestamp) bool {
			return value.Compare(instance) < 0
		}, nil
	case "":
		return func(instance interface{}, value *types.Timestamp) bool {
			return value.Compare(instance) == 0
		}, nil
	default:
		return nil, fmt.Errorf("unrecognized comparator: %s", cmp)
	}
}

func createTimestampPredicate(fullPath, value string) (internalPredicate, error) {
	if value == "-" {
		return alwaysFalse, nil
	}

	cmpStr, value := getNumericComparator(value)
	comparator, err := timestampComparator(cmpStr)
	if err != nil {
		return nil, err
	}
	timestampValue, durationValue, err := parseTimestamp(value)
	if err != nil {
		return nil, err
	}
	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		ts := timestampValue
		if durationValue != nil {
			var err error
			ts, err = types.TimestampProto(time.Now().Add(-*durationValue))
			if err != nil {
				return nil, false
			}
		}

		cmpResult := comparator(instance.Interface(), ts)
		if durationValue != nil {
			cmpResult = !cmpResult
		}
		if !cmpResult {
			return nil, false
		}
		instanceTS := instance.Interface().(*types.Timestamp)
		return &search.Result{
			Matches: formatSingleMatchf(fullPath, "%d", instanceTS.Seconds),
		}, true
	}), nil
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
