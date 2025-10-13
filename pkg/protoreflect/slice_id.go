package protoreflect

import "unsafe"

// sliceIdentity is a struct that identifies a slice and can be used as a key in maps.
type sliceIdentity struct {
	base   uintptr
	length int
}

func identityOfSlice(slice []byte) sliceIdentity {
	if len(slice) == 0 {
		return sliceIdentity{}
	}
	return sliceIdentity{
		//#nosec G103
		base:   uintptr(unsafe.Pointer(&slice[0])),
		length: len(slice),
	}
}
