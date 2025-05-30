// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v4.25.3
// source: storage/delegated_registry_config.proto

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

type DelegatedRegistryConfig_EnabledFor int32

const (
	DelegatedRegistryConfig_NONE     DelegatedRegistryConfig_EnabledFor = 0
	DelegatedRegistryConfig_ALL      DelegatedRegistryConfig_EnabledFor = 1
	DelegatedRegistryConfig_SPECIFIC DelegatedRegistryConfig_EnabledFor = 2
)

// Enum value maps for DelegatedRegistryConfig_EnabledFor.
var (
	DelegatedRegistryConfig_EnabledFor_name = map[int32]string{
		0: "NONE",
		1: "ALL",
		2: "SPECIFIC",
	}
	DelegatedRegistryConfig_EnabledFor_value = map[string]int32{
		"NONE":     0,
		"ALL":      1,
		"SPECIFIC": 2,
	}
)

func (x DelegatedRegistryConfig_EnabledFor) Enum() *DelegatedRegistryConfig_EnabledFor {
	p := new(DelegatedRegistryConfig_EnabledFor)
	*p = x
	return p
}

func (x DelegatedRegistryConfig_EnabledFor) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (DelegatedRegistryConfig_EnabledFor) Descriptor() protoreflect.EnumDescriptor {
	return file_storage_delegated_registry_config_proto_enumTypes[0].Descriptor()
}

func (DelegatedRegistryConfig_EnabledFor) Type() protoreflect.EnumType {
	return &file_storage_delegated_registry_config_proto_enumTypes[0]
}

func (x DelegatedRegistryConfig_EnabledFor) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use DelegatedRegistryConfig_EnabledFor.Descriptor instead.
func (DelegatedRegistryConfig_EnabledFor) EnumDescriptor() ([]byte, []int) {
	return file_storage_delegated_registry_config_proto_rawDescGZIP(), []int{0, 0}
}

// DelegatedRegistryConfig determines how to handle scan requests.
//
// Refer to v1.DelegatedRegistryConfig for more detailed docs.
//
// Any changes made to this message must also be reflected in central/delegatedregistryconfig/convert/convert.go.
type DelegatedRegistryConfig struct {
	state            protoimpl.MessageState                       `protogen:"open.v1"`
	EnabledFor       DelegatedRegistryConfig_EnabledFor           `protobuf:"varint,1,opt,name=enabled_for,json=enabledFor,proto3,enum=storage.DelegatedRegistryConfig_EnabledFor" json:"enabled_for,omitempty"`
	DefaultClusterId string                                       `protobuf:"bytes,2,opt,name=default_cluster_id,json=defaultClusterId,proto3" json:"default_cluster_id,omitempty"`
	Registries       []*DelegatedRegistryConfig_DelegatedRegistry `protobuf:"bytes,3,rep,name=registries,proto3" json:"registries,omitempty"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *DelegatedRegistryConfig) Reset() {
	*x = DelegatedRegistryConfig{}
	mi := &file_storage_delegated_registry_config_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DelegatedRegistryConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DelegatedRegistryConfig) ProtoMessage() {}

func (x *DelegatedRegistryConfig) ProtoReflect() protoreflect.Message {
	mi := &file_storage_delegated_registry_config_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DelegatedRegistryConfig.ProtoReflect.Descriptor instead.
func (*DelegatedRegistryConfig) Descriptor() ([]byte, []int) {
	return file_storage_delegated_registry_config_proto_rawDescGZIP(), []int{0}
}

func (x *DelegatedRegistryConfig) GetEnabledFor() DelegatedRegistryConfig_EnabledFor {
	if x != nil {
		return x.EnabledFor
	}
	return DelegatedRegistryConfig_NONE
}

func (x *DelegatedRegistryConfig) GetDefaultClusterId() string {
	if x != nil {
		return x.DefaultClusterId
	}
	return ""
}

func (x *DelegatedRegistryConfig) GetRegistries() []*DelegatedRegistryConfig_DelegatedRegistry {
	if x != nil {
		return x.Registries
	}
	return nil
}

type DelegatedRegistryConfig_DelegatedRegistry struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Path          string                 `protobuf:"bytes,1,opt,name=path,proto3" json:"path,omitempty"`
	ClusterId     string                 `protobuf:"bytes,2,opt,name=cluster_id,json=clusterId,proto3" json:"cluster_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DelegatedRegistryConfig_DelegatedRegistry) Reset() {
	*x = DelegatedRegistryConfig_DelegatedRegistry{}
	mi := &file_storage_delegated_registry_config_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DelegatedRegistryConfig_DelegatedRegistry) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DelegatedRegistryConfig_DelegatedRegistry) ProtoMessage() {}

func (x *DelegatedRegistryConfig_DelegatedRegistry) ProtoReflect() protoreflect.Message {
	mi := &file_storage_delegated_registry_config_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DelegatedRegistryConfig_DelegatedRegistry.ProtoReflect.Descriptor instead.
func (*DelegatedRegistryConfig_DelegatedRegistry) Descriptor() ([]byte, []int) {
	return file_storage_delegated_registry_config_proto_rawDescGZIP(), []int{0, 0}
}

func (x *DelegatedRegistryConfig_DelegatedRegistry) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

func (x *DelegatedRegistryConfig_DelegatedRegistry) GetClusterId() string {
	if x != nil {
		return x.ClusterId
	}
	return ""
}

var File_storage_delegated_registry_config_proto protoreflect.FileDescriptor

const file_storage_delegated_registry_config_proto_rawDesc = "" +
	"\n" +
	"'storage/delegated_registry_config.proto\x12\astorage\"\xe0\x02\n" +
	"\x17DelegatedRegistryConfig\x12L\n" +
	"\venabled_for\x18\x01 \x01(\x0e2+.storage.DelegatedRegistryConfig.EnabledForR\n" +
	"enabledFor\x12,\n" +
	"\x12default_cluster_id\x18\x02 \x01(\tR\x10defaultClusterId\x12R\n" +
	"\n" +
	"registries\x18\x03 \x03(\v22.storage.DelegatedRegistryConfig.DelegatedRegistryR\n" +
	"registries\x1aF\n" +
	"\x11DelegatedRegistry\x12\x12\n" +
	"\x04path\x18\x01 \x01(\tR\x04path\x12\x1d\n" +
	"\n" +
	"cluster_id\x18\x02 \x01(\tR\tclusterId\"-\n" +
	"\n" +
	"EnabledFor\x12\b\n" +
	"\x04NONE\x10\x00\x12\a\n" +
	"\x03ALL\x10\x01\x12\f\n" +
	"\bSPECIFIC\x10\x02B.\n" +
	"\x19io.stackrox.proto.storageZ\x11./storage;storageb\x06proto3"

var (
	file_storage_delegated_registry_config_proto_rawDescOnce sync.Once
	file_storage_delegated_registry_config_proto_rawDescData []byte
)

func file_storage_delegated_registry_config_proto_rawDescGZIP() []byte {
	file_storage_delegated_registry_config_proto_rawDescOnce.Do(func() {
		file_storage_delegated_registry_config_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_storage_delegated_registry_config_proto_rawDesc), len(file_storage_delegated_registry_config_proto_rawDesc)))
	})
	return file_storage_delegated_registry_config_proto_rawDescData
}

var file_storage_delegated_registry_config_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_storage_delegated_registry_config_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_storage_delegated_registry_config_proto_goTypes = []any{
	(DelegatedRegistryConfig_EnabledFor)(0),           // 0: storage.DelegatedRegistryConfig.EnabledFor
	(*DelegatedRegistryConfig)(nil),                   // 1: storage.DelegatedRegistryConfig
	(*DelegatedRegistryConfig_DelegatedRegistry)(nil), // 2: storage.DelegatedRegistryConfig.DelegatedRegistry
}
var file_storage_delegated_registry_config_proto_depIdxs = []int32{
	0, // 0: storage.DelegatedRegistryConfig.enabled_for:type_name -> storage.DelegatedRegistryConfig.EnabledFor
	2, // 1: storage.DelegatedRegistryConfig.registries:type_name -> storage.DelegatedRegistryConfig.DelegatedRegistry
	2, // [2:2] is the sub-list for method output_type
	2, // [2:2] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_storage_delegated_registry_config_proto_init() }
func file_storage_delegated_registry_config_proto_init() {
	if File_storage_delegated_registry_config_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_storage_delegated_registry_config_proto_rawDesc), len(file_storage_delegated_registry_config_proto_rawDesc)),
			NumEnums:      1,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_storage_delegated_registry_config_proto_goTypes,
		DependencyIndexes: file_storage_delegated_registry_config_proto_depIdxs,
		EnumInfos:         file_storage_delegated_registry_config_proto_enumTypes,
		MessageInfos:      file_storage_delegated_registry_config_proto_msgTypes,
	}.Build()
	File_storage_delegated_registry_config_proto = out.File
	file_storage_delegated_registry_config_proto_goTypes = nil
	file_storage_delegated_registry_config_proto_depIdxs = nil
}
