// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v4.25.3
// source: api/v1/common.proto

package v1

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

type ResourceByID struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ResourceByID) Reset() {
	*x = ResourceByID{}
	mi := &file_api_v1_common_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ResourceByID) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ResourceByID) ProtoMessage() {}

func (x *ResourceByID) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_common_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ResourceByID.ProtoReflect.Descriptor instead.
func (*ResourceByID) Descriptor() ([]byte, []int) {
	return file_api_v1_common_proto_rawDescGZIP(), []int{0}
}

func (x *ResourceByID) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

// EXPERIMENTAL.
//
// Used in combination with MutabilityMode.ALLOW_MUTATE_FORCED to enable forced removal.
type DeleteByIDWithForce struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Force         bool                   `protobuf:"varint,2,opt,name=force,proto3" json:"force,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DeleteByIDWithForce) Reset() {
	*x = DeleteByIDWithForce{}
	mi := &file_api_v1_common_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DeleteByIDWithForce) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteByIDWithForce) ProtoMessage() {}

func (x *DeleteByIDWithForce) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_common_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteByIDWithForce.ProtoReflect.Descriptor instead.
func (*DeleteByIDWithForce) Descriptor() ([]byte, []int) {
	return file_api_v1_common_proto_rawDescGZIP(), []int{1}
}

func (x *DeleteByIDWithForce) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *DeleteByIDWithForce) GetForce() bool {
	if x != nil {
		return x.Force
	}
	return false
}

var File_api_v1_common_proto protoreflect.FileDescriptor

const file_api_v1_common_proto_rawDesc = "" +
	"\n" +
	"\x13api/v1/common.proto\x12\x02v1\"\x1e\n" +
	"\fResourceByID\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\";\n" +
	"\x13DeleteByIDWithForce\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12\x14\n" +
	"\x05force\x18\x02 \x01(\bR\x05forceB'\n" +
	"\x18io.stackrox.proto.api.v1Z\v./api/v1;v1b\x06proto3"

var (
	file_api_v1_common_proto_rawDescOnce sync.Once
	file_api_v1_common_proto_rawDescData []byte
)

func file_api_v1_common_proto_rawDescGZIP() []byte {
	file_api_v1_common_proto_rawDescOnce.Do(func() {
		file_api_v1_common_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_api_v1_common_proto_rawDesc), len(file_api_v1_common_proto_rawDesc)))
	})
	return file_api_v1_common_proto_rawDescData
}

var file_api_v1_common_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_api_v1_common_proto_goTypes = []any{
	(*ResourceByID)(nil),        // 0: v1.ResourceByID
	(*DeleteByIDWithForce)(nil), // 1: v1.DeleteByIDWithForce
}
var file_api_v1_common_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_api_v1_common_proto_init() }
func file_api_v1_common_proto_init() {
	if File_api_v1_common_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_api_v1_common_proto_rawDesc), len(file_api_v1_common_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_api_v1_common_proto_goTypes,
		DependencyIndexes: file_api_v1_common_proto_depIdxs,
		MessageInfos:      file_api_v1_common_proto_msgTypes,
	}.Build()
	File_api_v1_common_proto = out.File
	file_api_v1_common_proto_goTypes = nil
	file_api_v1_common_proto_depIdxs = nil
}
