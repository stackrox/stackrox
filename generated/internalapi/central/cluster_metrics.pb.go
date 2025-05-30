// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        v4.25.3
// source: internalapi/central/cluster_metrics.proto

package central

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

// ClusterMetrics defines a set of metrics, which are collected by Sensor and
// send to Central.
type ClusterMetrics struct {
	state                     protoimpl.MessageState `protogen:"open.v1"`
	NodeCount                 int64                  `protobuf:"varint,1,opt,name=node_count,json=nodeCount,proto3" json:"node_count,omitempty"`                                                  // The number of nodes in the cluster accessible by Sensor.
	CpuCapacity               int64                  `protobuf:"varint,2,opt,name=cpu_capacity,json=cpuCapacity,proto3" json:"cpu_capacity,omitempty"`                                            // The total cpu capacity of all nodes accessible by Sensor.
	ComplianceOperatorVersion string                 `protobuf:"bytes,3,opt,name=compliance_operator_version,json=complianceOperatorVersion,proto3" json:"compliance_operator_version,omitempty"` // Compliance operator version discovered by this Sensor.
	unknownFields             protoimpl.UnknownFields
	sizeCache                 protoimpl.SizeCache
}

func (x *ClusterMetrics) Reset() {
	*x = ClusterMetrics{}
	mi := &file_internalapi_central_cluster_metrics_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ClusterMetrics) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ClusterMetrics) ProtoMessage() {}

func (x *ClusterMetrics) ProtoReflect() protoreflect.Message {
	mi := &file_internalapi_central_cluster_metrics_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ClusterMetrics.ProtoReflect.Descriptor instead.
func (*ClusterMetrics) Descriptor() ([]byte, []int) {
	return file_internalapi_central_cluster_metrics_proto_rawDescGZIP(), []int{0}
}

func (x *ClusterMetrics) GetNodeCount() int64 {
	if x != nil {
		return x.NodeCount
	}
	return 0
}

func (x *ClusterMetrics) GetCpuCapacity() int64 {
	if x != nil {
		return x.CpuCapacity
	}
	return 0
}

func (x *ClusterMetrics) GetComplianceOperatorVersion() string {
	if x != nil {
		return x.ComplianceOperatorVersion
	}
	return ""
}

var File_internalapi_central_cluster_metrics_proto protoreflect.FileDescriptor

const file_internalapi_central_cluster_metrics_proto_rawDesc = "" +
	"\n" +
	")internalapi/central/cluster_metrics.proto\x12\acentral\"\x92\x01\n" +
	"\x0eClusterMetrics\x12\x1d\n" +
	"\n" +
	"node_count\x18\x01 \x01(\x03R\tnodeCount\x12!\n" +
	"\fcpu_capacity\x18\x02 \x01(\x03R\vcpuCapacity\x12>\n" +
	"\x1bcompliance_operator_version\x18\x03 \x01(\tR\x19complianceOperatorVersionB\x1fZ\x1d./internalapi/central;centralb\x06proto3"

var (
	file_internalapi_central_cluster_metrics_proto_rawDescOnce sync.Once
	file_internalapi_central_cluster_metrics_proto_rawDescData []byte
)

func file_internalapi_central_cluster_metrics_proto_rawDescGZIP() []byte {
	file_internalapi_central_cluster_metrics_proto_rawDescOnce.Do(func() {
		file_internalapi_central_cluster_metrics_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_internalapi_central_cluster_metrics_proto_rawDesc), len(file_internalapi_central_cluster_metrics_proto_rawDesc)))
	})
	return file_internalapi_central_cluster_metrics_proto_rawDescData
}

var file_internalapi_central_cluster_metrics_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_internalapi_central_cluster_metrics_proto_goTypes = []any{
	(*ClusterMetrics)(nil), // 0: central.ClusterMetrics
}
var file_internalapi_central_cluster_metrics_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_internalapi_central_cluster_metrics_proto_init() }
func file_internalapi_central_cluster_metrics_proto_init() {
	if File_internalapi_central_cluster_metrics_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_internalapi_central_cluster_metrics_proto_rawDesc), len(file_internalapi_central_cluster_metrics_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_internalapi_central_cluster_metrics_proto_goTypes,
		DependencyIndexes: file_internalapi_central_cluster_metrics_proto_depIdxs,
		MessageInfos:      file_internalapi_central_cluster_metrics_proto_msgTypes,
	}.Build()
	File_internalapi_central_cluster_metrics_proto = out.File
	file_internalapi_central_cluster_metrics_proto_goTypes = nil
	file_internalapi_central_cluster_metrics_proto_depIdxs = nil
}
