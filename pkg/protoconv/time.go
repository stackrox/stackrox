package protoconv

import (
	"time"

	gogoTimestamp "github.com/gogo/protobuf/types"
	golangTimestamp "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/timestamp"
)

var (
	log = logging.LoggerForModule()
)

const (
	timeFormat         = "2006-01-02T15:04Z"
	extendedTimeFormat = "2006-01-02T15:04:03Z"
)

// ConvertGoGoProtoTimeToGolangProtoTime converts the Gogo Timestamp to the golang protobuf timestamp.
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

// ConvertTimeString converts a vulnerability time string into a proto timestamp
func ConvertTimeString(str string) *gogoTimestamp.Timestamp {
	if str == "" {
		return nil
	}
	if ts, err := time.Parse(timeFormat, str); err == nil {
		return ConvertTimeToTimestamp(ts)
	} else if ts, err := time.Parse(extendedTimeFormat, str); err == nil {
		return ConvertTimeToTimestamp(ts)
	}
	return nil
}

// ReadableTime takes a proto time type and converts it to a human readable string down to seconds.
// It prints a UTC time for valid input Timestamp objects.
func ReadableTime(ts *gogoTimestamp.Timestamp) string {
	t, err := protocompat.ConvertTimestampToTimeOrError(ts)
	if err != nil {
		log.Error(err)
		return "<malformed time>"
	}
	return t.UTC().Format(time.DateTime)
}

// NowMinus substracts a specified amount of time from the current timestamp
func NowMinus(t time.Duration) *gogoTimestamp.Timestamp {
	return ConvertTimeToTimestamp(time.Now().Add(-t))
}

// TimeBeforeDays subtracts a specified number of days from the current timestamp
func TimeBeforeDays(days int) *gogoTimestamp.Timestamp {
	return NowMinus(24 * time.Duration(days) * time.Hour)
}

// NowMinus substracts a specified amount of time from the current timestamp
func NowMinus(t time.Duration) *gogoTimestamp.Timestamp {
	return ConvertTimeToTimestamp(time.Now().Add(-t))
}

// TimeBeforeDays subtracts a specified number of days from the current timestamp
func TimeBeforeDays(days int) *gogoTimestamp.Timestamp {
	return NowMinus(24 * time.Duration(days) * time.Hour)
}

// ConvertMicroTSToProtobufTS converts a microtimestamp to a (Gogo) protobuf representation.
func ConvertMicroTSToProtobufTS(ts timestamp.MicroTS) *gogoTimestamp.Timestamp {
	return protocompat.GetProtoTimestampFromSecondsAndNanos(ts.UnixSeconds(), ts.UnixNanosFraction())
}
