package protocompat

import (
	"unsafe"

	"github.com/gogo/protobuf/proto"
)

// ShallowClone performs a shallow copy on the given protobuf message.
// Don't use this unless you know exactly what you're doing and why it is justified for performance reasons. If unsure,
// use Clone() to perform a deep clone.
func ShallowClone[T any, M interface {
	proto.Message
	*T
}](msg M) *T {
	var result T
	msgSize := unsafe.Sizeof(result)

	// Use direct memory copy to bypass vet checks.
	msgBytes := unsafe.Slice((*byte)(unsafe.Pointer((*T)(msg))), msgSize)
	resultBytes := unsafe.Slice((*byte)(unsafe.Pointer(&result)), msgSize)
	copy(resultBytes, msgBytes)

	return &result
}
