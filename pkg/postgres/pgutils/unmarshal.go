package pgutils

import "github.com/stackrox/rox/pkg/env"

var useUnsafe = env.UseUnsafeUnmarshal.BooleanSetting()

// VTUnmarshaler is implemented by all vtprotobuf-generated message types.
type VTUnmarshaler interface {
	UnmarshalVT([]byte) error
	UnmarshalVTUnsafe([]byte) error
}

func UnmarshalVTMessage(msg VTUnmarshaler, data []byte) error {
	if useUnsafe {
		return msg.UnmarshalVTUnsafe(data)
	}
	return msg.UnmarshalVT(data)
}
