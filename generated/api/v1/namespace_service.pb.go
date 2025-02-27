// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.5
// 	protoc        v4.25.3
// source: api/v1/namespace_service.proto

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

type Namespace struct {
	state              protoimpl.MessageState     `protogen:"open.v1"`
	Metadata           *storage.NamespaceMetadata `protobuf:"bytes,1,opt,name=metadata,proto3" json:"metadata,omitempty"`
	NumDeployments     int32                      `protobuf:"varint,2,opt,name=num_deployments,json=numDeployments,proto3" json:"num_deployments,omitempty"`
	NumSecrets         int32                      `protobuf:"varint,3,opt,name=num_secrets,json=numSecrets,proto3" json:"num_secrets,omitempty"`
	NumNetworkPolicies int32                      `protobuf:"varint,4,opt,name=num_network_policies,json=numNetworkPolicies,proto3" json:"num_network_policies,omitempty"`
	unknownFields      protoimpl.UnknownFields
	sizeCache          protoimpl.SizeCache
}

func (x *Namespace) Reset() {
	*x = Namespace{}
	mi := &file_api_v1_namespace_service_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Namespace) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Namespace) ProtoMessage() {}

func (x *Namespace) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_namespace_service_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Namespace.ProtoReflect.Descriptor instead.
func (*Namespace) Descriptor() ([]byte, []int) {
	return file_api_v1_namespace_service_proto_rawDescGZIP(), []int{0}
}

func (x *Namespace) GetMetadata() *storage.NamespaceMetadata {
	if x != nil {
		return x.Metadata
	}
	return nil
}

func (x *Namespace) GetNumDeployments() int32 {
	if x != nil {
		return x.NumDeployments
	}
	return 0
}

func (x *Namespace) GetNumSecrets() int32 {
	if x != nil {
		return x.NumSecrets
	}
	return 0
}

func (x *Namespace) GetNumNetworkPolicies() int32 {
	if x != nil {
		return x.NumNetworkPolicies
	}
	return 0
}

type GetNamespacesResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Namespaces    []*Namespace           `protobuf:"bytes,1,rep,name=namespaces,proto3" json:"namespaces,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetNamespacesResponse) Reset() {
	*x = GetNamespacesResponse{}
	mi := &file_api_v1_namespace_service_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetNamespacesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetNamespacesResponse) ProtoMessage() {}

func (x *GetNamespacesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_namespace_service_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetNamespacesResponse.ProtoReflect.Descriptor instead.
func (*GetNamespacesResponse) Descriptor() ([]byte, []int) {
	return file_api_v1_namespace_service_proto_rawDescGZIP(), []int{1}
}

func (x *GetNamespacesResponse) GetNamespaces() []*Namespace {
	if x != nil {
		return x.Namespaces
	}
	return nil
}

type GetNamespaceRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Query         *RawQuery              `protobuf:"bytes,1,opt,name=query,proto3" json:"query,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetNamespaceRequest) Reset() {
	*x = GetNamespaceRequest{}
	mi := &file_api_v1_namespace_service_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetNamespaceRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetNamespaceRequest) ProtoMessage() {}

func (x *GetNamespaceRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_namespace_service_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetNamespaceRequest.ProtoReflect.Descriptor instead.
func (*GetNamespaceRequest) Descriptor() ([]byte, []int) {
	return file_api_v1_namespace_service_proto_rawDescGZIP(), []int{2}
}

func (x *GetNamespaceRequest) GetQuery() *RawQuery {
	if x != nil {
		return x.Query
	}
	return nil
}

var File_api_v1_namespace_service_proto protoreflect.FileDescriptor

var file_api_v1_namespace_service_proto_rawDesc = string([]byte{
	0x0a, 0x1e, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x31, 0x2f, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61,
	0x63, 0x65, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x12, 0x02, 0x76, 0x31, 0x1a, 0x13, 0x61, 0x70, 0x69, 0x2f, 0x76, 0x31, 0x2f, 0x63, 0x6f, 0x6d,
	0x6d, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x61, 0x70, 0x69, 0x2f, 0x76,
	0x31, 0x2f, 0x73, 0x65, 0x61, 0x72, 0x63, 0x68, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x61,
	0x70, 0x69, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x20, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65, 0x2f, 0x6e, 0x61,
	0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x5f, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xbf, 0x01, 0x0a, 0x09, 0x4e, 0x61, 0x6d, 0x65, 0x73,
	0x70, 0x61, 0x63, 0x65, 0x12, 0x36, 0x0a, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x73, 0x74, 0x6f, 0x72, 0x61, 0x67, 0x65,
	0x2e, 0x4e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x4d, 0x65, 0x74, 0x61, 0x64, 0x61,
	0x74, 0x61, 0x52, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x12, 0x27, 0x0a, 0x0f,
	0x6e, 0x75, 0x6d, 0x5f, 0x64, 0x65, 0x70, 0x6c, 0x6f, 0x79, 0x6d, 0x65, 0x6e, 0x74, 0x73, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0e, 0x6e, 0x75, 0x6d, 0x44, 0x65, 0x70, 0x6c, 0x6f, 0x79,
	0x6d, 0x65, 0x6e, 0x74, 0x73, 0x12, 0x1f, 0x0a, 0x0b, 0x6e, 0x75, 0x6d, 0x5f, 0x73, 0x65, 0x63,
	0x72, 0x65, 0x74, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0a, 0x6e, 0x75, 0x6d, 0x53,
	0x65, 0x63, 0x72, 0x65, 0x74, 0x73, 0x12, 0x30, 0x0a, 0x14, 0x6e, 0x75, 0x6d, 0x5f, 0x6e, 0x65,
	0x74, 0x77, 0x6f, 0x72, 0x6b, 0x5f, 0x70, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x05, 0x52, 0x12, 0x6e, 0x75, 0x6d, 0x4e, 0x65, 0x74, 0x77, 0x6f, 0x72, 0x6b,
	0x50, 0x6f, 0x6c, 0x69, 0x63, 0x69, 0x65, 0x73, 0x22, 0x46, 0x0a, 0x15, 0x47, 0x65, 0x74, 0x4e,
	0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x12, 0x2d, 0x0a, 0x0a, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x73, 0x18,
	0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0d, 0x2e, 0x76, 0x31, 0x2e, 0x4e, 0x61, 0x6d, 0x65, 0x73,
	0x70, 0x61, 0x63, 0x65, 0x52, 0x0a, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x73,
	0x22, 0x39, 0x0a, 0x13, 0x47, 0x65, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x22, 0x0a, 0x05, 0x71, 0x75, 0x65, 0x72, 0x79,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0c, 0x2e, 0x76, 0x31, 0x2e, 0x52, 0x61, 0x77, 0x51,
	0x75, 0x65, 0x72, 0x79, 0x52, 0x05, 0x71, 0x75, 0x65, 0x72, 0x79, 0x32, 0xbd, 0x01, 0x0a, 0x10,
	0x4e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65,
	0x12, 0x5b, 0x0a, 0x0d, 0x47, 0x65, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65,
	0x73, 0x12, 0x17, 0x2e, 0x76, 0x31, 0x2e, 0x47, 0x65, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x73, 0x70,
	0x61, 0x63, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x19, 0x2e, 0x76, 0x31, 0x2e,
	0x47, 0x65, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x73, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x16, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x10, 0x12, 0x0e, 0x2f,
	0x76, 0x31, 0x2f, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x73, 0x12, 0x4c, 0x0a,
	0x0c, 0x47, 0x65, 0x74, 0x4e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x12, 0x10, 0x2e,
	0x76, 0x31, 0x2e, 0x52, 0x65, 0x73, 0x6f, 0x75, 0x72, 0x63, 0x65, 0x42, 0x79, 0x49, 0x44, 0x1a,
	0x0d, 0x2e, 0x76, 0x31, 0x2e, 0x4e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x22, 0x1b,
	0x82, 0xd3, 0xe4, 0x93, 0x02, 0x15, 0x12, 0x13, 0x2f, 0x76, 0x31, 0x2f, 0x6e, 0x61, 0x6d, 0x65,
	0x73, 0x70, 0x61, 0x63, 0x65, 0x73, 0x2f, 0x7b, 0x69, 0x64, 0x7d, 0x42, 0x27, 0x0a, 0x18, 0x69,
	0x6f, 0x2e, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x72, 0x6f, 0x78, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x2e, 0x61, 0x70, 0x69, 0x2e, 0x76, 0x31, 0x5a, 0x0b, 0x2e, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x76,
	0x31, 0x3b, 0x76, 0x31, 0x58, 0x02, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
})

var (
	file_api_v1_namespace_service_proto_rawDescOnce sync.Once
	file_api_v1_namespace_service_proto_rawDescData []byte
)

func file_api_v1_namespace_service_proto_rawDescGZIP() []byte {
	file_api_v1_namespace_service_proto_rawDescOnce.Do(func() {
		file_api_v1_namespace_service_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_api_v1_namespace_service_proto_rawDesc), len(file_api_v1_namespace_service_proto_rawDesc)))
	})
	return file_api_v1_namespace_service_proto_rawDescData
}

var file_api_v1_namespace_service_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_api_v1_namespace_service_proto_goTypes = []any{
	(*Namespace)(nil),                 // 0: v1.Namespace
	(*GetNamespacesResponse)(nil),     // 1: v1.GetNamespacesResponse
	(*GetNamespaceRequest)(nil),       // 2: v1.GetNamespaceRequest
	(*storage.NamespaceMetadata)(nil), // 3: storage.NamespaceMetadata
	(*RawQuery)(nil),                  // 4: v1.RawQuery
	(*ResourceByID)(nil),              // 5: v1.ResourceByID
}
var file_api_v1_namespace_service_proto_depIdxs = []int32{
	3, // 0: v1.Namespace.metadata:type_name -> storage.NamespaceMetadata
	0, // 1: v1.GetNamespacesResponse.namespaces:type_name -> v1.Namespace
	4, // 2: v1.GetNamespaceRequest.query:type_name -> v1.RawQuery
	2, // 3: v1.NamespaceService.GetNamespaces:input_type -> v1.GetNamespaceRequest
	5, // 4: v1.NamespaceService.GetNamespace:input_type -> v1.ResourceByID
	1, // 5: v1.NamespaceService.GetNamespaces:output_type -> v1.GetNamespacesResponse
	0, // 6: v1.NamespaceService.GetNamespace:output_type -> v1.Namespace
	5, // [5:7] is the sub-list for method output_type
	3, // [3:5] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_api_v1_namespace_service_proto_init() }
func file_api_v1_namespace_service_proto_init() {
	if File_api_v1_namespace_service_proto != nil {
		return
	}
	file_api_v1_common_proto_init()
	file_api_v1_search_service_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_api_v1_namespace_service_proto_rawDesc), len(file_api_v1_namespace_service_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_api_v1_namespace_service_proto_goTypes,
		DependencyIndexes: file_api_v1_namespace_service_proto_depIdxs,
		MessageInfos:      file_api_v1_namespace_service_proto_msgTypes,
	}.Build()
	File_api_v1_namespace_service_proto = out.File
	file_api_v1_namespace_service_proto_goTypes = nil
	file_api_v1_namespace_service_proto_depIdxs = nil
}
