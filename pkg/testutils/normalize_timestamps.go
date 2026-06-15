package testutils

import (
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NormalizeTimestampsToMicros truncates all google.protobuf.Timestamp fields
// in a proto message to microsecond precision. PostgreSQL stores timestamps
// with microsecond precision, so sub-microsecond values are lost on round-trip.
func NormalizeTimestampsToMicros(msg proto.Message) {
	normalizeTimestamps(msg.ProtoReflect())
}

func normalizeTimestamps(m protoreflect.Message) {
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		if fd.Kind() != protoreflect.MessageKind {
			return true
		}
		if fd.IsList() {
			list := v.List()
			for i := 0; i < list.Len(); i++ {
				normalizeTimestamps(list.Get(i).Message())
			}
			return true
		}
		if fd.IsMap() {
			return true
		}
		inner := v.Message()
		if inner.Descriptor().FullName() == "google.protobuf.Timestamp" {
			ts := inner.Interface().(*timestamppb.Timestamp)
			if ts != nil {
				goTime := ts.AsTime().Truncate(time.Microsecond)
				ts.Seconds = goTime.Unix()
				ts.Nanos = int32(goTime.Nanosecond())
			}
			return true
		}
		normalizeTimestamps(inner)
		return true
	})
}
