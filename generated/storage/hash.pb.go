// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v4.25.3
// source: storage/hash.proto

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

type Hash struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ClusterId     string                 `protobuf:"bytes,1,opt,name=cluster_id,json=clusterId,proto3" json:"cluster_id,omitempty" sql:"pk"` // @gotags: sql:"pk"
	Hashes        map[string]uint64      `protobuf:"bytes,2,rep,name=hashes,proto3" json:"hashes,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"varint,2,opt,name=value"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *Hash) Reset() {
	*x = Hash{}
	mi := &file_storage_hash_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Hash) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Hash) ProtoMessage() {}

func (x *Hash) ProtoReflect() protoreflect.Message {
	mi := &file_storage_hash_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Hash.ProtoReflect.Descriptor instead.
func (*Hash) Descriptor() ([]byte, []int) {
	return file_storage_hash_proto_rawDescGZIP(), []int{0}
}

func (x *Hash) GetClusterId() string {
	if x != nil {
		return x.ClusterId
	}
	return ""
}

func (x *Hash) GetHashes() map[string]uint64 {
	if x != nil {
		return x.Hashes
	}
	return nil
}

var File_storage_hash_proto protoreflect.FileDescriptor

const file_storage_hash_proto_rawDesc = "" +
	"\n" +
	"\x12storage/hash.proto\x12\astorage\"\x93\x01\n" +
	"\x04Hash\x12\x1d\n" +
	"\n" +
	"cluster_id\x18\x01 \x01(\tR\tclusterId\x121\n" +
	"\x06hashes\x18\x02 \x03(\v2\x19.storage.Hash.HashesEntryR\x06hashes\x1a9\n" +
	"\vHashesEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\x04R\x05value:\x028\x01B.\n" +
	"\x19io.stackrox.proto.storageZ\x11./storage;storageb\x06proto3"

var (
	file_storage_hash_proto_rawDescOnce sync.Once
	file_storage_hash_proto_rawDescData []byte
)

func file_storage_hash_proto_rawDescGZIP() []byte {
	file_storage_hash_proto_rawDescOnce.Do(func() {
		file_storage_hash_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_storage_hash_proto_rawDesc), len(file_storage_hash_proto_rawDesc)))
	})
	return file_storage_hash_proto_rawDescData
}

var file_storage_hash_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_storage_hash_proto_goTypes = []any{
	(*Hash)(nil), // 0: storage.Hash
	nil,          // 1: storage.Hash.HashesEntry
}
var file_storage_hash_proto_depIdxs = []int32{
	1, // 0: storage.Hash.hashes:type_name -> storage.Hash.HashesEntry
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_storage_hash_proto_init() }
func file_storage_hash_proto_init() {
	if File_storage_hash_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_storage_hash_proto_rawDesc), len(file_storage_hash_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_storage_hash_proto_goTypes,
		DependencyIndexes: file_storage_hash_proto_depIdxs,
		MessageInfos:      file_storage_hash_proto_msgTypes,
	}.Build()
	File_storage_hash_proto = out.File
	file_storage_hash_proto_goTypes = nil
	file_storage_hash_proto_depIdxs = nil
}
