package pgutils

import "github.com/stackrox/rox/pkg/env"

// VTUnmarshaler is implemented by all vtprotobuf-generated message types.
type VTUnmarshaler interface {
	UnmarshalVT([]byte) error
	UnmarshalVTUnsafe([]byte) error
}

//go:fix inline
func UnmarshalVTMessage(msg VTUnmarshaler, data []byte) error {
	if env.UseUnsafeUnmarshal.BooleanSetting() {
		return msg.UnmarshalVTUnsafe(data)
	}
	return msg.UnmarshalVT(data)
}
