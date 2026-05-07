package pgutils

import (
	"reflect"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RoundTimestampsToMicroseconds rounds all protobuf Timestamp fields in a proto
// message to microsecond precision, matching Postgres timestamp column behavior.
// This modifies the message in-place and recurses into nested messages and slices.
func RoundTimestampsToMicroseconds(msg proto.Message) {
	if msg == nil {
		return
	}
	roundTimestamps(reflect.ValueOf(msg))
}

func roundTimestamps(v reflect.Value) {
	if !v.IsValid() {
		return
	}

	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		// Check if this is a timestamppb.Timestamp
		if ts, ok := v.Addr().Interface().(*timestamppb.Timestamp); ok && ts != nil {
			// Round nanos to nearest microsecond (not truncate)
			// This matches time.Time.Round(time.Microsecond) behavior
			remainder := ts.Nanos % 1000
			if remainder >= 500 {
				// Round up
				ts.Nanos = (ts.Nanos/1000 + 1) * 1000
				// Handle overflow into seconds
				if ts.Nanos >= 1_000_000_000 {
					ts.Seconds++
					ts.Nanos = 0
				}
			} else {
				// Round down
				ts.Nanos = (ts.Nanos / 1000) * 1000
			}
			return
		}

		// Recurse into struct fields
		for i := range v.NumField() {
			field := v.Field(i)
			if !field.CanInterface() {
				continue
			}
			roundTimestamps(field)
		}

	case reflect.Slice:
		// Recurse into slice elements
		for i := range v.Len() {
			roundTimestamps(v.Index(i))
		}

	case reflect.Map:
		// Recurse into map values
		iter := v.MapRange()
		for iter.Next() {
			roundTimestamps(iter.Value())
		}
	}
}
