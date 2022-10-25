package types

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// Timestamp is an alias for the timestamppb.Timestamp type
type Timestamp = timestamppb.Timestamp

// TimestampProto converts a time.Time to a Timestamp proto. If the resulting Timestamp is not valid,
// an error is returned.
func TimestampProto(t time.Time) (*Timestamp, error) {
	ts := timestamppb.New(t)
	return ts, ts.CheckValid()
}

// TimestampFromProto returns a time.Time for a Timestamp protobuf object. If the protobuf object is not valid,
// an error is returned.
func TimestampFromProto(pb *Timestamp) (time.Time, error) {
	return pb.AsTime(), pb.CheckValid()
}

// TimestampNow returns a Timestamp protobuf object for the current time.
func TimestampNow() *Timestamp {
	return timestamppb.Now()
}
