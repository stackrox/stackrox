// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.5
// 	protoc        v4.25.3
// source: api/v1/mitre_service.proto

package v1

import (
	storage "github.com/stackrox/rox/generated/storage"
	_ "google.golang.org/genproto/googleapis/api/annotations"
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

type ListMitreAttackVectorsResponse struct {
	state              protoimpl.MessageState       `protogen:"open.v1"`
	MitreAttackVectors []*storage.MitreAttackVector `protobuf:"bytes,1,rep,name=mitre_attack_vectors,json=mitreAttackVectors,proto3" json:"mitre_attack_vectors,omitempty"`
	unknownFields      protoimpl.UnknownFields
	sizeCache          protoimpl.SizeCache
}

func (x *ListMitreAttackVectorsResponse) Reset() {
	*x = ListMitreAttackVectorsResponse{}
	mi := &file_api_v1_mitre_service_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListMitreAttackVectorsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListMitreAttackVectorsResponse) ProtoMessage() {}

func (x *ListMitreAttackVectorsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_mitre_service_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListMitreAttackVectorsResponse.ProtoReflect.Descriptor instead.
func (*ListMitreAttackVectorsResponse) Descriptor() ([]byte, []int) {
	return file_api_v1_mitre_service_proto_rawDescGZIP(), []int{0}
}

func (x *ListMitreAttackVectorsResponse) GetMitreAttackVectors() []*storage.MitreAttackVector {
	if x != nil {
		return x.MitreAttackVectors
	}
	return nil
}

type GetMitreVectorResponse struct {
	state             protoimpl.MessageState     `protogen:"open.v1"`
	MitreAttackVector *storage.MitreAttackVector `protobuf:"bytes,1,opt,name=mitre_attack_vector,json=mitreAttackVector,proto3" json:"mitre_attack_vector,omitempty"`
	unknownFields     protoimpl.UnknownFields
	sizeCache         protoimpl.SizeCache
}

func (x *GetMitreVectorResponse) Reset() {
	*x = GetMitreVectorResponse{}
	mi := &file_api_v1_mitre_service_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetMitreVectorResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetMitreVectorResponse) ProtoMessage() {}

func (x *GetMitreVectorResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_mitre_service_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetMitreVectorResponse.ProtoReflect.Descriptor instead.
func (*GetMitreVectorResponse) Descriptor() ([]byte, []int) {
	return file_api_v1_mitre_service_proto_rawDescGZIP(), []int{1}
}

func (x *GetMitreVectorResponse) GetMitreAttackVector() *storage.MitreAttackVector {
	if x != nil {
		return x.MitreAttackVector
	}
	return nil
}

var File_api_v1_mitre_service_proto protoreflect.FileDescriptor

var file_api_v1_mitre_service_proto_rawDesc = string([]byte{
	0x0a, 0x1a, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x31, 0x2f, 0x6d, 0x69, 0x74, 0x72, 0x65, 0x5f, 0x73,
	0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x02, 0x76, 0x31,
	0x1a, 0x13, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x31, 0x2f, 0x63, 0x6f, 0x6d, 0x6d, 0x6f, 0x6e, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x12, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x31, 0x2f, 0x65, 0x6d,
	0x70, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e,
	0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x13, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65,
	0x2f, 0x6d, 0x69, 0x74, 0x72, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x6e, 0x0a, 0x1e,
	0x4c, 0x69, 0x73, 0x74, 0x4d, 0x69, 0x74, 0x72, 0x65, 0x41, 0x74, 0x74, 0x61, 0x63, 0x6b, 0x56,
	0x65, 0x63, 0x74, 0x6f, 0x72, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x4c,
	0x0a, 0x14, 0x6d, 0x69, 0x74, 0x72, 0x65, 0x5f, 0x61, 0x74, 0x74, 0x61, 0x63, 0x6b, 0x5f, 0x76,
	0x65, 0x63, 0x74, 0x6f, 0x72, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x73,
	0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x2e, 0x4d, 0x69, 0x74, 0x72, 0x65, 0x41, 0x74, 0x74, 0x61,
	0x63, 0x6b, 0x56, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x52, 0x12, 0x6d, 0x69, 0x74, 0x72, 0x65, 0x41,
	0x74, 0x74, 0x61, 0x63, 0x6b, 0x56, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x73, 0x22, 0x64, 0x0a, 0x16,
	0x47, 0x65, 0x74, 0x4d, 0x69, 0x74, 0x72, 0x65, 0x56, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x4a, 0x0a, 0x13, 0x6d, 0x69, 0x74, 0x72, 0x65, 0x5f,
	0x61, 0x74, 0x74, 0x61, 0x63, 0x6b, 0x5f, 0x76, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x2e, 0x4d, 0x69,
	0x74, 0x72, 0x65, 0x41, 0x74, 0x74, 0x61, 0x63, 0x6b, 0x56, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x52,
	0x11, 0x6d, 0x69, 0x74, 0x72, 0x65, 0x41, 0x74, 0x74, 0x61, 0x63, 0x6b, 0x56, 0x65, 0x63, 0x74,
	0x6f, 0x72, 0x32, 0xe8, 0x01, 0x0a, 0x12, 0x4d, 0x69, 0x74, 0x72, 0x65, 0x41, 0x74, 0x74, 0x61,
	0x63, 0x6b, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x67, 0x0a, 0x16, 0x4c, 0x69, 0x73,
	0x74, 0x4d, 0x69, 0x74, 0x72, 0x65, 0x41, 0x74, 0x74, 0x61, 0x63, 0x6b, 0x56, 0x65, 0x63, 0x74,
	0x6f, 0x72, 0x73, 0x12, 0x09, 0x2e, 0x76, 0x31, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x22,
	0x2e, 0x76, 0x31, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x4d, 0x69, 0x74, 0x72, 0x65, 0x41, 0x74, 0x74,
	0x61, 0x63, 0x6b, 0x56, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x22, 0x1e, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x18, 0x12, 0x16, 0x2f, 0x76, 0x31, 0x2f,
	0x6d, 0x69, 0x74, 0x72, 0x65, 0x61, 0x74, 0x74, 0x61, 0x63, 0x6b, 0x76, 0x65, 0x63, 0x74, 0x6f,
	0x72, 0x73, 0x12, 0x69, 0x0a, 0x14, 0x47, 0x65, 0x74, 0x4d, 0x69, 0x74, 0x72, 0x65, 0x41, 0x74,
	0x74, 0x61, 0x63, 0x6b, 0x56, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x12, 0x10, 0x2e, 0x76, 0x31, 0x2e,
	0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x42, 0x79, 0x49, 0x44, 0x1a, 0x1a, 0x2e, 0x76,
	0x31, 0x2e, 0x47, 0x65, 0x74, 0x4d, 0x69, 0x74, 0x72, 0x65, 0x56, 0x65, 0x63, 0x74, 0x6f, 0x72,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x23, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x1d,
	0x12, 0x1b, 0x2f, 0x76, 0x31, 0x2f, 0x6d, 0x69, 0x74, 0x72, 0x65, 0x61, 0x74, 0x74, 0x61, 0x63,
	0x6b, 0x76, 0x65, 0x63, 0x74, 0x6f, 0x72, 0x73, 0x2f, 0x7b, 0x69, 0x64, 0x7d, 0x42, 0x27, 0x0a,
	0x18, 0x69, 0x6f, 0x2e, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x72, 0x6f, 0x78, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x5a, 0x0b, 0x2e, 0x2f, 0x61, 0x70, 0x69,
	0x2f, 0x76, 0x31, 0x3b, 0x76, 0x31, 0x58, 0x02, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
})

var (
	file_api_v1_mitre_service_proto_rawDescOnce sync.Once
	file_api_v1_mitre_service_proto_rawDescData []byte
)

func file_api_v1_mitre_service_proto_rawDescGZIP() []byte {
	file_api_v1_mitre_service_proto_rawDescOnce.Do(func() {
		file_api_v1_mitre_service_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_api_v1_mitre_service_proto_rawDesc), len(file_api_v1_mitre_service_proto_rawDesc)))
	})
	return file_api_v1_mitre_service_proto_rawDescData
}

var file_api_v1_mitre_service_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_api_v1_mitre_service_proto_goTypes = []any{
	(*ListMitreAttackVectorsResponse)(nil), // 0: v1.ListMitreAttackVectorsResponse
	(*GetMitreVectorResponse)(nil),         // 1: v1.GetMitreVectorResponse
	(*storage.MitreAttackVector)(nil),      // 2: storage.MitreAttackVector
	(*Empty)(nil),                          // 3: v1.Empty
	(*ResourceByID)(nil),                   // 4: v1.ResourceByID
}
var file_api_v1_mitre_service_proto_depIdxs = []int32{
	2, // 0: v1.ListMitreAttackVectorsResponse.mitre_attack_vectors:type_name -> storage.MitreAttackVector
	2, // 1: v1.GetMitreVectorResponse.mitre_attack_vector:type_name -> storage.MitreAttackVector
	3, // 2: v1.MitreAttackService.ListMitreAttackVectors:input_type -> v1.Empty
	4, // 3: v1.MitreAttackService.GetMitreAttackVector:input_type -> v1.ResourceByID
	0, // 4: v1.MitreAttackService.ListMitreAttackVectors:output_type -> v1.ListMitreAttackVectorsResponse
	1, // 5: v1.MitreAttackService.GetMitreAttackVector:output_type -> v1.GetMitreVectorResponse
	4, // [4:6] is the sub-list for method output_type
	2, // [2:4] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_api_v1_mitre_service_proto_init() }
func file_api_v1_mitre_service_proto_init() {
	if File_api_v1_mitre_service_proto != nil {
		return
	}
	file_api_v1_common_proto_init()
	file_api_v1_empty_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_api_v1_mitre_service_proto_rawDesc), len(file_api_v1_mitre_service_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_api_v1_mitre_service_proto_goTypes,
		DependencyIndexes: file_api_v1_mitre_service_proto_depIdxs,
		MessageInfos:      file_api_v1_mitre_service_proto_msgTypes,
	}.Build()
	File_api_v1_mitre_service_proto = out.File
	file_api_v1_mitre_service_proto_goTypes = nil
	file_api_v1_mitre_service_proto_depIdxs = nil
}
