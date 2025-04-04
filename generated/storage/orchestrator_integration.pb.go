// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v4.25.3
// source: storage/orchestrator_integration.proto

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

// Next Tag: 5
type OrchestratorIntegration struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	Id    string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Name  string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Type  string                 `protobuf:"bytes,3,opt,name=type,proto3" json:"type,omitempty"`
	// Types that are valid to be assigned to IntegrationConfig:
	//
	//	*OrchestratorIntegration_Clairify
	IntegrationConfig isOrchestratorIntegration_IntegrationConfig `protobuf_oneof:"IntegrationConfig"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *OrchestratorIntegration) Reset() {
	*x = OrchestratorIntegration{}
	mi := &file_storage_orchestrator_integration_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *OrchestratorIntegration) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*OrchestratorIntegration) ProtoMessage() {}

func (x *OrchestratorIntegration) ProtoReflect() protoreflect.Message {
	mi := &file_storage_orchestrator_integration_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use OrchestratorIntegration.ProtoReflect.Descriptor instead.
func (*OrchestratorIntegration) Descriptor() ([]byte, []int) {
	return file_storage_orchestrator_integration_proto_rawDescGZIP(), []int{0}
}

func (x *OrchestratorIntegration) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *OrchestratorIntegration) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *OrchestratorIntegration) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *OrchestratorIntegration) GetIntegrationConfig() isOrchestratorIntegration_IntegrationConfig {
	if x != nil {
		return x.IntegrationConfig
	}
	return nil
}

func (x *OrchestratorIntegration) GetClairify() *ClairifyConfig {
	if x != nil {
		if x, ok := x.IntegrationConfig.(*OrchestratorIntegration_Clairify); ok {
			return x.Clairify
		}
	}
	return nil
}

type isOrchestratorIntegration_IntegrationConfig interface {
	isOrchestratorIntegration_IntegrationConfig()
}

type OrchestratorIntegration_Clairify struct {
	Clairify *ClairifyConfig `protobuf:"bytes,4,opt,name=clairify,proto3,oneof"`
}

func (*OrchestratorIntegration_Clairify) isOrchestratorIntegration_IntegrationConfig() {}

var File_storage_orchestrator_integration_proto protoreflect.FileDescriptor

const file_storage_orchestrator_integration_proto_rawDesc = "" +
	"\n" +
	"&storage/orchestrator_integration.proto\x12\astorage\x1a\x1fstorage/image_integration.proto\"\x9d\x01\n" +
	"\x17OrchestratorIntegration\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12\x12\n" +
	"\x04name\x18\x02 \x01(\tR\x04name\x12\x12\n" +
	"\x04type\x18\x03 \x01(\tR\x04type\x125\n" +
	"\bclairify\x18\x04 \x01(\v2\x17.storage.ClairifyConfigH\x00R\bclairifyB\x13\n" +
	"\x11IntegrationConfigB.\n" +
	"\x19io.stackrox.proto.storageZ\x11./storage;storageb\x06proto3"

var (
	file_storage_orchestrator_integration_proto_rawDescOnce sync.Once
	file_storage_orchestrator_integration_proto_rawDescData []byte
)

func file_storage_orchestrator_integration_proto_rawDescGZIP() []byte {
	file_storage_orchestrator_integration_proto_rawDescOnce.Do(func() {
		file_storage_orchestrator_integration_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_storage_orchestrator_integration_proto_rawDesc), len(file_storage_orchestrator_integration_proto_rawDesc)))
	})
	return file_storage_orchestrator_integration_proto_rawDescData
}

var file_storage_orchestrator_integration_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_storage_orchestrator_integration_proto_goTypes = []any{
	(*OrchestratorIntegration)(nil), // 0: storage.OrchestratorIntegration
	(*ClairifyConfig)(nil),          // 1: storage.ClairifyConfig
}
var file_storage_orchestrator_integration_proto_depIdxs = []int32{
	1, // 0: storage.OrchestratorIntegration.clairify:type_name -> storage.ClairifyConfig
	1, // [1:1] is the sub-list for method output_type
	1, // [1:1] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_storage_orchestrator_integration_proto_init() }
func file_storage_orchestrator_integration_proto_init() {
	if File_storage_orchestrator_integration_proto != nil {
		return
	}
	file_storage_image_integration_proto_init()
	file_storage_orchestrator_integration_proto_msgTypes[0].OneofWrappers = []any{
		(*OrchestratorIntegration_Clairify)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_storage_orchestrator_integration_proto_rawDesc), len(file_storage_orchestrator_integration_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_storage_orchestrator_integration_proto_goTypes,
		DependencyIndexes: file_storage_orchestrator_integration_proto_depIdxs,
		MessageInfos:      file_storage_orchestrator_integration_proto_msgTypes,
	}.Build()
	File_storage_orchestrator_integration_proto = out.File
	file_storage_orchestrator_integration_proto_goTypes = nil
	file_storage_orchestrator_integration_proto_depIdxs = nil
}
