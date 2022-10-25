package proto

import golangProto "github.com/golang/protobuf/proto"

// The following function aliases refer to the golang proto package, which already offers a compatibility layer.
// While those functions are deprecated and to be replaced mostly by invocations to protoregistry, a migration
// can only be performed once the code using it is fully switched over to the V2 message format.
var (
	FileDescriptor = golangProto.FileDescriptor
	EnumValueMap   = golangProto.EnumValueMap
	MessageType    = golangProto.MessageType
	MessageName    = golangProto.MessageName
)
