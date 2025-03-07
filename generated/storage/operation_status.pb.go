// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.5
// 	protoc        v4.25.3
// source: storage/operation_status.proto

package storage

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type OperationStatus int32

const (
	OperationStatus_FAIL OperationStatus = 0
	OperationStatus_PASS OperationStatus = 1
)

// Enum value maps for OperationStatus.
var (
	OperationStatus_name = map[int32]string{
		0: "FAIL",
		1: "PASS",
	}
	OperationStatus_value = map[string]int32{
		"FAIL": 0,
		"PASS": 1,
	}
)

func (x OperationStatus) Enum() *OperationStatus {
	p := new(OperationStatus)
	*p = x
	return p
}

func (x OperationStatus) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (OperationStatus) Descriptor() protoreflect.EnumDescriptor {
	return file_storage_operation_status_proto_enumTypes[0].Descriptor()
}

func (OperationStatus) Type() protoreflect.EnumType {
	return &file_storage_operation_status_proto_enumTypes[0]
}

func (x OperationStatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use OperationStatus.Descriptor instead.
func (OperationStatus) EnumDescriptor() ([]byte, []int) {
	return file_storage_operation_status_proto_rawDescGZIP(), []int{0}
}

var File_storage_operation_status_proto protoreflect.FileDescriptor

var file_storage_operation_status_proto_rawDesc = string([]byte{
	0x0a, 0x1e, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x2f, 0x6f, 0x70, 0x65, 0x72, 0x61, 0x74,
	0x69, 0x6f, 0x6e, 0x5f, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x07, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x2a, 0x25, 0x0a, 0x0f, 0x4f, 0x70, 0x65,
	0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x08, 0x0a, 0x04,
	0x46, 0x41, 0x49, 0x4c, 0x10, 0x00, 0x12, 0x08, 0x0a, 0x04, 0x50, 0x41, 0x53, 0x53, 0x10, 0x01,
	0x42, 0x2e, 0x0a, 0x19, 0x69, 0x6f, 0x2e, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x72, 0x6f, 0x78, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x5a, 0x11, 0x2e,
	0x2f, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x3b, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
})

var (
	file_storage_operation_status_proto_rawDescOnce sync.Once
	file_storage_operation_status_proto_rawDescData []byte
)

func file_storage_operation_status_proto_rawDescGZIP() []byte {
	file_storage_operation_status_proto_rawDescOnce.Do(func() {
		file_storage_operation_status_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_storage_operation_status_proto_rawDesc), len(file_storage_operation_status_proto_rawDesc)))
	})
	return file_storage_operation_status_proto_rawDescData
}

var file_storage_operation_status_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_storage_operation_status_proto_goTypes = []any{
	(OperationStatus)(0), // 0: storage.OperationStatus
}
var file_storage_operation_status_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_storage_operation_status_proto_init() }
func file_storage_operation_status_proto_init() {
	if File_storage_operation_status_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_storage_operation_status_proto_rawDesc), len(file_storage_operation_status_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_storage_operation_status_proto_goTypes,
		DependencyIndexes: file_storage_operation_status_proto_depIdxs,
		EnumInfos:         file_storage_operation_status_proto_enumTypes,
	}.Build()
	File_storage_operation_status_proto = out.File
	file_storage_operation_status_proto_goTypes = nil
	file_storage_operation_status_proto_depIdxs = nil
}
