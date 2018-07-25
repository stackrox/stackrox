package protoconv

import (
	"time"

	gogoTimestamp "github.com/gogo/protobuf/types"
	golangTimestamp "github.com/golang/protobuf/ptypes/timestamp"
)

// CompareProtoTimestamps compares two of the proto timestamps
// This is necessary because the library has few equality checks
func CompareProtoTimestamps(t1, t2 *gogoTimestamp.Timestamp) int {
	if t1 == nil && t2 == nil {
		return 0
	}
	if t1 == nil {
		return -1
	}
	if t2 == nil {
		return 1
	}
	if t1.Seconds < t2.Seconds {
		return -1
	} else if t1.Seconds > t2.Seconds {
		return 1
	}
	if t1.Nanos < t2.Nanos {
		return -1
	} else if t1.Nanos > t2.Nanos {
		return 1
	}
	return 0
}

// ConvertGoGoProtoTimeToGolangProtoTime converts the Gogo Timestamp to the golang protobuf timestamp
func ConvertGoGoProtoTimeToGolangProtoTime(gogo *gogoTimestamp.Timestamp) *golangTimestamp.Timestamp {
	return &golangTimestamp.Timestamp{
		Seconds: gogo.GetSeconds(),
		Nanos:   gogo.GetNanos(),
	}
}

// ConvertTimestampToTime converts a proto timestamp to a golang Time
func ConvertTimestampToTime(gogo *gogoTimestamp.Timestamp) time.Time {
	t, err := gogoTimestamp.TimestampFromProto(gogo)
	if err != nil {
		return time.Now()
	}
	return t
}

// ConvertTimeToTimestamp converts golang time to proto timestamp
func ConvertTimeToTimestamp(goTime time.Time) *gogoTimestamp.Timestamp {
	t, err := gogoTimestamp.TimestampProto(goTime)
	if err != nil {
		return gogoTimestamp.TimestampNow()
	}
	return t
}
