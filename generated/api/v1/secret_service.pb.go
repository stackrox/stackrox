// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v4.25.3
// source: api/v1/secret_service.proto

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

// A list of secrets (free of scoped information)
// Next Tag: 2
type SecretList struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Secrets       []*storage.Secret      `protobuf:"bytes,1,rep,name=secrets,proto3" json:"secrets,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SecretList) Reset() {
	*x = SecretList{}
	mi := &file_api_v1_secret_service_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SecretList) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SecretList) ProtoMessage() {}

func (x *SecretList) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_secret_service_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use SecretList.ProtoReflect.Descriptor instead.
func (*SecretList) Descriptor() ([]byte, []int) {
	return file_api_v1_secret_service_proto_rawDescGZIP(), []int{0}
}

func (x *SecretList) GetSecrets() []*storage.Secret {
	if x != nil {
		return x.Secrets
	}
	return nil
}

// A list of secrets with their relationships.
// Next Tag: 2
type ListSecretsResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Secrets       []*storage.ListSecret  `protobuf:"bytes,1,rep,name=secrets,proto3" json:"secrets,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ListSecretsResponse) Reset() {
	*x = ListSecretsResponse{}
	mi := &file_api_v1_secret_service_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ListSecretsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListSecretsResponse) ProtoMessage() {}

func (x *ListSecretsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_secret_service_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListSecretsResponse.ProtoReflect.Descriptor instead.
func (*ListSecretsResponse) Descriptor() ([]byte, []int) {
	return file_api_v1_secret_service_proto_rawDescGZIP(), []int{1}
}

func (x *ListSecretsResponse) GetSecrets() []*storage.ListSecret {
	if x != nil {
		return x.Secrets
	}
	return nil
}

type CountSecretsResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Count         int32                  `protobuf:"varint,1,opt,name=count,proto3" json:"count,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CountSecretsResponse) Reset() {
	*x = CountSecretsResponse{}
	mi := &file_api_v1_secret_service_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CountSecretsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CountSecretsResponse) ProtoMessage() {}

func (x *CountSecretsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_v1_secret_service_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CountSecretsResponse.ProtoReflect.Descriptor instead.
func (*CountSecretsResponse) Descriptor() ([]byte, []int) {
	return file_api_v1_secret_service_proto_rawDescGZIP(), []int{2}
}

func (x *CountSecretsResponse) GetCount() int32 {
	if x != nil {
		return x.Count
	}
	return 0
}

var File_api_v1_secret_service_proto protoreflect.FileDescriptor

const file_api_v1_secret_service_proto_rawDesc = "" +
	"\n" +
	"\x1bapi/v1/secret_service.proto\x12\x02v1\x1a\x13api/v1/common.proto\x1a\x1bapi/v1/search_service.proto\x1a\x1cgoogle/api/annotations.proto\x1a\x14storage/secret.proto\"7\n" +
	"\n" +
	"SecretList\x12)\n" +
	"\asecrets\x18\x01 \x03(\v2\x0f.storage.SecretR\asecrets\"D\n" +
	"\x13ListSecretsResponse\x12-\n" +
	"\asecrets\x18\x01 \x03(\v2\x13.storage.ListSecretR\asecrets\",\n" +
	"\x14CountSecretsResponse\x12\x14\n" +
	"\x05count\x18\x01 \x01(\x05R\x05count2\xf6\x01\n" +
	"\rSecretService\x12H\n" +
	"\tGetSecret\x12\x10.v1.ResourceByID\x1a\x0f.storage.Secret\"\x18\x82\xd3\xe4\x93\x02\x12\x12\x10/v1/secrets/{id}\x12P\n" +
	"\fCountSecrets\x12\f.v1.RawQuery\x1a\x18.v1.CountSecretsResponse\"\x18\x82\xd3\xe4\x93\x02\x12\x12\x10/v1/secretscount\x12I\n" +
	"\vListSecrets\x12\f.v1.RawQuery\x1a\x17.v1.ListSecretsResponse\"\x13\x82\xd3\xe4\x93\x02\r\x12\v/v1/secretsB'\n" +
	"\x18io.stackrox.proto.api.v1Z\v./api/v1;v1X\x02b\x06proto3"

var (
	file_api_v1_secret_service_proto_rawDescOnce sync.Once
	file_api_v1_secret_service_proto_rawDescData []byte
)

func file_api_v1_secret_service_proto_rawDescGZIP() []byte {
	file_api_v1_secret_service_proto_rawDescOnce.Do(func() {
		file_api_v1_secret_service_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_api_v1_secret_service_proto_rawDesc), len(file_api_v1_secret_service_proto_rawDesc)))
	})
	return file_api_v1_secret_service_proto_rawDescData
}

var file_api_v1_secret_service_proto_msgTypes = make([]protoimpl.MessageInfo, 3)
var file_api_v1_secret_service_proto_goTypes = []any{
	(*SecretList)(nil),           // 0: v1.SecretList
	(*ListSecretsResponse)(nil),  // 1: v1.ListSecretsResponse
	(*CountSecretsResponse)(nil), // 2: v1.CountSecretsResponse
	(*storage.Secret)(nil),       // 3: storage.Secret
	(*storage.ListSecret)(nil),   // 4: storage.ListSecret
	(*ResourceByID)(nil),         // 5: v1.ResourceByID
	(*RawQuery)(nil),             // 6: v1.RawQuery
}
var file_api_v1_secret_service_proto_depIdxs = []int32{
	3, // 0: v1.SecretList.secrets:type_name -> storage.Secret
	4, // 1: v1.ListSecretsResponse.secrets:type_name -> storage.ListSecret
	5, // 2: v1.SecretService.GetSecret:input_type -> v1.ResourceByID
	6, // 3: v1.SecretService.CountSecrets:input_type -> v1.RawQuery
	6, // 4: v1.SecretService.ListSecrets:input_type -> v1.RawQuery
	3, // 5: v1.SecretService.GetSecret:output_type -> storage.Secret
	2, // 6: v1.SecretService.CountSecrets:output_type -> v1.CountSecretsResponse
	1, // 7: v1.SecretService.ListSecrets:output_type -> v1.ListSecretsResponse
	5, // [5:8] is the sub-list for method output_type
	2, // [2:5] is the sub-list for method input_type
	2, // [2:2] is the sub-list for extension type_name
	2, // [2:2] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_api_v1_secret_service_proto_init() }
func file_api_v1_secret_service_proto_init() {
	if File_api_v1_secret_service_proto != nil {
		return
	}
	file_api_v1_common_proto_init()
	file_api_v1_search_service_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_api_v1_secret_service_proto_rawDesc), len(file_api_v1_secret_service_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   3,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_api_v1_secret_service_proto_goTypes,
		DependencyIndexes: file_api_v1_secret_service_proto_depIdxs,
		MessageInfos:      file_api_v1_secret_service_proto_msgTypes,
	}.Build()
	File_api_v1_secret_service_proto = out.File
	file_api_v1_secret_service_proto_goTypes = nil
	file_api_v1_secret_service_proto_depIdxs = nil
}
