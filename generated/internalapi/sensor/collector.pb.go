// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v4.25.3
// source: internalapi/sensor/collector.proto

package sensor

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
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

type ExternalIpsEnabled int32

const (
	ExternalIpsEnabled_DISABLED ExternalIpsEnabled = 0
	ExternalIpsEnabled_ENABLED  ExternalIpsEnabled = 1
)

// Enum value maps for ExternalIpsEnabled.
var (
	ExternalIpsEnabled_name = map[int32]string{
		0: "DISABLED",
		1: "ENABLED",
	}
	ExternalIpsEnabled_value = map[string]int32{
		"DISABLED": 0,
		"ENABLED":  1,
	}
)

func (x ExternalIpsEnabled) Enum() *ExternalIpsEnabled {
	p := new(ExternalIpsEnabled)
	*p = x
	return p
}

func (x ExternalIpsEnabled) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ExternalIpsEnabled) Descriptor() protoreflect.EnumDescriptor {
	return file_internalapi_sensor_collector_proto_enumTypes[0].Descriptor()
}

func (ExternalIpsEnabled) Type() protoreflect.EnumType {
	return &file_internalapi_sensor_collector_proto_enumTypes[0]
}

func (x ExternalIpsEnabled) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ExternalIpsEnabled.Descriptor instead.
func (ExternalIpsEnabled) EnumDescriptor() ([]byte, []int) {
	return file_internalapi_sensor_collector_proto_rawDescGZIP(), []int{0}
}

type ExternalIpsDirection int32

const (
	ExternalIpsDirection_UNSPECIFIED ExternalIpsDirection = 0
	ExternalIpsDirection_BOTH        ExternalIpsDirection = 1
	ExternalIpsDirection_INGRESS     ExternalIpsDirection = 2
	ExternalIpsDirection_EGRESS      ExternalIpsDirection = 3
)

// Enum value maps for ExternalIpsDirection.
var (
	ExternalIpsDirection_name = map[int32]string{
		0: "UNSPECIFIED",
		1: "BOTH",
		2: "INGRESS",
		3: "EGRESS",
	}
	ExternalIpsDirection_value = map[string]int32{
		"UNSPECIFIED": 0,
		"BOTH":        1,
		"INGRESS":     2,
		"EGRESS":      3,
	}
)

func (x ExternalIpsDirection) Enum() *ExternalIpsDirection {
	p := new(ExternalIpsDirection)
	*p = x
	return p
}

func (x ExternalIpsDirection) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ExternalIpsDirection) Descriptor() protoreflect.EnumDescriptor {
	return file_internalapi_sensor_collector_proto_enumTypes[1].Descriptor()
}

func (ExternalIpsDirection) Type() protoreflect.EnumType {
	return &file_internalapi_sensor_collector_proto_enumTypes[1]
}

func (x ExternalIpsDirection) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use ExternalIpsDirection.Descriptor instead.
func (ExternalIpsDirection) EnumDescriptor() ([]byte, []int) {
	return file_internalapi_sensor_collector_proto_rawDescGZIP(), []int{1}
}

// A request message sent by collector to register with Sensor. Typically the first message in any streams.
type CollectorRegisterRequest struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// The hostname on which collector is running.
	Hostname string `protobuf:"bytes,1,opt,name=hostname,proto3" json:"hostname,omitempty"`
	// A unique identifier for an instance of collector.
	InstanceId    string `protobuf:"bytes,2,opt,name=instance_id,json=instanceId,proto3" json:"instance_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CollectorRegisterRequest) Reset() {
	*x = CollectorRegisterRequest{}
	mi := &file_internalapi_sensor_collector_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CollectorRegisterRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CollectorRegisterRequest) ProtoMessage() {}

func (x *CollectorRegisterRequest) ProtoReflect() protoreflect.Message {
	mi := &file_internalapi_sensor_collector_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CollectorRegisterRequest.ProtoReflect.Descriptor instead.
func (*CollectorRegisterRequest) Descriptor() ([]byte, []int) {
	return file_internalapi_sensor_collector_proto_rawDescGZIP(), []int{0}
}

func (x *CollectorRegisterRequest) GetHostname() string {
	if x != nil {
		return x.Hostname
	}
	return ""
}

func (x *CollectorRegisterRequest) GetInstanceId() string {
	if x != nil {
		return x.InstanceId
	}
	return ""
}

// CollectorConfig controls which type of data is reported by collector
// and how it is processed by collector. These configurations are used
// to fine-tune collector's performance on large scale clusters.
// At this time it only controls if external IPs are aggregated at the
// cluster level and the maximum number of open connections reported
// for each container per minute.
type CollectorConfig struct {
	state         protoimpl.MessageState      `protogen:"open.v1"`
	Networking    *CollectorConfig_Networking `protobuf:"bytes,1,opt,name=networking,proto3" json:"networking,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CollectorConfig) Reset() {
	*x = CollectorConfig{}
	mi := &file_internalapi_sensor_collector_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CollectorConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CollectorConfig) ProtoMessage() {}

func (x *CollectorConfig) ProtoReflect() protoreflect.Message {
	mi := &file_internalapi_sensor_collector_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CollectorConfig.ProtoReflect.Descriptor instead.
func (*CollectorConfig) Descriptor() ([]byte, []int) {
	return file_internalapi_sensor_collector_proto_rawDescGZIP(), []int{1}
}

func (x *CollectorConfig) GetNetworking() *CollectorConfig_Networking {
	if x != nil {
		return x.Networking
	}
	return nil
}

type ProcessSignal struct {
	state protoimpl.MessageState `protogen:"open.v1"`
	// A unique UUID for identifying the message
	// We have this here instead of at the top level
	// because we want to have each message to be
	// self contained.
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// ID of container associated with this process
	ContainerId string `protobuf:"bytes,2,opt,name=container_id,json=containerId,proto3" json:"container_id,omitempty"`
	// Process creation time
	CreationTime *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=creation_time,json=creationTime,proto3" json:"creation_time,omitempty"`
	// Process name
	Name string `protobuf:"bytes,4,opt,name=name,proto3" json:"name,omitempty"`
	// Process arguments
	Args string `protobuf:"bytes,5,opt,name=args,proto3" json:"args,omitempty"`
	// Process executable file path
	ExecFilePath string `protobuf:"bytes,6,opt,name=exec_file_path,json=execFilePath,proto3" json:"exec_file_path,omitempty"`
	// Host process ID
	Pid uint32 `protobuf:"varint,7,opt,name=pid,proto3" json:"pid,omitempty"`
	// Real user ID
	Uid uint32 `protobuf:"varint,8,opt,name=uid,proto3" json:"uid,omitempty"`
	// Real group ID
	Gid uint32 `protobuf:"varint,9,opt,name=gid,proto3" json:"gid,omitempty"`
	// Signal origin
	Scraped bool `protobuf:"varint,10,opt,name=scraped,proto3" json:"scraped,omitempty"`
	// Process LineageInfo
	LineageInfo   []*ProcessSignal_LineageInfo `protobuf:"bytes,11,rep,name=lineage_info,json=lineageInfo,proto3" json:"lineage_info,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ProcessSignal) Reset() {
	*x = ProcessSignal{}
	mi := &file_internalapi_sensor_collector_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ProcessSignal) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProcessSignal) ProtoMessage() {}

func (x *ProcessSignal) ProtoReflect() protoreflect.Message {
	mi := &file_internalapi_sensor_collector_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProcessSignal.ProtoReflect.Descriptor instead.
func (*ProcessSignal) Descriptor() ([]byte, []int) {
	return file_internalapi_sensor_collector_proto_rawDescGZIP(), []int{2}
}

func (x *ProcessSignal) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *ProcessSignal) GetContainerId() string {
	if x != nil {
		return x.ContainerId
	}
	return ""
}

func (x *ProcessSignal) GetCreationTime() *timestamppb.Timestamp {
	if x != nil {
		return x.CreationTime
	}
	return nil
}

func (x *ProcessSignal) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *ProcessSignal) GetArgs() string {
	if x != nil {
		return x.Args
	}
	return ""
}

func (x *ProcessSignal) GetExecFilePath() string {
	if x != nil {
		return x.ExecFilePath
	}
	return ""
}

func (x *ProcessSignal) GetPid() uint32 {
	if x != nil {
		return x.Pid
	}
	return 0
}

func (x *ProcessSignal) GetUid() uint32 {
	if x != nil {
		return x.Uid
	}
	return 0
}

func (x *ProcessSignal) GetGid() uint32 {
	if x != nil {
		return x.Gid
	}
	return 0
}

func (x *ProcessSignal) GetScraped() bool {
	if x != nil {
		return x.Scraped
	}
	return false
}

func (x *ProcessSignal) GetLineageInfo() []*ProcessSignal_LineageInfo {
	if x != nil {
		return x.LineageInfo
	}
	return nil
}

type CollectorConfig_ExternalIPs struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Enabled       ExternalIpsEnabled     `protobuf:"varint,1,opt,name=enabled,proto3,enum=sensor.ExternalIpsEnabled" json:"enabled,omitempty"`
	Direction     ExternalIpsDirection   `protobuf:"varint,2,opt,name=direction,proto3,enum=sensor.ExternalIpsDirection" json:"direction,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *CollectorConfig_ExternalIPs) Reset() {
	*x = CollectorConfig_ExternalIPs{}
	mi := &file_internalapi_sensor_collector_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CollectorConfig_ExternalIPs) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CollectorConfig_ExternalIPs) ProtoMessage() {}

func (x *CollectorConfig_ExternalIPs) ProtoReflect() protoreflect.Message {
	mi := &file_internalapi_sensor_collector_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CollectorConfig_ExternalIPs.ProtoReflect.Descriptor instead.
func (*CollectorConfig_ExternalIPs) Descriptor() ([]byte, []int) {
	return file_internalapi_sensor_collector_proto_rawDescGZIP(), []int{1, 0}
}

func (x *CollectorConfig_ExternalIPs) GetEnabled() ExternalIpsEnabled {
	if x != nil {
		return x.Enabled
	}
	return ExternalIpsEnabled_DISABLED
}

func (x *CollectorConfig_ExternalIPs) GetDirection() ExternalIpsDirection {
	if x != nil {
		return x.Direction
	}
	return ExternalIpsDirection_UNSPECIFIED
}

type CollectorConfig_Networking struct {
	state                   protoimpl.MessageState       `protogen:"open.v1"`
	ExternalIps             *CollectorConfig_ExternalIPs `protobuf:"bytes,1,opt,name=external_ips,json=externalIps,proto3" json:"external_ips,omitempty"`
	MaxConnectionsPerMinute int64                        `protobuf:"varint,2,opt,name=max_connections_per_minute,json=maxConnectionsPerMinute,proto3" json:"max_connections_per_minute,omitempty"`
	unknownFields           protoimpl.UnknownFields
	sizeCache               protoimpl.SizeCache
}

func (x *CollectorConfig_Networking) Reset() {
	*x = CollectorConfig_Networking{}
	mi := &file_internalapi_sensor_collector_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *CollectorConfig_Networking) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CollectorConfig_Networking) ProtoMessage() {}

func (x *CollectorConfig_Networking) ProtoReflect() protoreflect.Message {
	mi := &file_internalapi_sensor_collector_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CollectorConfig_Networking.ProtoReflect.Descriptor instead.
func (*CollectorConfig_Networking) Descriptor() ([]byte, []int) {
	return file_internalapi_sensor_collector_proto_rawDescGZIP(), []int{1, 1}
}

func (x *CollectorConfig_Networking) GetExternalIps() *CollectorConfig_ExternalIPs {
	if x != nil {
		return x.ExternalIps
	}
	return nil
}

func (x *CollectorConfig_Networking) GetMaxConnectionsPerMinute() int64 {
	if x != nil {
		return x.MaxConnectionsPerMinute
	}
	return 0
}

type ProcessSignal_LineageInfo struct {
	state              protoimpl.MessageState `protogen:"open.v1"`
	ParentUid          uint32                 `protobuf:"varint,1,opt,name=parent_uid,json=parentUid,proto3" json:"parent_uid,omitempty"`
	ParentExecFilePath string                 `protobuf:"bytes,2,opt,name=parent_exec_file_path,json=parentExecFilePath,proto3" json:"parent_exec_file_path,omitempty"`
	unknownFields      protoimpl.UnknownFields
	sizeCache          protoimpl.SizeCache
}

func (x *ProcessSignal_LineageInfo) Reset() {
	*x = ProcessSignal_LineageInfo{}
	mi := &file_internalapi_sensor_collector_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ProcessSignal_LineageInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProcessSignal_LineageInfo) ProtoMessage() {}

func (x *ProcessSignal_LineageInfo) ProtoReflect() protoreflect.Message {
	mi := &file_internalapi_sensor_collector_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ProcessSignal_LineageInfo.ProtoReflect.Descriptor instead.
func (*ProcessSignal_LineageInfo) Descriptor() ([]byte, []int) {
	return file_internalapi_sensor_collector_proto_rawDescGZIP(), []int{2, 0}
}

func (x *ProcessSignal_LineageInfo) GetParentUid() uint32 {
	if x != nil {
		return x.ParentUid
	}
	return 0
}

func (x *ProcessSignal_LineageInfo) GetParentExecFilePath() string {
	if x != nil {
		return x.ParentExecFilePath
	}
	return ""
}

var File_internalapi_sensor_collector_proto protoreflect.FileDescriptor

const file_internalapi_sensor_collector_proto_rawDesc = "" +
	"\n" +
	"\"internalapi/sensor/collector.proto\x12\x06sensor\x1a\x1fgoogle/protobuf/timestamp.proto\"W\n" +
	"\x18CollectorRegisterRequest\x12\x1a\n" +
	"\bhostname\x18\x01 \x01(\tR\bhostname\x12\x1f\n" +
	"\vinstance_id\x18\x02 \x01(\tR\n" +
	"instanceId\"\xea\x02\n" +
	"\x0fCollectorConfig\x12B\n" +
	"\n" +
	"networking\x18\x01 \x01(\v2\".sensor.CollectorConfig.NetworkingR\n" +
	"networking\x1a\x7f\n" +
	"\vExternalIPs\x124\n" +
	"\aenabled\x18\x01 \x01(\x0e2\x1a.sensor.ExternalIpsEnabledR\aenabled\x12:\n" +
	"\tdirection\x18\x02 \x01(\x0e2\x1c.sensor.ExternalIpsDirectionR\tdirection\x1a\x91\x01\n" +
	"\n" +
	"Networking\x12F\n" +
	"\fexternal_ips\x18\x01 \x01(\v2#.sensor.CollectorConfig.ExternalIPsR\vexternalIps\x12;\n" +
	"\x1amax_connections_per_minute\x18\x02 \x01(\x03R\x17maxConnectionsPerMinute\"\xc8\x03\n" +
	"\rProcessSignal\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12!\n" +
	"\fcontainer_id\x18\x02 \x01(\tR\vcontainerId\x12?\n" +
	"\rcreation_time\x18\x03 \x01(\v2\x1a.google.protobuf.TimestampR\fcreationTime\x12\x12\n" +
	"\x04name\x18\x04 \x01(\tR\x04name\x12\x12\n" +
	"\x04args\x18\x05 \x01(\tR\x04args\x12$\n" +
	"\x0eexec_file_path\x18\x06 \x01(\tR\fexecFilePath\x12\x10\n" +
	"\x03pid\x18\a \x01(\rR\x03pid\x12\x10\n" +
	"\x03uid\x18\b \x01(\rR\x03uid\x12\x10\n" +
	"\x03gid\x18\t \x01(\rR\x03gid\x12\x18\n" +
	"\ascraped\x18\n" +
	" \x01(\bR\ascraped\x12D\n" +
	"\flineage_info\x18\v \x03(\v2!.sensor.ProcessSignal.LineageInfoR\vlineageInfo\x1a_\n" +
	"\vLineageInfo\x12\x1d\n" +
	"\n" +
	"parent_uid\x18\x01 \x01(\rR\tparentUid\x121\n" +
	"\x15parent_exec_file_path\x18\x02 \x01(\tR\x12parentExecFilePath*/\n" +
	"\x12ExternalIpsEnabled\x12\f\n" +
	"\bDISABLED\x10\x00\x12\v\n" +
	"\aENABLED\x10\x01*J\n" +
	"\x14ExternalIpsDirection\x12\x0f\n" +
	"\vUNSPECIFIED\x10\x00\x12\b\n" +
	"\x04BOTH\x10\x01\x12\v\n" +
	"\aINGRESS\x10\x02\x12\n" +
	"\n" +
	"\x06EGRESS\x10\x03B Z\x1b./internalapi/sensor;sensor\xf8\x01\x01b\x06proto3"

var (
	file_internalapi_sensor_collector_proto_rawDescOnce sync.Once
	file_internalapi_sensor_collector_proto_rawDescData []byte
)

func file_internalapi_sensor_collector_proto_rawDescGZIP() []byte {
	file_internalapi_sensor_collector_proto_rawDescOnce.Do(func() {
		file_internalapi_sensor_collector_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_internalapi_sensor_collector_proto_rawDesc), len(file_internalapi_sensor_collector_proto_rawDesc)))
	})
	return file_internalapi_sensor_collector_proto_rawDescData
}

var file_internalapi_sensor_collector_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_internalapi_sensor_collector_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_internalapi_sensor_collector_proto_goTypes = []any{
	(ExternalIpsEnabled)(0),             // 0: sensor.ExternalIpsEnabled
	(ExternalIpsDirection)(0),           // 1: sensor.ExternalIpsDirection
	(*CollectorRegisterRequest)(nil),    // 2: sensor.CollectorRegisterRequest
	(*CollectorConfig)(nil),             // 3: sensor.CollectorConfig
	(*ProcessSignal)(nil),               // 4: sensor.ProcessSignal
	(*CollectorConfig_ExternalIPs)(nil), // 5: sensor.CollectorConfig.ExternalIPs
	(*CollectorConfig_Networking)(nil),  // 6: sensor.CollectorConfig.Networking
	(*ProcessSignal_LineageInfo)(nil),   // 7: sensor.ProcessSignal.LineageInfo
	(*timestamppb.Timestamp)(nil),       // 8: google.protobuf.Timestamp
}
var file_internalapi_sensor_collector_proto_depIdxs = []int32{
	6, // 0: sensor.CollectorConfig.networking:type_name -> sensor.CollectorConfig.Networking
	8, // 1: sensor.ProcessSignal.creation_time:type_name -> google.protobuf.Timestamp
	7, // 2: sensor.ProcessSignal.lineage_info:type_name -> sensor.ProcessSignal.LineageInfo
	0, // 3: sensor.CollectorConfig.ExternalIPs.enabled:type_name -> sensor.ExternalIpsEnabled
	1, // 4: sensor.CollectorConfig.ExternalIPs.direction:type_name -> sensor.ExternalIpsDirection
	5, // 5: sensor.CollectorConfig.Networking.external_ips:type_name -> sensor.CollectorConfig.ExternalIPs
	6, // [6:6] is the sub-list for method output_type
	6, // [6:6] is the sub-list for method input_type
	6, // [6:6] is the sub-list for extension type_name
	6, // [6:6] is the sub-list for extension extendee
	0, // [0:6] is the sub-list for field type_name
}

func init() { file_internalapi_sensor_collector_proto_init() }
func file_internalapi_sensor_collector_proto_init() {
	if File_internalapi_sensor_collector_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_internalapi_sensor_collector_proto_rawDesc), len(file_internalapi_sensor_collector_proto_rawDesc)),
			NumEnums:      2,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_internalapi_sensor_collector_proto_goTypes,
		DependencyIndexes: file_internalapi_sensor_collector_proto_depIdxs,
		EnumInfos:         file_internalapi_sensor_collector_proto_enumTypes,
		MessageInfos:      file_internalapi_sensor_collector_proto_msgTypes,
	}.Build()
	File_internalapi_sensor_collector_proto = out.File
	file_internalapi_sensor_collector_proto_goTypes = nil
	file_internalapi_sensor_collector_proto_depIdxs = nil
}
