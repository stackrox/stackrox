package protoconv

import (
	"time"

	gogoTimestamp "github.com/gogo/protobuf/types"
	golangTimestamp "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// ConvertGoGoProtoTimeToGolangProtoTime converts the Gogo Timestamp to the golang protobuf timestamp
func ConvertGoGoProtoTimeToGolangProtoTime(gogo *gogoTimestamp.Timestamp) *golangTimestamp.Timestamp {
	if gogo == nil {
		return nil
	}
	return &golangTimestamp.Timestamp{
		Seconds: gogo.GetSeconds(),
		Nanos:   gogo.GetNanos(),
	}
}

// ConvertTimestampToTimeOrNow converts a proto timestamp to a golang Time, and returns time.Now() if there is an error.
func ConvertTimestampToTimeOrNow(gogo *gogoTimestamp.Timestamp) time.Time {
	return ConvertTimestampToTimeOrDefault(gogo, time.Now())
}

// ConvertTimestampToTimeOrDefault converts a proto timestamp to a golang Time, and returns the default value if there is an error.
func ConvertTimestampToTimeOrDefault(gogo *gogoTimestamp.Timestamp, defaultVal time.Time) time.Time {
	t, err := gogoTimestamp.TimestampFromProto(gogo)
	if err != nil {
		return defaultVal
	}
	return t
}

// ConvertTimeToTimestampOrNow converts golang time to proto timestamp.
func ConvertTimeToTimestampOrNow(goTime *time.Time) *gogoTimestamp.Timestamp {
	if goTime == nil {
		return gogoTimestamp.TimestampNow()
	}
	return ConvertTimeToTimestamp(*goTime)
}

// ConvertTimeToTimestamp converts golang time to proto timestamp.
func ConvertTimeToTimestamp(goTime time.Time) *gogoTimestamp.Timestamp {
	t, err := gogoTimestamp.TimestampProto(goTime)
	if err != nil {
		return gogoTimestamp.TimestampNow()
	}
	return t
}

// ConvertTimeToTimestampOrNil converts golang time to proto timestamp or if it fails returns nil.
func ConvertTimeToTimestampOrNil(goTime time.Time) *gogoTimestamp.Timestamp {
	t, err := gogoTimestamp.TimestampProto(goTime)
	if err != nil {
		log.Error(err)
		return nil
	}
	return t
}

// MustConvertTimeToTimestamp converts golang time to proto timestamp and panics if it fails.
func MustConvertTimeToTimestamp(goTime time.Time) *gogoTimestamp.Timestamp {
	t, err := gogoTimestamp.TimestampProto(goTime)
	if err != nil {
		panic(err)
	}
	return t
}
