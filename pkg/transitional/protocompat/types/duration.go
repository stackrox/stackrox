package types

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
)

// Duration is an alias for the durationpb.Duration type
type Duration = durationpb.Duration

// DurationProto converts a time.Duration to a Duration proto.
func DurationProto(d time.Duration) *Duration {
	return durationpb.New(d)
}

// DurationFromProto returns a time.Duration for a Duration protobuf object. If the protobuf object is not valid,
// an error is returned.
func DurationFromProto(pb *Duration) (time.Duration, error) {
	return pb.AsDuration(), pb.CheckValid()
}
