package shallowclone

import (
	"unsafe"

	"google.golang.org/protobuf/proto"
)

// UnsafeShallowClone performs a shallow clone of the given pointee.
func UnsafeShallowClone[T any, M interface {
	proto.Message
	*T
}](msg *T) *T {
	var ret T
	dstMem := unsafe.Slice((*byte)(unsafe.Pointer(&ret)), unsafe.Sizeof(ret))
	srcMem := unsafe.Slice((*byte)(unsafe.Pointer(msg)), unsafe.Sizeof(*msg))
	copy(dstMem, srcMem)
	return &ret
}
