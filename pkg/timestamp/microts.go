package timestamp

import (
	"math"
	"sync/atomic"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stackrox/rox/pkg/transitional/protocompat/types"
)

const (
	microsecondsPerSecond     = 1000000
	nanosecondsPerMicrosecond = 1000
)

// MicroTS is a microsecond-granularity Unix UTC timestamp.
type MicroTS int64

// InfiniteFuture is a microtimestamp that is greater (or equal) to any other microtimestamp.
const InfiniteFuture MicroTS = math.MaxInt64

// Now returns the current time as a microtimestamp.
func Now() MicroTS {
	return FromGoTime(time.Now())
}

// LoadAtomic atomically gets this timestamp value.
func (ts *MicroTS) LoadAtomic() MicroTS {
	return MicroTS(atomic.LoadInt64((*int64)(ts)))
}

// StoreAtomic atomically sets this timestamp value.
func (ts *MicroTS) StoreAtomic(newTS MicroTS) {
	atomic.StoreInt64((*int64)(ts), int64(newTS))
}

// CompareAndSwapAtomic atomically sets the value of this timestamp value to newTS if its current value matches oldTS.
// The return value indicates whether a swap happened.
func (ts *MicroTS) CompareAndSwapAtomic(oldTS, newTS MicroTS) bool {
	return atomic.CompareAndSwapInt64((*int64)(ts), int64(oldTS), int64(newTS))
}

// GoTime returns this microtimestamp as a `time.Time` object.
func (ts MicroTS) GoTime() time.Time {
	return time.Unix(ts.UnixSeconds(), int64(ts.UnixNanosFraction()))
}

// UnixSeconds returns the number of seconds since Unix epoch represented by this microtimestamp.
func (ts MicroTS) UnixSeconds() int64 {
	return int64(ts) / microsecondsPerSecond
}

// UnixNanos returns the number of nanoseconds since Unix epoch represented by this microtimestamp.
func (ts MicroTS) UnixNanos() int64 {
	return int64(ts) * nanosecondsPerMicrosecond
}

// After returns whether the timestamp is after otherTS.
func (ts MicroTS) After(otherTS MicroTS) bool {
	return ts > otherTS
}

// UnixNanosFraction returns the number of nanoseconds since the last full second.
func (ts MicroTS) UnixNanosFraction() int32 {
	return int32(ts%microsecondsPerSecond) * nanosecondsPerMicrosecond
}

// Protobuf converts this microtimestamp to a (Google) protobuf representation.
func (ts MicroTS) Protobuf() *timestamp.Timestamp {
	return &timestamp.Timestamp{
		Seconds: ts.UnixSeconds(),
		Nanos:   ts.UnixNanosFraction(),
	}
}

// GogoProtobuf converts this microtimestamp to a (Gogo) protobuf representation.
func (ts MicroTS) GogoProtobuf() *types.Timestamp {
	return &types.Timestamp{
		Seconds: ts.UnixSeconds(),
		Nanos:   ts.UnixNanosFraction(),
	}
}

// ElapsedSince returns the time elapsed since the given timestamp, as a `time.Duration`.
func (ts MicroTS) ElapsedSince(otherTS MicroTS) time.Duration {
	return time.Duration(ts-otherTS) * time.Microsecond
}

// Add adds the given `time.Duration` to this microtimestamp, and returns a new microtimestamp.
func (ts MicroTS) Add(duration time.Duration) MicroTS {
	return ts + MicroTS(duration/time.Microsecond)
}

// FromGoTime converts the given `time.Time` object to a microtimestamp.
func FromGoTime(t time.Time) MicroTS {
	return MicroTS(t.UnixNano() / nanosecondsPerMicrosecond)
}

// ProtoTimestamp is a common interface for timestamp protobuf objects (satisfied by both Google and Gogo protobuf
// libraries).
type ProtoTimestamp interface {
	GetSeconds() int64
	GetNanos() int32
}

// FromProtobuf converts the given protobuf timestamp message to a microtimestamp.
func FromProtobuf(ts ProtoTimestamp) MicroTS {
	return MicroTS(ts.GetSeconds()*microsecondsPerSecond + int64(ts.GetNanos()/nanosecondsPerMicrosecond))
}
