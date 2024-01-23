// source: storage/compliance_operator_v2.proto

package storage

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	types "github.com/gogo/protobuf/types"
	proto "github.com/golang/protobuf/proto"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

// Represents the role of the node within the cluster
type NodeRole int32

const (
	NodeRole_INFRA  NodeRole = 0
	NodeRole_WORKER NodeRole = 1
	NodeRole_MASTER NodeRole = 2
)

var NodeRole_name = map[int32]string{
	0: "INFRA",
	1: "WORKER",
	2: "MASTER",
}

var NodeRole_value = map[string]int32{
	"INFRA":  0,
	"WORKER": 1,
	"MASTER": 2,
}

func (x NodeRole) String() string {
	return proto.EnumName(NodeRole_name, int32(x))
}

func (NodeRole) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_26c2ec1f62102154, []int{0}
}

// Represents the scan type whether a node or platform scan.
type ScanType int32

const (
	ScanType_UNSET_SCAN_TYPE ScanType = 0
	ScanType_NODE_SCAN       ScanType = 1
	ScanType_PLATFORM_SCAN   ScanType = 2
)

var ScanType_name = map[int32]string{
	0: "UNSET_SCAN_TYPE",
	1: "NODE_SCAN",
	2: "PLATFORM_SCAN",
}

var ScanType_value = map[string]int32{
	"UNSET_SCAN_TYPE": 0,
	"NODE_SCAN":       1,
	"PLATFORM_SCAN":   2,
}

func (x ScanType) String() string {
	return proto.EnumName(ScanType_name, int32(x))
}

func (ScanType) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_26c2ec1f62102154, []int{1}
}

// Represents the severity of the rule
type RuleSeverity int32

const (
	RuleSeverity_UNSET_RULE_SEVERITY   RuleSeverity = 0
	RuleSeverity_UNKNOWN_RULE_SEVERITY RuleSeverity = 1
	RuleSeverity_INFO_RULE_SEVERITY    RuleSeverity = 2
	RuleSeverity_LOW_RULE_SEVERITY     RuleSeverity = 3
	RuleSeverity_MEDIUM_RULE_SEVERITY  RuleSeverity = 4
	RuleSeverity_HIGH_RULE_SEVERITY    RuleSeverity = 5
)

var RuleSeverity_name = map[int32]string{
	0: "UNSET_RULE_SEVERITY",
	1: "UNKNOWN_RULE_SEVERITY",
	2: "INFO_RULE_SEVERITY",
	3: "LOW_RULE_SEVERITY",
	4: "MEDIUM_RULE_SEVERITY",
	5: "HIGH_RULE_SEVERITY",
}

var RuleSeverity_value = map[string]int32{
	"UNSET_RULE_SEVERITY":   0,
	"UNKNOWN_RULE_SEVERITY": 1,
	"INFO_RULE_SEVERITY":    2,
	"LOW_RULE_SEVERITY":     3,
	"MEDIUM_RULE_SEVERITY":  4,
	"HIGH_RULE_SEVERITY":    5,
}

func (x RuleSeverity) String() string {
	return proto.EnumName(RuleSeverity_name, int32(x))
}

func (RuleSeverity) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_26c2ec1f62102154, []int{2}
}

type ComplianceOperatorCheckResultV2_CheckStatus int32

const (
	ComplianceOperatorCheckResultV2_UNSET          ComplianceOperatorCheckResultV2_CheckStatus = 0
	ComplianceOperatorCheckResultV2_PASS           ComplianceOperatorCheckResultV2_CheckStatus = 1
	ComplianceOperatorCheckResultV2_FAIL           ComplianceOperatorCheckResultV2_CheckStatus = 2
	ComplianceOperatorCheckResultV2_ERROR          ComplianceOperatorCheckResultV2_CheckStatus = 3
	ComplianceOperatorCheckResultV2_INFO           ComplianceOperatorCheckResultV2_CheckStatus = 4
	ComplianceOperatorCheckResultV2_MANUAL         ComplianceOperatorCheckResultV2_CheckStatus = 5
	ComplianceOperatorCheckResultV2_NOT_APPLICABLE ComplianceOperatorCheckResultV2_CheckStatus = 6
	ComplianceOperatorCheckResultV2_INCONSISTENT   ComplianceOperatorCheckResultV2_CheckStatus = 7
)

var ComplianceOperatorCheckResultV2_CheckStatus_name = map[int32]string{
	0: "UNSET",
	1: "PASS",
	2: "FAIL",
	3: "ERROR",
	4: "INFO",
	5: "MANUAL",
	6: "NOT_APPLICABLE",
	7: "INCONSISTENT",
}

var ComplianceOperatorCheckResultV2_CheckStatus_value = map[string]int32{
	"UNSET":          0,
	"PASS":           1,
	"FAIL":           2,
	"ERROR":          3,
	"INFO":           4,
	"MANUAL":         5,
	"NOT_APPLICABLE": 6,
	"INCONSISTENT":   7,
}

func (x ComplianceOperatorCheckResultV2_CheckStatus) String() string {
	return proto.EnumName(ComplianceOperatorCheckResultV2_CheckStatus_name, int32(x))
}

func (ComplianceOperatorCheckResultV2_CheckStatus) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_26c2ec1f62102154, []int{6, 0}
}

type ProfileShim struct {
	ProfileId            string   `protobuf:"bytes,1,opt,name=profile_id,json=profileId,proto3" json:"profile_id,omitempty" search:"-" sql:"fk(ComplianceOperatorProfileV2:id),no-fk-constraint"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ProfileShim) Reset()         { *m = ProfileShim{} }
func (m *ProfileShim) String() string { return proto.CompactTextString(m) }
func (*ProfileShim) ProtoMessage()    {}
func (*ProfileShim) Descriptor() ([]byte, []int) {
	return fileDescriptor_26c2ec1f62102154, []int{0}
}
func (m *ProfileShim) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ProfileShim) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ProfileShim.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ProfileShim) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ProfileShim.Merge(m, src)
}
func (m *ProfileShim) XXX_Size() int {
	return m.Size()
}
func (m *ProfileShim) XXX_DiscardUnknown() {
	xxx_messageInfo_ProfileShim.DiscardUnknown(m)
}

var xxx_messageInfo_ProfileShim proto.InternalMessageInfo

func (m *ProfileShim) GetProfileId() string {
	if m != nil {
		return m.ProfileId
	}
	return ""
}

func (m *ProfileShim) MessageClone() proto.Message {
	return m.Clone()
}
func (m *ProfileShim) Clone() *ProfileShim {
	if m == nil {
		return nil
	}
	cloned := new(ProfileShim)
	*cloned = *m

	return cloned
}

// Next Tag: 13
type ComplianceOperatorProfileV2 struct {
	// The primary key is name-profile_version as that is guaranteed unique in the operator and how
	// the profile is referenced in scans and settings
	Id                   string                              `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty" sql:"pk"`
	ProfileId            string                              `protobuf:"bytes,2,opt,name=profile_id,json=profileId,proto3" json:"profile_id,omitempty" search:"Compliance Profile ID,hidden"`
	Name                 string                              `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty" search:"Compliance Profile Name,hidden" sql:"index=category:unique;name:profile_unique_indicator"`
	ProfileVersion       string                              `protobuf:"bytes,4,opt,name=profile_version,json=profileVersion,proto3" json:"profile_version,omitempty" search:"Compliance Profile Version,hidden" sql:"index=category:unique;name:profile_unique_indicator"`
	ProductType          string                              `protobuf:"bytes,5,opt,name=product_type,json=productType,proto3" json:"product_type,omitempty" search:"Compliance Profile Product Type,hidden"`
	Standard             string                              `protobuf:"bytes,6,opt,name=standard,proto3" json:"standard,omitempty" search:"Compliance Standard,hidden"`
	Labels               map[string]string                   `protobuf:"bytes,7,rep,name=labels,proto3" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Annotations          map[string]string                   `protobuf:"bytes,8,rep,name=annotations,proto3" json:"annotations,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Description          string                              `protobuf:"bytes,9,opt,name=description,proto3" json:"description,omitempty"`
	Rules                []*ComplianceOperatorProfileV2_Rule `protobuf:"bytes,10,rep,name=rules,proto3" json:"rules,omitempty"`
	Product              string                              `protobuf:"bytes,11,opt,name=product,proto3" json:"product,omitempty"`
	Title                string                              `protobuf:"bytes,12,opt,name=title,proto3" json:"title,omitempty"`
	Values               []string                            `protobuf:"bytes,13,rep,name=values,proto3" json:"values,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                            `json:"-"`
	XXX_unrecognized     []byte                              `json:"-"`
	XXX_sizecache        int32                               `json:"-"`
}

func (m *ComplianceOperatorProfileV2) Reset()         { *m = ComplianceOperatorProfileV2{} }
func (m *ComplianceOperatorProfileV2) String() string { return proto.CompactTextString(m) }
func (*ComplianceOperatorProfileV2) ProtoMessage()    {}
func (*ComplianceOperatorProfileV2) Descriptor() ([]byte, []int) {
	return fileDescriptor_26c2ec1f62102154, []int{1}
}
func (m *ComplianceOperatorProfileV2) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ComplianceOperatorProfileV2) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ComplianceOperatorProfileV2.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ComplianceOperatorProfileV2) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ComplianceOperatorProfileV2.Merge(m, src)
}
func (m *ComplianceOperatorProfileV2) XXX_Size() int {
	return m.Size()
}
func (m *ComplianceOperatorProfileV2) XXX_DiscardUnknown() {
	xxx_messageInfo_ComplianceOperatorProfileV2.DiscardUnknown(m)
}

var xxx_messageInfo_ComplianceOperatorProfileV2 proto.InternalMessageInfo

func (m *ComplianceOperatorProfileV2) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *ComplianceOperatorProfileV2) GetProfileId() string {
	if m != nil {
		return m.ProfileId
	}
	return ""
}

func (m *ComplianceOperatorProfileV2) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *ComplianceOperatorProfileV2) GetProfileVersion() string {
	if m != nil {
		return m.ProfileVersion
	}
	return ""
}

func (m *ComplianceOperatorProfileV2) GetProductType() string {
	if m != nil {
		return m.ProductType
	}
	return ""
}

func (m *ComplianceOperatorProfileV2) GetStandard() string {
	if m != nil {
		return m.Standard
	}
	return ""
}

func (m *ComplianceOperatorProfileV2) GetLabels() map[string]string {
	if m != nil {
		return m.Labels
	}
	return nil
}

func (m *ComplianceOperatorProfileV2) GetAnnotations() map[string]string {
	if m != nil {
		return m.Annotations
	}
	return nil
}

func (m *ComplianceOperatorProfileV2) GetDescription() string {
	if m != nil {
		return m.Description
	}
	return ""
}

func (m *ComplianceOperatorProfileV2) GetRules() []*ComplianceOperatorProfileV2_Rule {
	if m != nil {
		return m.Rules
	}
	return nil
}

func (m *ComplianceOperatorProfileV2) GetProduct() string {
	if m != nil {
		return m.Product
	}
	return ""
}

func (m *ComplianceOperatorProfileV2) GetTitle() string {
	if m != nil {
		return m.Title
	}
	return ""
}

func (m *ComplianceOperatorProfileV2) GetValues() []string {
	if m != nil {
		return m.Values
	}
	return nil
}

func (m *ComplianceOperatorProfileV2) MessageClone() proto.Message {
	return m.Clone()
}
func (m *ComplianceOperatorProfileV2) Clone() *ComplianceOperatorProfileV2 {
	if m == nil {
		return nil
	}
	cloned := new(ComplianceOperatorProfileV2)
	*cloned = *m

	if m.Labels != nil {
		cloned.Labels = make(map[string]string, len(m.Labels))
		for k, v := range m.Labels {
			cloned.Labels[k] = v
		}
	}
	if m.Annotations != nil {
		cloned.Annotations = make(map[string]string, len(m.Annotations))
		for k, v := range m.Annotations {
			cloned.Annotations[k] = v
		}
	}
	if m.Rules != nil {
		cloned.Rules = make([]*ComplianceOperatorProfileV2_Rule, len(m.Rules))
		for idx, v := range m.Rules {
			cloned.Rules[idx] = v.Clone()
		}
	}
	if m.Values != nil {
		cloned.Values = make([]string, len(m.Values))
		copy(cloned.Values, m.Values)
	}
	return cloned
}

type ComplianceOperatorProfileV2_Rule struct {
	RuleName             string   `protobuf:"bytes,1,opt,name=rule_name,json=ruleName,proto3" json:"rule_name,omitempty" search:"-" sql:"fk(ComplianceOperatorRuleV2:name),no-fk-constraint"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ComplianceOperatorProfileV2_Rule) Reset()         { *m = ComplianceOperatorProfileV2_Rule{} }
func (m *ComplianceOperatorProfileV2_Rule) String() string { return proto.CompactTextString(m) }
func (*ComplianceOperatorProfileV2_Rule) ProtoMessage()    {}
func (*ComplianceOperatorProfileV2_Rule) Descriptor() ([]byte, []int) {
	return fileDescriptor_26c2ec1f62102154, []int{1, 2}
}
func (m *ComplianceOperatorProfileV2_Rule) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ComplianceOperatorProfileV2_Rule) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ComplianceOperatorProfileV2_Rule.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ComplianceOperatorProfileV2_Rule) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ComplianceOperatorProfileV2_Rule.Merge(m, src)
}
func (m *ComplianceOperatorProfileV2_Rule) XXX_Size() int {
	return m.Size()
}
func (m *ComplianceOperatorProfileV2_Rule) XXX_DiscardUnknown() {
	xxx_messageInfo_ComplianceOperatorProfileV2_Rule.DiscardUnknown(m)
}

var xxx_messageInfo_ComplianceOperatorProfileV2_Rule proto.InternalMessageInfo

func (m *ComplianceOperatorProfileV2_Rule) GetRuleName() string {
	if m != nil {
		return m.RuleName
	}
	return ""
}

func (m *ComplianceOperatorProfileV2_Rule) MessageClone() proto.Message {
	return m.Clone()
}
func (m *ComplianceOperatorProfileV2_Rule) Clone() *ComplianceOperatorProfileV2_Rule {
	if m == nil {
		return nil
	}
	cloned := new(ComplianceOperatorProfileV2_Rule)
	*cloned = *m

	return cloned
}

// Next Tag: 15
type ComplianceOperatorRuleV2 struct {
	Id                   string                          `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty" sql:"pk"`
	RuleId               string                          `protobuf:"bytes,2,opt,name=rule_id,json=ruleId,proto3" json:"rule_id,omitempty"`
	Name                 string                          `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty" search:"Compliance Rule Name,hidden"`
	RuleType             string                          `protobuf:"bytes,4,opt,name=rule_type,json=ruleType,proto3" json:"rule_type,omitempty" search:"Compliance Rule Type,hidden"`
	Severity             RuleSeverity                    `protobuf:"varint,5,opt,name=severity,proto3,enum=storage.RuleSeverity" json:"severity,omitempty" search:"Compliance Rule Severity,hidden"`
	Labels               map[string]string               `protobuf:"bytes,6,rep,name=labels,proto3" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Annotations          map[string]string               `protobuf:"bytes,7,rep,name=annotations,proto3" json:"annotations,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Title                string                          `protobuf:"bytes,8,opt,name=title,proto3" json:"title,omitempty"`
	Description          string                          `protobuf:"bytes,9,opt,name=description,proto3" json:"description,omitempty"`
	Rationale            string                          `protobuf:"bytes,10,opt,name=rationale,proto3" json:"rationale,omitempty"`
	Fixes                []*ComplianceOperatorRuleV2_Fix `protobuf:"bytes,11,rep,name=fixes,proto3" json:"fixes,omitempty"`
	Warning              string                          `protobuf:"bytes,12,opt,name=warning,proto3" json:"warning,omitempty"`
	Controls             []*RuleControls                 `protobuf:"bytes,13,rep,name=controls,proto3" json:"controls,omitempty"`
	ClusterId            string                          `protobuf:"bytes,14,opt,name=cluster_id,json=clusterId,proto3" json:"cluster_id,omitempty" search:"Cluster ID,hidden" sql:"fk(Cluster:id),no-fk-constraint,type(uuid)"`
	XXX_NoUnkeyedLiteral struct{}                        `json:"-"`
	XXX_unrecognized     []byte                          `json:"-"`
	XXX_sizecache        int32                           `json:"-"`
}

func (m *ComplianceOperatorRuleV2) Reset()         { *m = ComplianceOperatorRuleV2{} }
func (m *ComplianceOperatorRuleV2) String() string { return proto.CompactTextString(m) }
func (*ComplianceOperatorRuleV2) ProtoMessage()    {}
func (*ComplianceOperatorRuleV2) Descriptor() ([]byte, []int) {
	return fileDescriptor_26c2ec1f62102154, []int{2}
}
func (m *ComplianceOperatorRuleV2) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ComplianceOperatorRuleV2) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ComplianceOperatorRuleV2.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ComplianceOperatorRuleV2) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ComplianceOperatorRuleV2.Merge(m, src)
}
func (m *ComplianceOperatorRuleV2) XXX_Size() int {
	return m.Size()
}
func (m *ComplianceOperatorRuleV2) XXX_DiscardUnknown() {
	xxx_messageInfo_ComplianceOperatorRuleV2.DiscardUnknown(m)
}

var xxx_messageInfo_ComplianceOperatorRuleV2 proto.InternalMessageInfo

func (m *ComplianceOperatorRuleV2) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *ComplianceOperatorRuleV2) GetRuleId() string {
	if m != nil {
		return m.RuleId
	}
	return ""
}

func (m *ComplianceOperatorRuleV2) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *ComplianceOperatorRuleV2) GetRuleType() string {
	if m != nil {
		return m.RuleType
	}
	return ""
}

func (m *ComplianceOperatorRuleV2) GetSeverity() RuleSeverity {
	if m != nil {
		return m.Severity
	}
	return RuleSeverity_UNSET_RULE_SEVERITY
}

func (m *ComplianceOperatorRuleV2) GetLabels() map[string]string {
	if m != nil {
		return m.Labels
	}
	return nil
}

func (m *ComplianceOperatorRuleV2) GetAnnotations() map[string]string {
	if m != nil {
		return m.Annotations
	}
	return nil
}

func (m *ComplianceOperatorRuleV2) GetTitle() string {
	if m != nil {
		return m.Title
	}
	return ""
}

func (m *ComplianceOperatorRuleV2) GetDescription() string {
	if m != nil {
		return m.Description
	}
	return ""
}

func (m *ComplianceOperatorRuleV2) GetRationale() string {
	if m != nil {
		return m.Rationale
	}
	return ""
}

func (m *ComplianceOperatorRuleV2) GetFixes() []*ComplianceOperatorRuleV2_Fix {
	if m != nil {
		return m.Fixes
	}
	return nil
}

func (m *ComplianceOperatorRuleV2) GetWarning() string {
	if m != nil {
		return m.Warning
	}
	return ""
}

func (m *ComplianceOperatorRuleV2) GetControls() []*RuleControls {
	if m != nil {
		return m.Controls
	}
	return nil
}

func (m *ComplianceOperatorRuleV2) GetClusterId() string {
	if m != nil {
		return m.ClusterId
	}
	return ""
}

func (m *ComplianceOperatorRuleV2) MessageClone() proto.Message {
	return m.Clone()
}
func (m *ComplianceOperatorRuleV2) Clone() *ComplianceOperatorRuleV2 {
	if m == nil {
		return nil
	}
	cloned := new(ComplianceOperatorRuleV2)
	*cloned = *m

	if m.Labels != nil {
		cloned.Labels = make(map[string]string, len(m.Labels))
		for k, v := range m.Labels {
			cloned.Labels[k] = v
		}
	}
	if m.Annotations != nil {
		cloned.Annotations = make(map[string]string, len(m.Annotations))
		for k, v := range m.Annotations {
			cloned.Annotations[k] = v
		}
	}
	if m.Fixes != nil {
		cloned.Fixes = make([]*ComplianceOperatorRuleV2_Fix, len(m.Fixes))
		for idx, v := range m.Fixes {
			cloned.Fixes[idx] = v.Clone()
		}
	}
	if m.Controls != nil {
		cloned.Controls = make([]*RuleControls, len(m.Controls))
		for idx, v := range m.Controls {
			cloned.Controls[idx] = v.Clone()
		}
	}
	return cloned
}

type ComplianceOperatorRuleV2_Fix struct {
	Platform             string   `protobuf:"bytes,1,opt,name=platform,proto3" json:"platform,omitempty"`
	Disruption           string   `protobuf:"bytes,2,opt,name=disruption,proto3" json:"disruption,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ComplianceOperatorRuleV2_Fix) Reset()         { *m = ComplianceOperatorRuleV2_Fix{} }
func (m *ComplianceOperatorRuleV2_Fix) String() string { return proto.CompactTextString(m) }
func (*ComplianceOperatorRuleV2_Fix) ProtoMessage()    {}
func (*ComplianceOperatorRuleV2_Fix) Descriptor() ([]byte, []int) {
	return fileDescriptor_26c2ec1f62102154, []int{2, 2}
}
func (m *ComplianceOperatorRuleV2_Fix) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ComplianceOperatorRuleV2_Fix) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ComplianceOperatorRuleV2_Fix.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ComplianceOperatorRuleV2_Fix) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ComplianceOperatorRuleV2_Fix.Merge(m, src)
}
func (m *ComplianceOperatorRuleV2_Fix) XXX_Size() int {
	return m.Size()
}
func (m *ComplianceOperatorRuleV2_Fix) XXX_DiscardUnknown() {
	xxx_messageInfo_ComplianceOperatorRuleV2_Fix.DiscardUnknown(m)
}

var xxx_messageInfo_ComplianceOperatorRuleV2_Fix proto.InternalMessageInfo

func (m *ComplianceOperatorRuleV2_Fix) GetPlatform() string {
	if m != nil {
		return m.Platform
	}
	return ""
}

func (m *ComplianceOperatorRuleV2_Fix) GetDisruption() string {
	if m != nil {
		return m.Disruption
	}
	return ""
}

func (m *ComplianceOperatorRuleV2_Fix) MessageClone() proto.Message {
	return m.Clone()
}
func (m *ComplianceOperatorRuleV2_Fix) Clone() *ComplianceOperatorRuleV2_Fix {
	if m == nil {
		return nil
	}
	cloned := new(ComplianceOperatorRuleV2_Fix)
	*cloned = *m

	return cloned
}

// Next Tag: 3
type RuleControls struct {
	Standard             string   `protobuf:"bytes,1,opt,name=standard,proto3" json:"standard,omitempty" search:"Compliance Standard,hidden"`
	Controls             []string `protobuf:"bytes,2,rep,name=controls,proto3" json:"controls,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *RuleControls) Reset()         { *m = RuleControls{} }
func (m *RuleControls) String() string { return proto.CompactTextString(m) }
func (*RuleControls) ProtoMessage()    {}
func (*RuleControls) Descriptor() ([]byte, []int) {
	return fileDescriptor_26c2ec1f62102154, []int{3}
}
func (m *RuleControls) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *RuleControls) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_RuleControls.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *RuleControls) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RuleControls.Merge(m, src)
}
func (m *RuleControls) XXX_Size() int {
	return m.Size()
}
func (m *RuleControls) XXX_DiscardUnknown() {
	xxx_messageInfo_RuleControls.DiscardUnknown(m)
}

var xxx_messageInfo_RuleControls proto.InternalMessageInfo

func (m *RuleControls) GetStandard() string {
	if m != nil {
		return m.Standard
	}
	return ""
}

func (m *RuleControls) GetControls() []string {
	if m != nil {
		return m.Controls
	}
	return nil
}

func (m *RuleControls) MessageClone() proto.Message {
	return m.Clone()
}
func (m *RuleControls) Clone() *RuleControls {
	if m == nil {
		return nil
	}
	cloned := new(RuleControls)
	*cloned = *m

	if m.Controls != nil {
		cloned.Controls = make([]string, len(m.Controls))
		copy(cloned.Controls, m.Controls)
	}
	return cloned
}

// Next Tag: 16
type ComplianceOperatorScanConfigurationV2 struct {
	Id                     string            `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty" search:"Compliance Scan Config ID,hidden" sql:"pk,type(uuid)"`
	ScanConfigName         string            `protobuf:"bytes,2,opt,name=scan_config_name,json=scanConfigName,proto3" json:"scan_config_name,omitempty" search:"Compliance Scan Config Name" sql:"unique"`
	AutoApplyRemediations  bool              `protobuf:"varint,3,opt,name=auto_apply_remediations,json=autoApplyRemediations,proto3" json:"auto_apply_remediations,omitempty"`
	AutoUpdateRemediations bool              `protobuf:"varint,4,opt,name=auto_update_remediations,json=autoUpdateRemediations,proto3" json:"auto_update_remediations,omitempty"`
	OneTimeScan            bool              `protobuf:"varint,5,opt,name=one_time_scan,json=oneTimeScan,proto3" json:"one_time_scan,omitempty"`
	Labels                 map[string]string `protobuf:"bytes,6,rep,name=labels,proto3" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Annotations            map[string]string `protobuf:"bytes,7,rep,name=annotations,proto3" json:"annotations,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Profiles               []*ProfileShim    `protobuf:"bytes,8,rep,name=profiles,proto3" json:"profiles,omitempty"`
	NodeRoles              []NodeRole        `protobuf:"varint,9,rep,packed,name=node_roles,json=nodeRoles,proto3,enum=storage.NodeRole" json:"node_roles,omitempty"`
	// Will be configurable via env var
	StrictNodeScan bool `protobuf:"varint,10,opt,name=strict_node_scan,json=strictNodeScan,proto3" json:"strict_node_scan,omitempty"`
	// Starting point for schedule will probably have to build upon it
	Schedule        *Schedule        `protobuf:"bytes,11,opt,name=schedule,proto3" json:"schedule,omitempty"`
	CreatedTime     *types.Timestamp `protobuf:"bytes,12,opt,name=created_time,json=createdTime,proto3" json:"created_time,omitempty"`
	LastUpdatedTime *types.Timestamp `protobuf:"bytes,13,opt,name=last_updated_time,json=lastUpdatedTime,proto3" json:"last_updated_time,omitempty"`
	// Most recent user to update the scan configurations
	ModifiedBy           *SlimUser                                        `protobuf:"bytes,14,opt,name=modified_by,json=modifiedBy,proto3" json:"modified_by,omitempty" sql:"ignore_labels(User ID)"`
	Description          string                                           `protobuf:"bytes,15,opt,name=description,proto3" json:"description,omitempty"`
	Clusters             []*ComplianceOperatorScanConfigurationV2_Cluster `protobuf:"bytes,16,rep,name=clusters,proto3" json:"clusters,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                                         `json:"-"`
	XXX_unrecognized     []byte                                           `json:"-"`
	XXX_sizecache        int32                                            `json:"-"`
}

func (m *ComplianceOperatorScanConfigurationV2) Reset()         { *m = ComplianceOperatorScanConfigurationV2{} }
func (m *ComplianceOperatorScanConfigurationV2) String() string { return proto.CompactTextString(m) }
func (*ComplianceOperatorScanConfigurationV2) ProtoMessage()    {}
func (*ComplianceOperatorScanConfigurationV2) Descriptor() ([]byte, []int) {
	return fileDescriptor_26c2ec1f62102154, []int{4}
}
func (m *ComplianceOperatorScanConfigurationV2) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ComplianceOperatorScanConfigurationV2) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ComplianceOperatorScanConfigurationV2.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ComplianceOperatorScanConfigurationV2) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ComplianceOperatorScanConfigurationV2.Merge(m, src)
}
func (m *ComplianceOperatorScanConfigurationV2) XXX_Size() int {
	return m.Size()
}
func (m *ComplianceOperatorScanConfigurationV2) XXX_DiscardUnknown() {
	xxx_messageInfo_ComplianceOperatorScanConfigurationV2.DiscardUnknown(m)
}

var xxx_messageInfo_ComplianceOperatorScanConfigurationV2 proto.InternalMessageInfo

func (m *ComplianceOperatorScanConfigurationV2) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *ComplianceOperatorScanConfigurationV2) GetScanConfigName() string {
	if m != nil {
		return m.ScanConfigName
	}
	return ""
}

func (m *ComplianceOperatorScanConfigurationV2) GetAutoApplyRemediations() bool {
	if m != nil {
		return m.AutoApplyRemediations
	}
	return false
}

func (m *ComplianceOperatorScanConfigurationV2) GetAutoUpdateRemediations() bool {
	if m != nil {
		return m.AutoUpdateRemediations
	}
	return false
}

func (m *ComplianceOperatorScanConfigurationV2) GetOneTimeScan() bool {
	if m != nil {
		return m.OneTimeScan
	}
	return false
}

func (m *ComplianceOperatorScanConfigurationV2) GetLabels() map[string]string {
	if m != nil {
		return m.Labels
	}
	return nil
}

func (m *ComplianceOperatorScanConfigurationV2) GetAnnotations() map[string]string {
	if m != nil {
		return m.Annotations
	}
	return nil
}

func (m *ComplianceOperatorScanConfigurationV2) GetProfiles() []*ProfileShim {
	if m != nil {
		return m.Profiles
	}
	return nil
}

func (m *ComplianceOperatorScanConfigurationV2) GetNodeRoles() []NodeRole {
	if m != nil {
		return m.NodeRoles
	}
	return nil
}

func (m *ComplianceOperatorScanConfigurationV2) GetStrictNodeScan() bool {
	if m != nil {
		return m.StrictNodeScan
	}
	return false
}

func (m *ComplianceOperatorScanConfigurationV2) GetSchedule() *Schedule {
	if m != nil {
		return m.Schedule
	}
	return nil
}

func (m *ComplianceOperatorScanConfigurationV2) GetCreatedTime() *types.Timestamp {
	if m != nil {
		return m.CreatedTime
	}
	return nil
}

func (m *ComplianceOperatorScanConfigurationV2) GetLastUpdatedTime() *types.Timestamp {
	if m != nil {
		return m.LastUpdatedTime
	}
	return nil
}

func (m *ComplianceOperatorScanConfigurationV2) GetModifiedBy() *SlimUser {
	if m != nil {
		return m.ModifiedBy
	}
	return nil
}

func (m *ComplianceOperatorScanConfigurationV2) GetDescription() string {
	if m != nil {
		return m.Description
	}
	return ""
}

func (m *ComplianceOperatorScanConfigurationV2) GetClusters() []*ComplianceOperatorScanConfigurationV2_Cluster {
	if m != nil {
		return m.Clusters
	}
	return nil
}

func (m *ComplianceOperatorScanConfigurationV2) MessageClone() proto.Message {
	return m.Clone()
}
func (m *ComplianceOperatorScanConfigurationV2) Clone() *ComplianceOperatorScanConfigurationV2 {
	if m == nil {
		return nil
	}
	cloned := new(ComplianceOperatorScanConfigurationV2)
	*cloned = *m

	if m.Labels != nil {
		cloned.Labels = make(map[string]string, len(m.Labels))
		for k, v := range m.Labels {
			cloned.Labels[k] = v
		}
	}
	if m.Annotations != nil {
		cloned.Annotations = make(map[string]string, len(m.Annotations))
		for k, v := range m.Annotations {
			cloned.Annotations[k] = v
		}
	}
	if m.Profiles != nil {
		cloned.Profiles = make([]*ProfileShim, len(m.Profiles))
		for idx, v := range m.Profiles {
			cloned.Profiles[idx] = v.Clone()
		}
	}
	if m.NodeRoles != nil {
		cloned.NodeRoles = make([]NodeRole, len(m.NodeRoles))
		copy(cloned.NodeRoles, m.NodeRoles)
	}
	cloned.Schedule = m.Schedule.Clone()
	cloned.CreatedTime = m.CreatedTime.Clone()
	cloned.LastUpdatedTime = m.LastUpdatedTime.Clone()
	cloned.ModifiedBy = m.ModifiedBy.Clone()
	if m.Clusters != nil {
		cloned.Clusters = make([]*ComplianceOperatorScanConfigurationV2_Cluster, len(m.Clusters))
		for idx, v := range m.Clusters {
			cloned.Clusters[idx] = v.Clone()
		}
	}
	return cloned
}

type ComplianceOperatorScanConfigurationV2_Cluster struct {
	ClusterId            string   `protobuf:"bytes,1,opt,name=cluster_id,json=clusterId,proto3" json:"cluster_id,omitempty" search:"Cluster ID,hidden" sql:"fk(Cluster:id),no-fk-constraint,type(uuid)"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ComplianceOperatorScanConfigurationV2_Cluster) Reset() {
	*m = ComplianceOperatorScanConfigurationV2_Cluster{}
}
func (m *ComplianceOperatorScanConfigurationV2_Cluster) String() string {
	return proto.CompactTextString(m)
}
func (*ComplianceOperatorScanConfigurationV2_Cluster) ProtoMessage() {}
func (*ComplianceOperatorScanConfigurationV2_Cluster) Descriptor() ([]byte, []int) {
	return fileDescriptor_26c2ec1f62102154, []int{4, 2}
}
func (m *ComplianceOperatorScanConfigurationV2_Cluster) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ComplianceOperatorScanConfigurationV2_Cluster) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ComplianceOperatorScanConfigurationV2_Cluster.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ComplianceOperatorScanConfigurationV2_Cluster) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ComplianceOperatorScanConfigurationV2_Cluster.Merge(m, src)
}
func (m *ComplianceOperatorScanConfigurationV2_Cluster) XXX_Size() int {
	return m.Size()
}
func (m *ComplianceOperatorScanConfigurationV2_Cluster) XXX_DiscardUnknown() {
	xxx_messageInfo_ComplianceOperatorScanConfigurationV2_Cluster.DiscardUnknown(m)
}

var xxx_messageInfo_ComplianceOperatorScanConfigurationV2_Cluster proto.InternalMessageInfo

func (m *ComplianceOperatorScanConfigurationV2_Cluster) GetClusterId() string {
	if m != nil {
		return m.ClusterId
	}
	return ""
}

func (m *ComplianceOperatorScanConfigurationV2_Cluster) MessageClone() proto.Message {
	return m.Clone()
}
func (m *ComplianceOperatorScanConfigurationV2_Cluster) Clone() *ComplianceOperatorScanConfigurationV2_Cluster {
	if m == nil {
		return nil
	}
	cloned := new(ComplianceOperatorScanConfigurationV2_Cluster)
	*cloned = *m

	return cloned
}

// Next Tag: 7
// Cluster and an error if necessary to handle cases where the scan configuration is
// unable to be applied to a cluster for whatever reason.
type ComplianceOperatorClusterScanConfigStatus struct {
	Id                   string           `protobuf:"bytes,6,opt,name=id,proto3" json:"id,omitempty" sql:"pk,type(uuid)"`
	ClusterId            string           `protobuf:"bytes,1,opt,name=cluster_id,json=clusterId,proto3" json:"cluster_id,omitempty" search:"Cluster ID,hidden" sql:"fk(Cluster:id),no-fk-constraint,type(uuid)"`
	ScanConfigId         string           `protobuf:"bytes,2,opt,name=scan_config_id,json=scanConfigId,proto3" json:"scan_config_id,omitempty" search:"Compliance Scan Config ID,hidden" sql:"fk(ComplianceOperatorScanConfigurationV2:id),no-fk-constraint,type(uuid)"`
	Errors               []string         `protobuf:"bytes,3,rep,name=errors,proto3" json:"errors,omitempty"`
	LastUpdatedTime      *types.Timestamp `protobuf:"bytes,4,opt,name=last_updated_time,json=lastUpdatedTime,proto3" json:"last_updated_time,omitempty" search:"Compliance Scan Config Last Updated Time,hidden"`
	ClusterName          string           `protobuf:"bytes,5,opt,name=cluster_name,json=clusterName,proto3" json:"cluster_name,omitempty"`
	XXX_NoUnkeyedLiteral struct{}         `json:"-"`
	XXX_unrecognized     []byte           `json:"-"`
	XXX_sizecache        int32            `json:"-"`
}

func (m *ComplianceOperatorClusterScanConfigStatus) Reset() {
	*m = ComplianceOperatorClusterScanConfigStatus{}
}
func (m *ComplianceOperatorClusterScanConfigStatus) String() string {
	return proto.CompactTextString(m)
}
func (*ComplianceOperatorClusterScanConfigStatus) ProtoMessage() {}
func (*ComplianceOperatorClusterScanConfigStatus) Descriptor() ([]byte, []int) {
	return fileDescriptor_26c2ec1f62102154, []int{5}
}
func (m *ComplianceOperatorClusterScanConfigStatus) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ComplianceOperatorClusterScanConfigStatus) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ComplianceOperatorClusterScanConfigStatus.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ComplianceOperatorClusterScanConfigStatus) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ComplianceOperatorClusterScanConfigStatus.Merge(m, src)
}
func (m *ComplianceOperatorClusterScanConfigStatus) XXX_Size() int {
	return m.Size()
}
func (m *ComplianceOperatorClusterScanConfigStatus) XXX_DiscardUnknown() {
	xxx_messageInfo_ComplianceOperatorClusterScanConfigStatus.DiscardUnknown(m)
}

var xxx_messageInfo_ComplianceOperatorClusterScanConfigStatus proto.InternalMessageInfo

func (m *ComplianceOperatorClusterScanConfigStatus) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *ComplianceOperatorClusterScanConfigStatus) GetClusterId() string {
	if m != nil {
		return m.ClusterId
	}
	return ""
}

func (m *ComplianceOperatorClusterScanConfigStatus) GetScanConfigId() string {
	if m != nil {
		return m.ScanConfigId
	}
	return ""
}

func (m *ComplianceOperatorClusterScanConfigStatus) GetErrors() []string {
	if m != nil {
		return m.Errors
	}
	return nil
}

func (m *ComplianceOperatorClusterScanConfigStatus) GetLastUpdatedTime() *types.Timestamp {
	if m != nil {
		return m.LastUpdatedTime
	}
	return nil
}

func (m *ComplianceOperatorClusterScanConfigStatus) GetClusterName() string {
	if m != nil {
		return m.ClusterName
	}
	return ""
}

func (m *ComplianceOperatorClusterScanConfigStatus) MessageClone() proto.Message {
	return m.Clone()
}
func (m *ComplianceOperatorClusterScanConfigStatus) Clone() *ComplianceOperatorClusterScanConfigStatus {
	if m == nil {
		return nil
	}
	cloned := new(ComplianceOperatorClusterScanConfigStatus)
	*cloned = *m

	if m.Errors != nil {
		cloned.Errors = make([]string, len(m.Errors))
		copy(cloned.Errors, m.Errors)
	}
	cloned.LastUpdatedTime = m.LastUpdatedTime.Clone()
	return cloned
}

// Next Tag: 20
// This object has been flattened vs joining with rule.  The rationale is to spend the time to query rule
// while processing results vs reporting them to the user.  Additionally, flattening it helps with the historical data
// as the rules can change without impacting the historical result.
type ComplianceOperatorCheckResultV2 struct {
	Id                   string                                      `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty" search:"Compliance Check UID,hidden" sql:"pk"`
	CheckId              string                                      `protobuf:"bytes,2,opt,name=check_id,json=checkId,proto3" json:"check_id,omitempty" search:"Compliance Check ID,hidden"`
	CheckName            string                                      `protobuf:"bytes,3,opt,name=check_name,json=checkName,proto3" json:"check_name,omitempty" search:"Compliance Check Name,hidden"`
	ClusterId            string                                      `protobuf:"bytes,4,opt,name=cluster_id,json=clusterId,proto3" json:"cluster_id,omitempty" search:"Cluster ID,hidden" sql:"fk(Cluster:id),no-fk-constraint,type(uuid)"`
	ClusterName          string                                      `protobuf:"bytes,15,opt,name=cluster_name,json=clusterName,proto3" json:"cluster_name,omitempty"`
	Status               ComplianceOperatorCheckResultV2_CheckStatus `protobuf:"varint,5,opt,name=status,proto3,enum=storage.ComplianceOperatorCheckResultV2_CheckStatus" json:"status,omitempty" search:"Compliance Check Status,hidden"`
	Severity             RuleSeverity                                `protobuf:"varint,6,opt,name=severity,proto3,enum=storage.RuleSeverity" json:"severity,omitempty" search:"Compliance Rule Severity,hidden"`
	Description          string                                      `protobuf:"bytes,7,opt,name=description,proto3" json:"description,omitempty"`
	Instructions         string                                      `protobuf:"bytes,8,opt,name=instructions,proto3" json:"instructions,omitempty"`
	Labels               map[string]string                           `protobuf:"bytes,9,rep,name=labels,proto3" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Annotations          map[string]string                           `protobuf:"bytes,10,rep,name=annotations,proto3" json:"annotations,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	CreatedTime          *types.Timestamp                            `protobuf:"bytes,11,opt,name=created_time,json=createdTime,proto3" json:"created_time,omitempty" search:"Compliance Check Result Created Time,hidden"`
	Standard             string                                      `protobuf:"bytes,12,opt,name=standard,proto3" json:"standard,omitempty" search:"Compliance Standard,hidden"`
	Control              string                                      `protobuf:"bytes,13,opt,name=control,proto3" json:"control,omitempty"`
	ScanName             string                                      `protobuf:"bytes,14,opt,name=scan_name,json=scanName,proto3" json:"scan_name,omitempty" search:"Compliance Scan Name,hidden" sql:"fk(ComplianceOperatorScanV2:scan_name),no-fk-constraint"`
	ScanConfigName       string                                      `protobuf:"bytes,16,opt,name=scan_config_name,json=scanConfigName,proto3" json:"scan_config_name,omitempty" search:"Compliance Scan Config Name" sql:"fk(ComplianceOperatorScanConfigurationV2:scan_config_name),no-fk-constraint"`
	Rationale            string                                      `protobuf:"bytes,17,opt,name=rationale,proto3" json:"rationale,omitempty"`
	ValuesUsed           []string                                    `protobuf:"bytes,18,rep,name=valuesUsed,proto3" json:"valuesUsed,omitempty"`
	Warnings             []string                                    `protobuf:"bytes,19,rep,name=warnings,proto3" json:"warnings,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                                    `json:"-"`
	XXX_unrecognized     []byte                                      `json:"-"`
	XXX_sizecache        int32                                       `json:"-"`
}

func (m *ComplianceOperatorCheckResultV2) Reset()         { *m = ComplianceOperatorCheckResultV2{} }
func (m *ComplianceOperatorCheckResultV2) String() string { return proto.CompactTextString(m) }
func (*ComplianceOperatorCheckResultV2) ProtoMessage()    {}
func (*ComplianceOperatorCheckResultV2) Descriptor() ([]byte, []int) {
	return fileDescriptor_26c2ec1f62102154, []int{6}
}
func (m *ComplianceOperatorCheckResultV2) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ComplianceOperatorCheckResultV2) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ComplianceOperatorCheckResultV2.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ComplianceOperatorCheckResultV2) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ComplianceOperatorCheckResultV2.Merge(m, src)
}
func (m *ComplianceOperatorCheckResultV2) XXX_Size() int {
	return m.Size()
}
func (m *ComplianceOperatorCheckResultV2) XXX_DiscardUnknown() {
	xxx_messageInfo_ComplianceOperatorCheckResultV2.DiscardUnknown(m)
}

var xxx_messageInfo_ComplianceOperatorCheckResultV2 proto.InternalMessageInfo

func (m *ComplianceOperatorCheckResultV2) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *ComplianceOperatorCheckResultV2) GetCheckId() string {
	if m != nil {
		return m.CheckId
	}
	return ""
}

func (m *ComplianceOperatorCheckResultV2) GetCheckName() string {
	if m != nil {
		return m.CheckName
	}
	return ""
}

func (m *ComplianceOperatorCheckResultV2) GetClusterId() string {
	if m != nil {
		return m.ClusterId
	}
	return ""
}

func (m *ComplianceOperatorCheckResultV2) GetClusterName() string {
	if m != nil {
		return m.ClusterName
	}
	return ""
}

func (m *ComplianceOperatorCheckResultV2) GetStatus() ComplianceOperatorCheckResultV2_CheckStatus {
	if m != nil {
		return m.Status
	}
	return ComplianceOperatorCheckResultV2_UNSET
}

func (m *ComplianceOperatorCheckResultV2) GetSeverity() RuleSeverity {
	if m != nil {
		return m.Severity
	}
	return RuleSeverity_UNSET_RULE_SEVERITY
}

func (m *ComplianceOperatorCheckResultV2) GetDescription() string {
	if m != nil {
		return m.Description
	}
	return ""
}

func (m *ComplianceOperatorCheckResultV2) GetInstructions() string {
	if m != nil {
		return m.Instructions
	}
	return ""
}

func (m *ComplianceOperatorCheckResultV2) GetLabels() map[string]string {
	if m != nil {
		return m.Labels
	}
	return nil
}

func (m *ComplianceOperatorCheckResultV2) GetAnnotations() map[string]string {
	if m != nil {
		return m.Annotations
	}
	return nil
}

func (m *ComplianceOperatorCheckResultV2) GetCreatedTime() *types.Timestamp {
	if m != nil {
		return m.CreatedTime
	}
	return nil
}

func (m *ComplianceOperatorCheckResultV2) GetStandard() string {
	if m != nil {
		return m.Standard
	}
	return ""
}

func (m *ComplianceOperatorCheckResultV2) GetControl() string {
	if m != nil {
		return m.Control
	}
	return ""
}

func (m *ComplianceOperatorCheckResultV2) GetScanName() string {
	if m != nil {
		return m.ScanName
	}
	return ""
}

func (m *ComplianceOperatorCheckResultV2) GetScanConfigName() string {
	if m != nil {
		return m.ScanConfigName
	}
	return ""
}

func (m *ComplianceOperatorCheckResultV2) GetRationale() string {
	if m != nil {
		return m.Rationale
	}
	return ""
}

func (m *ComplianceOperatorCheckResultV2) GetValuesUsed() []string {
	if m != nil {
		return m.ValuesUsed
	}
	return nil
}

func (m *ComplianceOperatorCheckResultV2) GetWarnings() []string {
	if m != nil {
		return m.Warnings
	}
	return nil
}

func (m *ComplianceOperatorCheckResultV2) MessageClone() proto.Message {
	return m.Clone()
}
func (m *ComplianceOperatorCheckResultV2) Clone() *ComplianceOperatorCheckResultV2 {
	if m == nil {
		return nil
	}
	cloned := new(ComplianceOperatorCheckResultV2)
	*cloned = *m

	if m.Labels != nil {
		cloned.Labels = make(map[string]string, len(m.Labels))
		for k, v := range m.Labels {
			cloned.Labels[k] = v
		}
	}
	if m.Annotations != nil {
		cloned.Annotations = make(map[string]string, len(m.Annotations))
		for k, v := range m.Annotations {
			cloned.Annotations[k] = v
		}
	}
	cloned.CreatedTime = m.CreatedTime.Clone()
	if m.ValuesUsed != nil {
		cloned.ValuesUsed = make([]string, len(m.ValuesUsed))
		copy(cloned.ValuesUsed, m.ValuesUsed)
	}
	if m.Warnings != nil {
		cloned.Warnings = make([]string, len(m.Warnings))
		copy(cloned.Warnings, m.Warnings)
	}
	return cloned
}

type ScanStatus struct {
	Phase                string   `protobuf:"bytes,1,opt,name=phase,proto3" json:"phase,omitempty"`
	Result               string   `protobuf:"bytes,2,opt,name=result,proto3" json:"result,omitempty"`
	Warnings             string   `protobuf:"bytes,3,opt,name=warnings,proto3" json:"warnings,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ScanStatus) Reset()         { *m = ScanStatus{} }
func (m *ScanStatus) String() string { return proto.CompactTextString(m) }
func (*ScanStatus) ProtoMessage()    {}
func (*ScanStatus) Descriptor() ([]byte, []int) {
	return fileDescriptor_26c2ec1f62102154, []int{7}
}
func (m *ScanStatus) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ScanStatus) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ScanStatus.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ScanStatus) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ScanStatus.Merge(m, src)
}
func (m *ScanStatus) XXX_Size() int {
	return m.Size()
}
func (m *ScanStatus) XXX_DiscardUnknown() {
	xxx_messageInfo_ScanStatus.DiscardUnknown(m)
}

var xxx_messageInfo_ScanStatus proto.InternalMessageInfo

func (m *ScanStatus) GetPhase() string {
	if m != nil {
		return m.Phase
	}
	return ""
}

func (m *ScanStatus) GetResult() string {
	if m != nil {
		return m.Result
	}
	return ""
}

func (m *ScanStatus) GetWarnings() string {
	if m != nil {
		return m.Warnings
	}
	return ""
}

func (m *ScanStatus) MessageClone() proto.Message {
	return m.Clone()
}
func (m *ScanStatus) Clone() *ScanStatus {
	if m == nil {
		return nil
	}
	cloned := new(ScanStatus)
	*cloned = *m

	return cloned
}

// Next Tag: 15
// Scan object per cluster
type ComplianceOperatorScanV2 struct {
	Id                   string            `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty" sql:"pk"`
	ScanConfigName       string            `protobuf:"bytes,2,opt,name=scan_config_name,json=scanConfigName,proto3" json:"scan_config_name,omitempty" search:"Compliance Scan Config Name" sql:"fk(ComplianceOperatorScanConfigurationV2:scan_config_name),no-fk-constraint"`
	ScanName             string            `protobuf:"bytes,13,opt,name=scan_name,json=scanName,proto3" json:"scan_name,omitempty" search:"Compliance Scan Name,hidden"`
	ClusterId            string            `protobuf:"bytes,3,opt,name=cluster_id,json=clusterId,proto3" json:"cluster_id,omitempty" search:"Cluster ID,hidden" sql:"fk(Cluster:id),no-fk-constraint,type(uuid)"`
	Errors               string            `protobuf:"bytes,4,opt,name=errors,proto3" json:"errors,omitempty"`
	Warnings             string            `protobuf:"bytes,14,opt,name=warnings,proto3" json:"warnings,omitempty"`
	Profile              *ProfileShim      `protobuf:"bytes,5,opt,name=profile,proto3" json:"profile,omitempty"`
	Labels               map[string]string `protobuf:"bytes,6,rep,name=labels,proto3" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Annotations          map[string]string `protobuf:"bytes,7,rep,name=annotations,proto3" json:"annotations,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	ScanType             ScanType          `protobuf:"varint,8,opt,name=scan_type,json=scanType,proto3,enum=storage.ScanType" json:"scan_type,omitempty"`
	NodeSelector         NodeRole          `protobuf:"varint,9,opt,name=node_selector,json=nodeSelector,proto3,enum=storage.NodeRole" json:"node_selector,omitempty"`
	Status               *ScanStatus       `protobuf:"bytes,10,opt,name=status,proto3" json:"status,omitempty"`
	CreatedTime          *types.Timestamp  `protobuf:"bytes,11,opt,name=created_time,json=createdTime,proto3" json:"created_time,omitempty"`
	LastExecutedTime     *types.Timestamp  `protobuf:"bytes,12,opt,name=last_executed_time,json=lastExecutedTime,proto3" json:"last_executed_time,omitempty" search:"Compliance Scan Last Executed Time,hidden"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *ComplianceOperatorScanV2) Reset()         { *m = ComplianceOperatorScanV2{} }
func (m *ComplianceOperatorScanV2) String() string { return proto.CompactTextString(m) }
func (*ComplianceOperatorScanV2) ProtoMessage()    {}
func (*ComplianceOperatorScanV2) Descriptor() ([]byte, []int) {
	return fileDescriptor_26c2ec1f62102154, []int{8}
}
func (m *ComplianceOperatorScanV2) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ComplianceOperatorScanV2) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ComplianceOperatorScanV2.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ComplianceOperatorScanV2) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ComplianceOperatorScanV2.Merge(m, src)
}
func (m *ComplianceOperatorScanV2) XXX_Size() int {
	return m.Size()
}
func (m *ComplianceOperatorScanV2) XXX_DiscardUnknown() {
	xxx_messageInfo_ComplianceOperatorScanV2.DiscardUnknown(m)
}

var xxx_messageInfo_ComplianceOperatorScanV2 proto.InternalMessageInfo

func (m *ComplianceOperatorScanV2) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *ComplianceOperatorScanV2) GetScanConfigName() string {
	if m != nil {
		return m.ScanConfigName
	}
	return ""
}

func (m *ComplianceOperatorScanV2) GetScanName() string {
	if m != nil {
		return m.ScanName
	}
	return ""
}

func (m *ComplianceOperatorScanV2) GetClusterId() string {
	if m != nil {
		return m.ClusterId
	}
	return ""
}

func (m *ComplianceOperatorScanV2) GetErrors() string {
	if m != nil {
		return m.Errors
	}
	return ""
}

func (m *ComplianceOperatorScanV2) GetWarnings() string {
	if m != nil {
		return m.Warnings
	}
	return ""
}

func (m *ComplianceOperatorScanV2) GetProfile() *ProfileShim {
	if m != nil {
		return m.Profile
	}
	return nil
}

func (m *ComplianceOperatorScanV2) GetLabels() map[string]string {
	if m != nil {
		return m.Labels
	}
	return nil
}

func (m *ComplianceOperatorScanV2) GetAnnotations() map[string]string {
	if m != nil {
		return m.Annotations
	}
	return nil
}

func (m *ComplianceOperatorScanV2) GetScanType() ScanType {
	if m != nil {
		return m.ScanType
	}
	return ScanType_UNSET_SCAN_TYPE
}

func (m *ComplianceOperatorScanV2) GetNodeSelector() NodeRole {
	if m != nil {
		return m.NodeSelector
	}
	return NodeRole_INFRA
}

func (m *ComplianceOperatorScanV2) GetStatus() *ScanStatus {
	if m != nil {
		return m.Status
	}
	return nil
}

func (m *ComplianceOperatorScanV2) GetCreatedTime() *types.Timestamp {
	if m != nil {
		return m.CreatedTime
	}
	return nil
}

func (m *ComplianceOperatorScanV2) GetLastExecutedTime() *types.Timestamp {
	if m != nil {
		return m.LastExecutedTime
	}
	return nil
}

func (m *ComplianceOperatorScanV2) MessageClone() proto.Message {
	return m.Clone()
}
func (m *ComplianceOperatorScanV2) Clone() *ComplianceOperatorScanV2 {
	if m == nil {
		return nil
	}
	cloned := new(ComplianceOperatorScanV2)
	*cloned = *m

	cloned.Profile = m.Profile.Clone()
	if m.Labels != nil {
		cloned.Labels = make(map[string]string, len(m.Labels))
		for k, v := range m.Labels {
			cloned.Labels[k] = v
		}
	}
	if m.Annotations != nil {
		cloned.Annotations = make(map[string]string, len(m.Annotations))
		for k, v := range m.Annotations {
			cloned.Annotations[k] = v
		}
	}
	cloned.Status = m.Status.Clone()
	cloned.CreatedTime = m.CreatedTime.Clone()
	cloned.LastExecutedTime = m.LastExecutedTime.Clone()
	return cloned
}

// ComplianceOperatorProfileClusterEdge maps which profiles exist on which clusters
// Next Tag: 4
type ComplianceOperatorProfileClusterEdge struct {
	Id                   string   `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty" sql:"pk,id"`
	ProfileId            string   `protobuf:"bytes,2,opt,name=profile_id,json=profileId,proto3" json:"profile_id,omitempty" search:"Compliance Profile ID,hidden" sql:"fk(ComplianceOperatorProfileV2:id)"`
	ProfileUid           string   `protobuf:"bytes,3,opt,name=profile_uid,json=profileUid,proto3" json:"profile_uid,omitempty" search:"Compliance Profile UID,hidden"`
	ClusterId            string   `protobuf:"bytes,4,opt,name=cluster_id,json=clusterId,proto3" json:"cluster_id,omitempty" search:"Cluster ID,hidden" sql:"fk(Cluster:id),no-fk-constraint,type(uuid)"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ComplianceOperatorProfileClusterEdge) Reset()         { *m = ComplianceOperatorProfileClusterEdge{} }
func (m *ComplianceOperatorProfileClusterEdge) String() string { return proto.CompactTextString(m) }
func (*ComplianceOperatorProfileClusterEdge) ProtoMessage()    {}
func (*ComplianceOperatorProfileClusterEdge) Descriptor() ([]byte, []int) {
	return fileDescriptor_26c2ec1f62102154, []int{9}
}
func (m *ComplianceOperatorProfileClusterEdge) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ComplianceOperatorProfileClusterEdge) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ComplianceOperatorProfileClusterEdge.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ComplianceOperatorProfileClusterEdge) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ComplianceOperatorProfileClusterEdge.Merge(m, src)
}
func (m *ComplianceOperatorProfileClusterEdge) XXX_Size() int {
	return m.Size()
}
func (m *ComplianceOperatorProfileClusterEdge) XXX_DiscardUnknown() {
	xxx_messageInfo_ComplianceOperatorProfileClusterEdge.DiscardUnknown(m)
}

var xxx_messageInfo_ComplianceOperatorProfileClusterEdge proto.InternalMessageInfo

func (m *ComplianceOperatorProfileClusterEdge) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *ComplianceOperatorProfileClusterEdge) GetProfileId() string {
	if m != nil {
		return m.ProfileId
	}
	return ""
}

func (m *ComplianceOperatorProfileClusterEdge) GetProfileUid() string {
	if m != nil {
		return m.ProfileUid
	}
	return ""
}

func (m *ComplianceOperatorProfileClusterEdge) GetClusterId() string {
	if m != nil {
		return m.ClusterId
	}
	return ""
}

func (m *ComplianceOperatorProfileClusterEdge) MessageClone() proto.Message {
	return m.Clone()
}
func (m *ComplianceOperatorProfileClusterEdge) Clone() *ComplianceOperatorProfileClusterEdge {
	if m == nil {
		return nil
	}
	cloned := new(ComplianceOperatorProfileClusterEdge)
	*cloned = *m

	return cloned
}

func init() {
	proto.RegisterEnum("storage.NodeRole", NodeRole_name, NodeRole_value)
	proto.RegisterEnum("storage.ScanType", ScanType_name, ScanType_value)
	proto.RegisterEnum("storage.RuleSeverity", RuleSeverity_name, RuleSeverity_value)
	proto.RegisterEnum("storage.ComplianceOperatorCheckResultV2_CheckStatus", ComplianceOperatorCheckResultV2_CheckStatus_name, ComplianceOperatorCheckResultV2_CheckStatus_value)
	proto.RegisterType((*ProfileShim)(nil), "storage.ProfileShim")
	proto.RegisterType((*ComplianceOperatorProfileV2)(nil), "storage.ComplianceOperatorProfileV2")
	proto.RegisterMapType((map[string]string)(nil), "storage.ComplianceOperatorProfileV2.AnnotationsEntry")
	proto.RegisterMapType((map[string]string)(nil), "storage.ComplianceOperatorProfileV2.LabelsEntry")
	proto.RegisterType((*ComplianceOperatorProfileV2_Rule)(nil), "storage.ComplianceOperatorProfileV2.Rule")
	proto.RegisterType((*ComplianceOperatorRuleV2)(nil), "storage.ComplianceOperatorRuleV2")
	proto.RegisterMapType((map[string]string)(nil), "storage.ComplianceOperatorRuleV2.AnnotationsEntry")
	proto.RegisterMapType((map[string]string)(nil), "storage.ComplianceOperatorRuleV2.LabelsEntry")
	proto.RegisterType((*ComplianceOperatorRuleV2_Fix)(nil), "storage.ComplianceOperatorRuleV2.Fix")
	proto.RegisterType((*RuleControls)(nil), "storage.RuleControls")
	proto.RegisterType((*ComplianceOperatorScanConfigurationV2)(nil), "storage.ComplianceOperatorScanConfigurationV2")
	proto.RegisterMapType((map[string]string)(nil), "storage.ComplianceOperatorScanConfigurationV2.AnnotationsEntry")
	proto.RegisterMapType((map[string]string)(nil), "storage.ComplianceOperatorScanConfigurationV2.LabelsEntry")
	proto.RegisterType((*ComplianceOperatorScanConfigurationV2_Cluster)(nil), "storage.ComplianceOperatorScanConfigurationV2.Cluster")
	proto.RegisterType((*ComplianceOperatorClusterScanConfigStatus)(nil), "storage.ComplianceOperatorClusterScanConfigStatus")
	proto.RegisterType((*ComplianceOperatorCheckResultV2)(nil), "storage.ComplianceOperatorCheckResultV2")
	proto.RegisterMapType((map[string]string)(nil), "storage.ComplianceOperatorCheckResultV2.AnnotationsEntry")
	proto.RegisterMapType((map[string]string)(nil), "storage.ComplianceOperatorCheckResultV2.LabelsEntry")
	proto.RegisterType((*ScanStatus)(nil), "storage.ScanStatus")
	proto.RegisterType((*ComplianceOperatorScanV2)(nil), "storage.ComplianceOperatorScanV2")
	proto.RegisterMapType((map[string]string)(nil), "storage.ComplianceOperatorScanV2.AnnotationsEntry")
	proto.RegisterMapType((map[string]string)(nil), "storage.ComplianceOperatorScanV2.LabelsEntry")
	proto.RegisterType((*ComplianceOperatorProfileClusterEdge)(nil), "storage.ComplianceOperatorProfileClusterEdge")
}

func init() {
	proto.RegisterFile("storage/compliance_operator_v2.proto", fileDescriptor_26c2ec1f62102154)
}

var fileDescriptor_26c2ec1f62102154 = []byte{
<<<<<<< HEAD
	// 2297 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xcc, 0x59, 0xcd, 0x6f, 0xdb, 0xc8,
	0x15, 0x8f, 0x6c, 0xd9, 0x96, 0x9e, 0x64, 0x9b, 0x99, 0x7c, 0x69, 0xdd, 0x20, 0x54, 0xd5, 0x6c,
	0x63, 0x27, 0xb1, 0x9c, 0xd5, 0x66, 0x83, 0xac, 0xdb, 0x74, 0x21, 0x3b, 0xca, 0x46, 0x88, 0x23,
	0xb9, 0x23, 0xd9, 0xc1, 0x6e, 0x0f, 0x04, 0x43, 0x8e, 0x65, 0xc2, 0x34, 0xa9, 0x90, 0x94, 0x37,
	0x2a, 0x7a, 0x28, 0xd0, 0xfe, 0x03, 0xbd, 0xf5, 0xda, 0x8f, 0x5b, 0x81, 0xfe, 0x1d, 0x3d, 0xf6,
	0x2f, 0x20, 0x8a, 0xf4, 0xd2, 0x5e, 0x09, 0x14, 0xbd, 0x16, 0xf3, 0x41, 0x8a, 0x94, 0x29, 0xd9,
	0x2e, 0x36, 0x9b, 0xbd, 0x69, 0xde, 0xcc, 0xfb, 0xcd, 0xe3, 0xcc, 0xfb, 0xbd, 0x8f, 0x11, 0xdc,
	0x76, 0x3d, 0xdb, 0x51, 0x7b, 0x64, 0x43, 0xb3, 0x8f, 0xfb, 0xa6, 0xa1, 0x5a, 0x1a, 0x51, 0xec,
	0x3e, 0x71, 0x54, 0xcf, 0x76, 0x94, 0x93, 0x5a, 0xb5, 0xef, 0xd8, 0x9e, 0x8d, 0x16, 0xc4, 0xaa,
	0x95, 0xab, 0x3d, 0xbb, 0x67, 0x33, 0xd9, 0x06, 0xfd, 0xc5, 0xa7, 0x57, 0xe4, 0x9e, 0x6d, 0xf7,
	0x4c, 0xb2, 0xc1, 0x46, 0xaf, 0x07, 0x07, 0x1b, 0x9e, 0x71, 0x4c, 0x5c, 0x4f, 0x3d, 0xee, 0x8b,
	0x05, 0xd7, 0xc3, 0x5d, 0x5c, 0xed, 0x90, 0xe8, 0x03, 0x93, 0x08, 0x39, 0x0a, 0xe5, 0x03, 0x97,
	0x38, 0x5c, 0x56, 0x39, 0x81, 0xc2, 0xae, 0x63, 0x1f, 0x18, 0x26, 0xe9, 0x1c, 0x1a, 0xc7, 0xa8,
	0x07, 0xd0, 0xe7, 0x43, 0xc5, 0xd0, 0x4b, 0x99, 0x72, 0x66, 0x35, 0xbf, 0xf5, 0x3c, 0xf0, 0xe5,
	0xa7, 0x2e, 0x51, 0x1d, 0xed, 0x70, 0xb3, 0xb2, 0x5e, 0x29, 0xbb, 0x6f, 0xcc, 0xcd, 0xca, 0xc1,
	0xd1, 0xea, 0x76, 0xf4, 0x0d, 0x6d, 0xf1, 0x09, 0x02, 0x69, 0xbf, 0xb6, 0x69, 0xe8, 0x6b, 0xf7,
	0x2d, 0x7b, 0xfd, 0xe0, 0x68, 0x5d, 0xb3, 0x2d, 0xd7, 0x73, 0x54, 0xc3, 0xf2, 0x2a, 0x38, 0x2f,
	0xb0, 0x9b, 0x7a, 0xe5, 0x5f, 0x39, 0xf8, 0xc1, 0x14, 0x00, 0x74, 0x13, 0x66, 0x22, 0x03, 0x8a,
	0x81, 0x2f, 0xe7, 0xd8, 0xae, 0xfd, 0xa3, 0x0a, 0x9e, 0x31, 0x74, 0xf4, 0x3c, 0x61, 0xe6, 0x0c,
	0x5b, 0xb5, 0x16, 0xf8, 0xf2, 0xc7, 0xa1, 0x99, 0x23, 0xe8, 0xb2, 0xc0, 0x2c, 0x37, 0x9f, 0xde,
	0x3f, 0x34, 0x74, 0x9d, 0x58, 0x71, 0x3b, 0xd0, 0x10, 0xb2, 0x96, 0x7a, 0x4c, 0x4a, 0xb3, 0x0c,
	0x83, 0x04, 0xbe, 0xac, 0x4e, 0xc1, 0x68, 0xa9, 0xc7, 0x24, 0x44, 0xe1, 0xe7, 0x60, 0x58, 0x3a,
	0x79, 0xfb, 0x44, 0x53, 0x3d, 0xd2, 0xb3, 0x9d, 0xe1, 0xe6, 0xc0, 0x32, 0xde, 0x0c, 0xc8, 0x4f,
	0x28, 0xe4, 0x66, 0x68, 0x21, 0x97, 0x29, 0x86, 0xa5, 0x1b, 0x1a, 0xfd, 0xc8, 0x0a, 0x66, 0x5b,
	0xa2, 0xdf, 0x65, 0x60, 0x39, 0x5c, 0x73, 0x42, 0x1c, 0xd7, 0xb0, 0xad, 0x52, 0x96, 0x99, 0x71,
	0x18, 0xf8, 0xb2, 0x3e, 0xc5, 0x8c, 0x7d, 0xbe, 0xfa, 0x5b, 0xb0, 0x64, 0x49, 0x4c, 0x09, 0x44,
	0xb4, 0x0f, 0xc5, 0xbe, 0x63, 0xeb, 0x03, 0xcd, 0x53, 0xbc, 0x61, 0x9f, 0x94, 0xe6, 0x98, 0x3d,
	0x9f, 0x06, 0xbe, 0xbc, 0x31, 0xc5, 0x9e, 0x5d, 0xae, 0x52, 0xee, 0x0e, 0xfb, 0xd1, 0xf1, 0xe0,
	0x82, 0x00, 0xa2, 0x42, 0xb4, 0x0d, 0x39, 0xd7, 0x53, 0x2d, 0x5d, 0x75, 0xf4, 0xd2, 0x3c, 0xc3,
	0xbc, 0x13, 0xf8, 0xf2, 0x8f, 0x52, 0x30, 0x3b, 0x62, 0x59, 0x84, 0x13, 0x29, 0xa2, 0xe7, 0x30,
	0x6f, 0xaa, 0xaf, 0x89, 0xe9, 0x96, 0x16, 0xca, 0xb3, 0xab, 0x85, 0xda, 0x83, 0xaa, 0x70, 0xe8,
	0xea, 0x14, 0x4f, 0xaa, 0xee, 0x30, 0x95, 0x86, 0xe5, 0x39, 0x43, 0x2c, 0xf4, 0xd1, 0x2b, 0x28,
	0xa8, 0x96, 0x65, 0x7b, 0xaa, 0x67, 0xd8, 0x96, 0x5b, 0xca, 0x31, 0xb8, 0xcf, 0xce, 0x05, 0x57,
	0x1f, 0xe9, 0x71, 0xcc, 0x38, 0x12, 0x2a, 0x43, 0x41, 0x27, 0xae, 0xe6, 0x18, 0x7d, 0x3a, 0x2e,
	0xe5, 0xe9, 0xa7, 0xe2, 0xb8, 0x08, 0x7d, 0x01, 0x73, 0xce, 0xc0, 0x24, 0x6e, 0x09, 0xd8, 0xa6,
	0x6b, 0xe7, 0xda, 0x14, 0x0f, 0x4c, 0x82, 0xb9, 0x1e, 0x2a, 0xc1, 0x82, 0x38, 0xd9, 0x52, 0x81,
	0xc1, 0x87, 0x43, 0x74, 0x15, 0xe6, 0x3c, 0xc3, 0x33, 0x49, 0xa9, 0xc8, 0xe4, 0x7c, 0x80, 0xae,
	0xc3, 0xfc, 0x89, 0x6a, 0x0e, 0x88, 0x5b, 0x5a, 0x2c, 0xcf, 0xae, 0xe6, 0xb1, 0x18, 0xad, 0x7c,
	0x0e, 0x85, 0xd8, 0xd1, 0x20, 0x09, 0x66, 0x8f, 0xc8, 0x90, 0x33, 0x0e, 0xd3, 0x9f, 0x14, 0x8e,
	0x2d, 0xe5, 0xfc, 0xc2, 0x7c, 0xb0, 0x39, 0xf3, 0x38, 0xb3, 0xf2, 0x33, 0x90, 0xc6, 0x8f, 0xe1,
	0x42, 0xfa, 0x26, 0x64, 0xe9, 0x17, 0x21, 0x1d, 0xf2, 0xf4, 0x9b, 0x14, 0xc6, 0x40, 0xce, 0xf5,
	0x2f, 0x03, 0x5f, 0xde, 0x3e, 0x57, 0xb0, 0xa1, 0x08, 0xfb, 0xb5, 0x4d, 0xaa, 0x9b, 0x16, 0x6b,
	0x72, 0x14, 0x99, 0xf2, 0xb5, 0xf2, 0x9f, 0x79, 0x28, 0x4d, 0x52, 0x47, 0x0d, 0xc1, 0x7f, 0xbe,
	0xfb, 0x27, 0x81, 0x2f, 0xaf, 0xa7, 0x38, 0x25, 0x5d, 0x9c, 0x42, 0x7e, 0x1a, 0x8e, 0x38, 0x97,
	0xbf, 0x06, 0x69, 0x14, 0xc7, 0x05, 0x97, 0x79, 0x58, 0xda, 0x08, 0x7c, 0xf9, 0x5e, 0x0a, 0x64,
	0x68, 0xc7, 0x38, 0x99, 0xf1, 0x72, 0x08, 0x14, 0x72, 0xb2, 0x05, 0x45, 0x76, 0x4a, 0x21, 0x2e,
	0x0f, 0x55, 0xf7, 0x02, 0x5f, 0xbe, 0x33, 0xc9, 0xd4, 0x71, 0xcc, 0x02, 0x05, 0x08, 0xf1, 0x1a,
	0xe2, 0xd4, 0x19, 0xc1, 0x79, 0xc0, 0x59, 0x0d, 0x7c, 0xf9, 0xf6, 0x24, 0xb0, 0x04, 0xab, 0xd9,
	0xb1, 0x32, 0x4a, 0x2b, 0x90, 0x73, 0xc9, 0x09, 0x71, 0x0c, 0x6f, 0xc8, 0xc2, 0xc4, 0x52, 0xed,
	0x5a, 0xe4, 0xcb, 0x54, 0xaf, 0x23, 0x26, 0xb7, 0xee, 0x07, 0xbe, 0xbc, 0x3a, 0x09, 0x3c, 0x5c,
	0x15, 0xa3, 0xbb, 0x90, 0xa0, 0x46, 0x44, 0xf7, 0x79, 0x46, 0x95, 0xf5, 0x29, 0x54, 0xe1, 0xb7,
	0x99, 0xca, 0xf5, 0x6e, 0x92, 0xeb, 0x3c, 0x74, 0xd4, 0xce, 0xc6, 0x9a, 0x4e, 0xf4, 0x88, 0x6b,
	0xb9, 0x38, 0xd7, 0xce, 0xa6, 0xff, 0x4d, 0xc8, 0x3b, 0x0c, 0x42, 0x35, 0x49, 0x09, 0xd8, 0xfc,
	0x48, 0x40, 0x51, 0x0f, 0x8c, 0xb7, 0xc4, 0x15, 0xcc, 0xe6, 0x03, 0x74, 0x0b, 0x80, 0x73, 0x76,
	0xcf, 0x25, 0x7a, 0xa9, 0xc8, 0x58, 0x1c, 0x93, 0xa0, 0x15, 0xc8, 0x7d, 0xa3, 0x3a, 0x96, 0x61,
	0xf5, 0x42, 0x8e, 0x47, 0xe3, 0x0f, 0xc8, 0xf2, 0xca, 0x5f, 0x01, 0x3e, 0x3e, 0x7d, 0xba, 0x1d,
	0x4d, 0xb5, 0xb6, 0x6d, 0xeb, 0xc0, 0xe8, 0x0d, 0xf8, 0x77, 0xef, 0xd7, 0xd0, 0xcf, 0x63, 0xc9,
	0xbe, 0x1e, 0xf8, 0xf2, 0x93, 0xb4, 0xbc, 0xa0, 0xa9, 0x56, 0x99, 0x2b, 0xc6, 0x52, 0x79, 0xc8,
	0xc3, 0xfb, 0xd4, 0x99, 0x57, 0x07, 0x03, 0x43, 0x5f, 0xe3, 0x15, 0x82, 0x02, 0x92, 0xab, 0xa9,
	0x96, 0xa2, 0x31, 0x0d, 0x1e, 0x61, 0x38, 0x21, 0x3f, 0x0b, 0x7c, 0xf9, 0x93, 0x33, 0x36, 0x60,
	0x71, 0x83, 0x63, 0xf3, 0x9c, 0x59, 0xc1, 0x4b, 0x6e, 0x64, 0x38, 0x9d, 0x45, 0x8f, 0xe0, 0x86,
	0x3a, 0xf0, 0x6c, 0x45, 0xed, 0xf7, 0xcd, 0xa1, 0xe2, 0x90, 0x63, 0xa2, 0x1b, 0xc2, 0xc5, 0x28,
	0x41, 0x73, 0xf8, 0x1a, 0x9d, 0xae, 0xd3, 0x59, 0x1c, 0x9b, 0x44, 0x8f, 0xa1, 0xc4, 0xf4, 0x06,
	0x7d, 0x5d, 0xf5, 0x48, 0x52, 0x31, 0xcb, 0x14, 0xaf, 0xd3, 0xf9, 0x3d, 0x36, 0x9d, 0xd0, 0xac,
	0xc0, 0xa2, 0x6d, 0x11, 0x85, 0x56, 0x7b, 0x0a, 0x35, 0x86, 0xb1, 0x2e, 0x87, 0x0b, 0xb6, 0x45,
	0xba, 0xc6, 0x31, 0xa1, 0xf6, 0x23, 0x3c, 0xc6, 0x99, 0xcd, 0x29, 0x7e, 0x9e, 0x72, 0x13, 0xa9,
	0x04, 0x52, 0xd3, 0x08, 0xf4, 0xc5, 0x05, 0x81, 0xa7, 0xb3, 0xe9, 0x01, 0xe4, 0x44, 0x21, 0x12,
	0x26, 0xe3, 0xab, 0x11, 0x7e, 0xac, 0x3c, 0xc5, 0xd1, 0x2a, 0xf4, 0x00, 0xc0, 0xb2, 0x75, 0xa2,
	0x38, 0x36, 0xd5, 0xc9, 0x97, 0x67, 0x57, 0x97, 0x6a, 0x97, 0x23, 0x9d, 0x96, 0xad, 0x13, 0x6c,
	0x9b, 0x04, 0xe7, 0x2d, 0xf1, 0xcb, 0x45, 0xab, 0x20, 0xb9, 0x9e, 0x63, 0x68, 0x9e, 0xc2, 0x14,
	0xd9, 0x09, 0x02, 0x3b, 0xc1, 0x25, 0x2e, 0xa7, 0x4a, 0xec, 0x10, 0xd7, 0x21, 0x17, 0x56, 0xce,
	0x8c, 0x88, 0x85, 0x18, 0x72, 0x47, 0x4c, 0xe0, 0x68, 0x09, 0x7a, 0x02, 0x45, 0xcd, 0x21, 0xaa,
	0x47, 0x74, 0x76, 0x37, 0x2c, 0xfb, 0x16, 0x6a, 0x2b, 0x55, 0x5e, 0xa6, 0x57, 0xc3, 0x32, 0xbd,
	0xda, 0x0d, 0xcb, 0x74, 0x5c, 0x10, 0xeb, 0xa9, 0x04, 0x3d, 0x83, 0xcb, 0xa6, 0xea, 0x7a, 0xc2,
	0x21, 0x04, 0xc6, 0xe2, 0x99, 0x18, 0xcb, 0x54, 0x89, 0x7b, 0x09, 0xc7, 0xe9, 0x42, 0xe1, 0xd8,
	0xd6, 0x8d, 0x03, 0x83, 0xe8, 0xca, 0xeb, 0x61, 0x69, 0x69, 0xdc, 0x70, 0xd3, 0x38, 0xde, 0x73,
	0x89, 0xb3, 0x55, 0x0e, 0x7c, 0xf9, 0x26, 0xaf, 0x18, 0x7b, 0x96, 0xed, 0x10, 0x85, 0x5f, 0xf3,
	0x2a, 0x9d, 0x2c, 0x37, 0x9f, 0xae, 0x55, 0x30, 0x84, 0x38, 0x5b, 0xc3, 0xf1, 0x88, 0xb6, 0x7c,
	0x3a, 0xa2, 0x61, 0xc8, 0x69, 0xe6, 0xc0, 0xf5, 0x88, 0xe3, 0x96, 0x24, 0x76, 0x77, 0x8f, 0x2e,
	0xe8, 0x1b, 0xdb, 0x5c, 0x1d, 0x47, 0x38, 0x1f, 0xb2, 0x36, 0x19, 0xc2, 0x82, 0xb0, 0x07, 0x59,
	0x00, 0xc2, 0xa2, 0x51, 0x33, 0xd4, 0x0e, 0x7c, 0xf9, 0x45, 0x14, 0x3d, 0xf8, 0xec, 0xa9, 0x78,
	0x44, 0xeb, 0x15, 0x3e, 0x95, 0xda, 0x08, 0x25, 0x82, 0x55, 0x5e, 0x6c, 0xd1, 0xd4, 0x2b, 0x7f,
	0xc9, 0xc2, 0xda, 0xe9, 0x13, 0x13, 0x48, 0xa3, 0x83, 0xeb, 0x78, 0xaa, 0x37, 0x70, 0xd1, 0x1d,
	0x16, 0x34, 0x79, 0x31, 0x7d, 0x23, 0xf0, 0xe5, 0x2b, 0x93, 0x42, 0xe1, 0x77, 0xfc, 0x19, 0xe8,
	0x8f, 0x19, 0x58, 0x8a, 0xc7, 0xde, 0xa8, 0x43, 0xfb, 0x55, 0xe0, 0xcb, 0x6f, 0x2f, 0x18, 0xda,
	0x53, 0x4b, 0xbf, 0x14, 0x27, 0x3a, 0xdb, 0xc2, 0xe2, 0x28, 0x80, 0x37, 0x75, 0x5a, 0x15, 0x13,
	0xc7, 0xb1, 0x1d, 0x1a, 0xad, 0x59, 0x55, 0xcc, 0x47, 0xe8, 0xb7, 0x99, 0x34, 0x3a, 0x66, 0xcf,
	0xa2, 0xe3, 0xd6, 0x4f, 0x03, 0x5f, 0x7e, 0x7c, 0xc6, 0xb7, 0xed, 0xa8, 0xae, 0x57, 0x16, 0x94,
	0x2d, 0x53, 0xd5, 0x51, 0xcd, 0x37, 0x4e, 0xe6, 0x1f, 0x42, 0x31, 0xbc, 0x33, 0x96, 0xba, 0xe6,
	0x38, 0xef, 0x84, 0x8c, 0xa5, 0xa7, 0x3f, 0x2c, 0x82, 0x9c, 0xe2, 0x2d, 0x87, 0x44, 0x3b, 0xc2,
	0xc4, 0x1d, 0x98, 0xde, 0x7e, 0x0d, 0xd5, 0x63, 0x89, 0x75, 0x52, 0x6d, 0xcb, 0x34, 0xca, 0x7b,
	0xa7, 0x73, 0x2a, 0xf7, 0x9e, 0x2d, 0xc8, 0x69, 0x74, 0xc5, 0xe8, 0x1a, 0x27, 0x75, 0x6e, 0x1c,
	0x28, 0xd6, 0x66, 0x2f, 0x30, 0xc5, 0x26, 0x6b, 0xd7, 0x39, 0x46, 0xac, 0xd5, 0x9e, 0xd4, 0xae,
	0x73, 0x94, 0x78, 0xad, 0x8d, 0xf3, 0x4c, 0x99, 0x65, 0xdd, 0xa4, 0x2f, 0x67, 0xdf, 0xbb, 0x2f,
	0x8f, 0xdf, 0xc3, 0xf2, 0xa9, 0x7b, 0x40, 0x43, 0x98, 0x77, 0x19, 0x23, 0x45, 0x15, 0xfc, 0x70,
	0x4a, 0xf4, 0x4b, 0xdc, 0x4e, 0x95, 0x8d, 0x38, 0x9b, 0x27, 0x96, 0xf3, 0xfc, 0x38, 0xf8, 0xa2,
	0xe8, 0x40, 0xc4, 0x86, 0x89, 0x12, 0x7c, 0xfe, 0x7d, 0x94, 0xe0, 0x63, 0xd1, 0x7f, 0xe1, 0x74,
	0xf4, 0xaf, 0x40, 0xd1, 0xa0, 0x87, 0x38, 0xd0, 0xc2, 0x56, 0x9a, 0x2e, 0x49, 0xc8, 0xd0, 0x4e,
	0x54, 0x94, 0xe4, 0x59, 0x7e, 0x38, 0xff, 0x09, 0xa5, 0x95, 0x23, 0xbf, 0x48, 0x96, 0x23, 0xbc,
	0x8d, 0xfe, 0xfc, 0xdc, 0x90, 0xd3, 0x0b, 0x91, 0x6f, 0xc6, 0x72, 0x79, 0xe1, 0x4c, 0xe2, 0x3f,
	0x0e, 0x7c, 0xf9, 0xe1, 0xc4, 0x8b, 0xe3, 0xdb, 0x96, 0xb7, 0x39, 0x66, 0x92, 0xf4, 0x89, 0x2a,
	0x20, 0xfe, 0x40, 0x52, 0xfc, 0x7f, 0x1f, 0x48, 0x4a, 0xb0, 0xa0, 0xd9, 0x96, 0xe7, 0xd8, 0x26,
	0x2b, 0x20, 0xf2, 0x38, 0x1c, 0xa2, 0xdf, 0x64, 0x20, 0xcf, 0x62, 0x32, 0xf3, 0xe2, 0x25, 0xb6,
	0xc1, 0x41, 0xe0, 0xcb, 0xaf, 0x27, 0x85, 0xac, 0xd3, 0xcd, 0xee, 0xc4, 0x48, 0xbc, 0x5f, 0xdb,
	0x8c, 0xa0, 0x53, 0x3b, 0x71, 0x3a, 0xcb, 0xa8, 0xf2, 0xe7, 0x4c, 0x4a, 0x55, 0x2e, 0x31, 0x63,
	0x7e, 0x19, 0xf8, 0xf2, 0xc9, 0xf9, 0xab, 0xf2, 0x73, 0xa7, 0x85, 0xf1, 0xed, 0xd2, 0x0c, 0x1c,
	0x2f, 0xed, 0x13, 0x3d, 0xda, 0xe5, 0xf1, 0x1e, 0x2d, 0xd9, 0x8d, 0xa1, 0xa9, 0xdd, 0xd8, 0x95,
	0xef, 0x4f, 0x37, 0xe6, 0x42, 0x21, 0x16, 0x6f, 0x50, 0x1e, 0xe6, 0xf6, 0x5a, 0x9d, 0x46, 0x57,
	0xba, 0x84, 0x72, 0x90, 0xdd, 0xad, 0x77, 0x3a, 0x52, 0x86, 0xfe, 0x7a, 0x56, 0x6f, 0xee, 0x48,
	0x33, 0x74, 0xba, 0x81, 0x71, 0x1b, 0x4b, 0xb3, 0x54, 0xd8, 0x6c, 0x3d, 0x6b, 0x4b, 0x59, 0x04,
	0x30, 0xff, 0xb2, 0xde, 0xda, 0xab, 0xef, 0x48, 0x73, 0x08, 0xc1, 0x52, 0xab, 0xdd, 0x55, 0xea,
	0xbb, 0xbb, 0x3b, 0xcd, 0xed, 0xfa, 0xd6, 0x4e, 0x43, 0x9a, 0x47, 0x12, 0x14, 0x9b, 0xad, 0xed,
	0x76, 0xab, 0xd3, 0xec, 0x74, 0x1b, 0xad, 0xae, 0xb4, 0x50, 0xd9, 0x07, 0xa0, 0x97, 0x22, 0xf6,
	0xbc, 0x0a, 0x73, 0xfd, 0x43, 0xd5, 0x15, 0x8f, 0x2d, 0x98, 0x0f, 0x68, 0x26, 0x76, 0x18, 0x4b,
	0x84, 0xcd, 0x62, 0x94, 0x38, 0x47, 0x96, 0x32, 0x46, 0xe7, 0x58, 0xf9, 0x77, 0x2e, 0xed, 0x49,
	0x87, 0x3b, 0xe3, 0x19, 0x4f, 0xc7, 0xa9, 0x3e, 0x38, 0xf3, 0xbd, 0xf3, 0xc1, 0x46, 0x9c, 0xaf,
	0x8b, 0x53, 0x1f, 0x69, 0x4e, 0xf1, 0x35, 0xc6, 0xb8, 0x64, 0xbe, 0x9c, 0x7d, 0xef, 0xf9, 0x72,
	0x54, 0x56, 0x65, 0xf9, 0x65, 0x8a, 0xb2, 0x2a, 0x7e, 0x99, 0x4b, 0xc9, 0xcb, 0x44, 0x55, 0xf6,
	0xa0, 0x49, 0xdb, 0x3a, 0x96, 0x41, 0x27, 0xf5, 0x7e, 0xe1, 0xa2, 0x0b, 0xbd, 0x0b, 0x71, 0x97,
	0xf8, 0x76, 0xde, 0x85, 0x04, 0xd6, 0xf4, 0x04, 0x52, 0x15, 0xf7, 0xc6, 0x1e, 0xd7, 0x72, 0x2c,
	0x27, 0xc7, 0x9b, 0x47, 0xd5, 0xea, 0x0e, 0xfb, 0x84, 0x5f, 0x10, 0x7b, 0x45, 0x7b, 0x04, 0x8b,
	0xbc, 0x1d, 0x25, 0x26, 0xd1, 0x3c, 0xdb, 0x61, 0x6f, 0x46, 0xa9, 0xad, 0x6c, 0x91, 0xae, 0xeb,
	0x88, 0x65, 0xe8, 0x5e, 0x54, 0x75, 0x00, 0x3b, 0xb3, 0x2b, 0x89, 0x4d, 0x38, 0xe1, 0xa2, 0x3a,
	0xe1, 0xc9, 0x45, 0xb3, 0x5a, 0x32, 0x37, 0xfd, 0x3a, 0x03, 0x88, 0xd5, 0xc4, 0xe4, 0x2d, 0xd1,
	0x06, 0xe7, 0xef, 0x73, 0xb7, 0x1e, 0x05, 0xbe, 0x5c, 0x9b, 0xe4, 0xb1, 0xac, 0x1a, 0x6e, 0x08,
	0xc4, 0x64, 0x66, 0x94, 0xe8, 0x6e, 0xe1, 0x14, 0x9d, 0xf9, 0x90, 0x81, 0xf3, 0xbf, 0x33, 0x70,
	0x7b, 0xe2, 0xdb, 0xbc, 0xe0, 0x48, 0x43, 0xef, 0x11, 0x24, 0xc7, 0xe2, 0xce, 0x72, 0xe0, 0xcb,
	0x85, 0xb0, 0x21, 0x33, 0x74, 0x1e, 0x7a, 0xde, 0xa4, 0xfc, 0x6b, 0x85, 0x03, 0x5f, 0x6e, 0x9d,
	0xeb, 0x5f, 0xab, 0xf3, 0xfe, 0xef, 0x96, 0xf8, 0x7b, 0xeb, 0x05, 0x14, 0xa2, 0x3f, 0x7f, 0xa2,
	0x00, 0x70, 0x37, 0xf0, 0xe5, 0x1f, 0x4f, 0xd9, 0x33, 0xd6, 0x0b, 0xe0, 0xd0, 0xe2, 0xbd, 0x53,
	0x8d, 0xe4, 0x7b, 0x2f, 0xbe, 0xef, 0xae, 0x43, 0x2e, 0xf4, 0x7e, 0x9a, 0x90, 0x9a, 0xad, 0x67,
	0xb8, 0x2e, 0x5d, 0xa2, 0x69, 0xe8, 0x55, 0x1b, 0xbf, 0x68, 0x60, 0x29, 0xc3, 0x53, 0x52, 0xa7,
	0xdb, 0xc0, 0xd2, 0xcc, 0xdd, 0x3a, 0xe4, 0x42, 0x82, 0xa1, 0x2b, 0xb0, 0xcc, 0xd2, 0x9b, 0xd2,
	0xd9, 0xae, 0xb7, 0x94, 0xee, 0x57, 0xbb, 0x0d, 0xe9, 0x12, 0x5a, 0x84, 0x7c, 0xab, 0xfd, 0xb4,
	0xc1, 0x64, 0x52, 0x06, 0x5d, 0x86, 0xc5, 0xdd, 0x9d, 0x7a, 0xf7, 0x59, 0x1b, 0xbf, 0xe4, 0xa2,
	0x99, 0xbb, 0x7f, 0xca, 0x40, 0x31, 0x5e, 0x38, 0xa3, 0x1b, 0x70, 0x85, 0xe3, 0xe0, 0xbd, 0x9d,
	0x86, 0xd2, 0x69, 0xec, 0x37, 0x70, 0xb3, 0xfb, 0x95, 0x74, 0x09, 0x7d, 0x04, 0xd7, 0xf6, 0x5a,
	0x2f, 0x5a, 0xed, 0x57, 0xad, 0xb1, 0xa9, 0x0c, 0xba, 0x0e, 0x88, 0x26, 0xcc, 0x31, 0xf9, 0x0c,
	0xba, 0x06, 0x97, 0x77, 0xda, 0xaf, 0xc6, 0xc4, 0xb3, 0xa8, 0x04, 0x57, 0x5f, 0x36, 0x9e, 0x36,
	0xf7, 0x5e, 0x8e, 0xcd, 0x64, 0x29, 0xd0, 0xf3, 0xe6, 0x97, 0xcf, 0xc7, 0xe4, 0x73, 0x5b, 0x0f,
	0xff, 0xf6, 0xee, 0x56, 0xe6, 0xef, 0xef, 0x6e, 0x65, 0xfe, 0xf1, 0xee, 0x56, 0xe6, 0xf7, 0xff,
	0xbc, 0x75, 0x09, 0x3e, 0x32, 0xec, 0xaa, 0xeb, 0xa9, 0xda, 0x91, 0x63, 0xbf, 0xe5, 0x44, 0x0c,
	0xc3, 0xc1, 0xd7, 0xe1, 0x9f, 0xc9, 0xaf, 0xe7, 0x99, 0xfc, 0xd3, 0xff, 0x05, 0x00, 0x00, 0xff,
	0xff, 0xfe, 0xf3, 0xdc, 0xd7, 0x84, 0x1e, 0x00, 0x00,
=======
	// 2339 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xcc, 0x59, 0xcd, 0x72, 0xdb, 0xc8,
	0x11, 0x36, 0x25, 0x4a, 0x22, 0x9b, 0x94, 0x04, 0x8f, 0xff, 0xb0, 0x8a, 0x4b, 0x60, 0x18, 0x3b,
	0x96, 0xbc, 0x16, 0xed, 0xe5, 0x7a, 0x5d, 0x5e, 0xed, 0x3a, 0x5b, 0x94, 0x4c, 0xad, 0x59, 0x96,
	0x49, 0x65, 0x48, 0xc9, 0xb5, 0xc9, 0x81, 0x05, 0x03, 0x23, 0x0a, 0x25, 0x10, 0xa0, 0x01, 0x50,
	0x2b, 0xa6, 0x72, 0x48, 0x55, 0xf2, 0x02, 0xb9, 0xe5, 0x9c, 0xe4, 0x96, 0xaa, 0x3c, 0x47, 0x8e,
	0x79, 0x02, 0x54, 0xca, 0xb9, 0x24, 0x57, 0x5c, 0x72, 0xc9, 0x21, 0x35, 0x3f, 0x80, 0x00, 0x8a,
	0x3f, 0xd2, 0xd6, 0x3a, 0xde, 0x1b, 0xa7, 0x67, 0xfa, 0x9b, 0xc6, 0x4c, 0xf7, 0xd7, 0xdd, 0x43,
	0xb8, 0xe3, 0x7a, 0xb6, 0xa3, 0x76, 0xc8, 0x43, 0xcd, 0xee, 0xf6, 0x4c, 0x43, 0xb5, 0x34, 0xd2,
	0xb6, 0x7b, 0xc4, 0x51, 0x3d, 0xdb, 0x69, 0x9f, 0x94, 0x4b, 0x3d, 0xc7, 0xf6, 0x6c, 0xb4, 0x20,
	0x56, 0xad, 0x5c, 0xef, 0xd8, 0x1d, 0x9b, 0xc9, 0x1e, 0xd2, 0x5f, 0x7c, 0x7a, 0x45, 0xe9, 0xd8,
	0x76, 0xc7, 0x24, 0x0f, 0xd9, 0xe8, 0x4d, 0xff, 0xf0, 0xa1, 0x67, 0x74, 0x89, 0xeb, 0xa9, 0xdd,
	0x9e, 0x58, 0x70, 0x33, 0xdc, 0xc5, 0xd5, 0x8e, 0x88, 0xde, 0x37, 0x89, 0x90, 0xa3, 0x50, 0xde,
	0x77, 0x89, 0xc3, 0x65, 0xc5, 0x13, 0xc8, 0xed, 0x39, 0xf6, 0xa1, 0x61, 0x92, 0xe6, 0x91, 0xd1,
	0x45, 0x1d, 0x80, 0x1e, 0x1f, 0xb6, 0x0d, 0x5d, 0x4e, 0x15, 0x52, 0x6b, 0xd9, 0xad, 0x17, 0x81,
	0xaf, 0x3c, 0x77, 0x89, 0xea, 0x68, 0x47, 0x9b, 0xc5, 0x8d, 0x62, 0xc1, 0x7d, 0x6b, 0x6e, 0x16,
	0x0f, 0x8f, 0xd7, 0xb6, 0xa3, 0x6f, 0x68, 0x88, 0x4f, 0x10, 0x48, 0x07, 0xe5, 0x4d, 0x43, 0x5f,
	0x7f, 0x60, 0xd9, 0x1b, 0x87, 0xc7, 0x1b, 0x9a, 0x6d, 0xb9, 0x9e, 0xa3, 0x1a, 0x96, 0x57, 0xc4,
	0x59, 0x81, 0x5d, 0xd3, 0x8b, 0xff, 0xca, 0xc0, 0x8f, 0x26, 0x00, 0xa0, 0xdb, 0x30, 0x13, 0x19,
	0x90, 0x0f, 0x7c, 0x25, 0xc3, 0x76, 0xed, 0x1d, 0x17, 0xf1, 0x8c, 0xa1, 0xa3, 0x17, 0x09, 0x33,
	0x67, 0xd8, 0xaa, 0xf5, 0xc0, 0x57, 0xee, 0x86, 0x66, 0x9e, 0x41, 0x17, 0x04, 0x66, 0xa1, 0xf6,
	0xfc, 0xc1, 0x91, 0xa1, 0xeb, 0xc4, 0x8a, 0xdb, 0x81, 0x06, 0x90, 0xb6, 0xd4, 0x2e, 0x91, 0x67,
	0x19, 0x06, 0x09, 0x7c, 0x45, 0x9d, 0x80, 0x51, 0x57, 0xbb, 0x24, 0x44, 0xe1, 0xe7, 0x60, 0x58,
	0x3a, 0x39, 0x7d, 0xa6, 0xa9, 0x1e, 0xe9, 0xd8, 0xce, 0x60, 0xb3, 0x6f, 0x19, 0x6f, 0xfb, 0xe4,
	0x0b, 0x0a, 0xb9, 0x19, 0x5a, 0xc8, 0x65, 0x6d, 0xc3, 0xd2, 0x0d, 0x8d, 0x7e, 0x64, 0x11, 0xb3,
	0x2d, 0xd1, 0xef, 0x53, 0xb0, 0x1c, 0xae, 0x39, 0x21, 0x8e, 0x6b, 0xd8, 0x96, 0x9c, 0x66, 0x66,
	0x1c, 0x05, 0xbe, 0xa2, 0x4f, 0x30, 0xe3, 0x80, 0xaf, 0xfe, 0x1e, 0x2c, 0x59, 0x12, 0x53, 0x02,
	0x11, 0x1d, 0x40, 0xbe, 0xe7, 0xd8, 0x7a, 0x5f, 0xf3, 0xda, 0xde, 0xa0, 0x47, 0xe4, 0x39, 0x66,
	0xcf, 0xa7, 0x81, 0xaf, 0x3c, 0x9c, 0x60, 0xcf, 0x1e, 0x57, 0x29, 0xb4, 0x06, 0xbd, 0xe8, 0x78,
	0x70, 0x4e, 0x00, 0x51, 0x21, 0xda, 0x86, 0x8c, 0xeb, 0xa9, 0x96, 0xae, 0x3a, 0xba, 0x3c, 0xcf,
	0x30, 0xef, 0x05, 0xbe, 0xf2, 0x93, 0x11, 0x98, 0x4d, 0xb1, 0x2c, 0xc2, 0x89, 0x14, 0xd1, 0x0b,
	0x98, 0x37, 0xd5, 0x37, 0xc4, 0x74, 0xe5, 0x85, 0xc2, 0xec, 0x5a, 0xae, 0xfc, 0xa8, 0x24, 0x1c,
	0xba, 0x34, 0xc1, 0x93, 0x4a, 0xbb, 0x4c, 0xa5, 0x6a, 0x79, 0xce, 0x00, 0x0b, 0x7d, 0xf4, 0x1a,
	0x72, 0xaa, 0x65, 0xd9, 0x9e, 0xea, 0x19, 0xb6, 0xe5, 0xca, 0x19, 0x06, 0xf7, 0xd9, 0x85, 0xe0,
	0x2a, 0x67, 0x7a, 0x1c, 0x33, 0x8e, 0x84, 0x0a, 0x90, 0xd3, 0x89, 0xab, 0x39, 0x46, 0x8f, 0x8e,
	0xe5, 0x2c, 0xfd, 0x54, 0x1c, 0x17, 0xa1, 0xaf, 0x60, 0xce, 0xe9, 0x9b, 0xc4, 0x95, 0x81, 0x6d,
	0xba, 0x7e, 0xa1, 0x4d, 0x71, 0xdf, 0x24, 0x98, 0xeb, 0x21, 0x19, 0x16, 0xc4, 0xc9, 0xca, 0x39,
	0x06, 0x1f, 0x0e, 0xd1, 0x75, 0x98, 0xf3, 0x0c, 0xcf, 0x24, 0x72, 0x9e, 0xc9, 0xf9, 0x00, 0xdd,
	0x84, 0xf9, 0x13, 0xd5, 0xec, 0x13, 0x57, 0x5e, 0x2c, 0xcc, 0xae, 0x65, 0xb1, 0x18, 0xad, 0x7c,
	0x0e, 0xb9, 0xd8, 0xd1, 0x20, 0x09, 0x66, 0x8f, 0xc9, 0x80, 0x47, 0x1c, 0xa6, 0x3f, 0x29, 0x1c,
	0x5b, 0xca, 0xe3, 0x0b, 0xf3, 0xc1, 0xe6, 0xcc, 0xd3, 0xd4, 0xca, 0xcf, 0x40, 0x1a, 0x3e, 0x86,
	0x4b, 0xe9, 0x9b, 0x90, 0xa6, 0x5f, 0x84, 0x74, 0xc8, 0xd2, 0x6f, 0x6a, 0xb3, 0x08, 0xe4, 0xb1,
	0xfe, 0x75, 0xe0, 0x2b, 0xdb, 0x17, 0x22, 0x1b, 0x8a, 0x70, 0x50, 0xde, 0xa4, 0xba, 0xa3, 0xb8,
	0x26, 0x43, 0x91, 0x69, 0xbc, 0x16, 0xdf, 0x2d, 0x80, 0x3c, 0x4e, 0x7d, 0x0a, 0xcf, 0xdc, 0x82,
	0x05, 0x66, 0x60, 0x48, 0x32, 0x78, 0x9e, 0x0e, 0x6b, 0x3a, 0xfa, 0x32, 0x41, 0x1b, 0x6b, 0x81,
	0xaf, 0xdc, 0x19, 0xe1, 0xcb, 0x74, 0x8f, 0x04, 0x67, 0x88, 0xc8, 0xaf, 0x8a, 0xef, 0x66, 0x21,
	0x96, 0x9e, 0x0e, 0x91, 0x88, 0x2b, 0xf6, 0x61, 0x2c, 0xa8, 0xda, 0x90, 0x71, 0xc9, 0x09, 0x71,
	0x0c, 0x6f, 0xc0, 0x02, 0x75, 0xa9, 0x7c, 0x23, 0xf2, 0x26, 0xaa, 0xd7, 0x14, 0x93, 0x5b, 0x0f,
	0x02, 0x5f, 0x59, 0x1b, 0x07, 0x1e, 0xae, 0x8a, 0x05, 0x9c, 0x90, 0xa0, 0x6a, 0x14, 0x70, 0xf3,
	0xcc, 0x59, 0x37, 0x26, 0x38, 0x2b, 0x3f, 0xcf, 0x91, 0xd1, 0xd6, 0x4a, 0x46, 0x1b, 0x0f, 0xde,
	0xf2, 0x74, 0xac, 0xc9, 0xa1, 0x16, 0x79, 0x7b, 0x26, 0xee, 0xed, 0xd3, 0x03, 0xf0, 0x36, 0x64,
	0x1d, 0x06, 0xa1, 0x9a, 0x44, 0x06, 0x36, 0x7f, 0x26, 0x40, 0x5f, 0xc0, 0xdc, 0xa1, 0x71, 0x4a,
	0x5c, 0x39, 0xc7, 0xac, 0xbc, 0x3b, 0xdd, 0xca, 0x1d, 0xe3, 0x14, 0x73, 0x1d, 0x1a, 0x9a, 0xdf,
	0xaa, 0x8e, 0x65, 0x58, 0x1d, 0x11, 0x82, 0xe1, 0x10, 0x7d, 0x02, 0x19, 0xcd, 0xb6, 0x3c, 0xc7,
	0x36, 0x79, 0x18, 0xe6, 0x86, 0xae, 0x6a, 0x5b, 0x4c, 0xe2, 0x68, 0x19, 0xb2, 0x00, 0x34, 0xb3,
	0xef, 0x7a, 0xc4, 0xa1, 0xee, 0xb7, 0xc4, 0xbc, 0xa4, 0x11, 0xf8, 0xca, 0xcb, 0xe8, 0x22, 0xf9,
	0x6c, 0x2c, 0xb1, 0x9d, 0x45, 0x0b, 0x9f, 0x1a, 0x99, 0x86, 0x1f, 0x50, 0xaf, 0x5b, 0xeb, 0xf7,
	0x0d, 0x7d, 0xbd, 0x88, 0xb3, 0x62, 0x8b, 0x9a, 0xfe, 0x21, 0xf9, 0xa0, 0x02, 0xb3, 0x3b, 0xc6,
	0x29, 0x5a, 0x81, 0x4c, 0xcf, 0x54, 0xbd, 0x43, 0xdb, 0xe9, 0x0a, 0xbd, 0x68, 0x8c, 0x56, 0x01,
	0x74, 0xc3, 0x75, 0xfa, 0xfc, 0x5a, 0x39, 0x42, 0x4c, 0x52, 0xb4, 0x21, 0x1f, 0x3f, 0xc7, 0x44,
	0xc2, 0x49, 0x7d, 0xd7, 0x84, 0xb3, 0x12, 0xbb, 0xb5, 0x19, 0x46, 0x9e, 0xd1, 0xb8, 0xf8, 0x57,
	0x80, 0xbb, 0xe7, 0x7d, 0xa2, 0xa9, 0xa9, 0xd6, 0xb6, 0x6d, 0x1d, 0x1a, 0x9d, 0x3e, 0xf7, 0xa9,
	0x83, 0x32, 0xfa, 0x79, 0x8c, 0x62, 0x2a, 0x81, 0xaf, 0x3c, 0x1b, 0x65, 0x84, 0xa6, 0x5a, 0x05,
	0xae, 0x78, 0xee, 0x3e, 0x7b, 0xc7, 0x89, 0x2b, 0xa3, 0xbc, 0xd4, 0x06, 0xc9, 0xd5, 0x54, 0xab,
	0xad, 0x31, 0x0d, 0xce, 0x9f, 0xbc, 0x0a, 0xfa, 0x2c, 0xf0, 0x95, 0x4f, 0xa6, 0x6c, 0xc0, 0x58,
	0x91, 0x63, 0xf3, 0x8a, 0xa0, 0x88, 0x97, 0xdc, 0xc8, 0x70, 0x3a, 0x8b, 0x9e, 0xc0, 0x2d, 0xb5,
	0xef, 0xd9, 0x6d, 0xb5, 0xd7, 0x33, 0x07, 0x6d, 0x87, 0x74, 0x89, 0x6e, 0x88, 0xf0, 0xa5, 0x94,
	0x97, 0xc1, 0x37, 0xe8, 0x74, 0x85, 0xce, 0xe2, 0xd8, 0x24, 0x7a, 0x0a, 0x32, 0xd3, 0xeb, 0xf7,
	0x74, 0xd5, 0x23, 0x49, 0xc5, 0x34, 0x53, 0xbc, 0x49, 0xe7, 0xf7, 0xd9, 0x74, 0x42, 0xb3, 0x08,
	0x8b, 0xb6, 0x45, 0xda, 0xb4, 0x96, 0x6d, 0x53, 0x63, 0x18, 0xa3, 0x65, 0x70, 0xce, 0xb6, 0x48,
	0xcb, 0xe8, 0x12, 0x6a, 0x3f, 0xc2, 0x43, 0x7c, 0xb4, 0x39, 0x21, 0x3a, 0x47, 0xdc, 0xc4, 0x48,
	0x72, 0x52, 0x47, 0x91, 0xd3, 0x57, 0x97, 0x04, 0x9e, 0xcc, 0x54, 0x8f, 0x20, 0x23, 0xca, 0xac,
	0xb0, 0xd4, 0xb8, 0x1e, 0xe1, 0xc7, 0x8a, 0x6f, 0x1c, 0xad, 0x42, 0x8f, 0x00, 0x2c, 0x5b, 0x27,
	0x6d, 0xc7, 0xa6, 0x3a, 0xd9, 0xc2, 0xec, 0xda, 0x52, 0xf9, 0x6a, 0xa4, 0x53, 0xb7, 0x75, 0x82,
	0x6d, 0x93, 0xe0, 0xac, 0x25, 0x7e, 0xb9, 0x68, 0x0d, 0x24, 0xd7, 0x73, 0x0c, 0xcd, 0x6b, 0x33,
	0x45, 0x76, 0x82, 0xc0, 0x4e, 0x70, 0x89, 0xcb, 0xa9, 0x12, 0x3b, 0xc4, 0x0d, 0xc8, 0x84, 0x7d,
	0x01, 0x2b, 0x20, 0x72, 0x31, 0xe4, 0xa6, 0x98, 0xc0, 0xd1, 0x12, 0xf4, 0x0c, 0xf2, 0x9a, 0x43,
	0x54, 0x8f, 0xe8, 0xec, 0x6e, 0x18, 0xb1, 0xe5, 0xca, 0x2b, 0x25, 0xde, 0x84, 0x94, 0xc2, 0x26,
	0xa4, 0xd4, 0x0a, 0x9b, 0x10, 0x9c, 0x13, 0xeb, 0xa9, 0x04, 0xed, 0xc0, 0x55, 0x53, 0x75, 0x3d,
	0xe1, 0x10, 0x02, 0x63, 0x71, 0x2a, 0xc6, 0x32, 0x55, 0xe2, 0x5e, 0xc2, 0x71, 0x5a, 0x90, 0xeb,
	0xda, 0xba, 0x71, 0x68, 0x10, 0xbd, 0xfd, 0x66, 0xc0, 0xe8, 0x30, 0x61, 0xb8, 0x69, 0x74, 0xf7,
	0x5d, 0xe2, 0x6c, 0x15, 0x02, 0x5f, 0xb9, 0xcd, 0xeb, 0xe1, 0x8e, 0x65, 0x3b, 0xa4, 0xcd, 0xaf,
	0x79, 0x8d, 0x4e, 0x16, 0x6a, 0xcf, 0xd7, 0x8b, 0x18, 0x42, 0x9c, 0xad, 0xc1, 0x70, 0xb6, 0x58,
	0x3e, 0x9f, 0x2d, 0x30, 0x64, 0x04, 0x45, 0xba, 0xb2, 0xc4, 0xee, 0xee, 0xc9, 0x25, 0x7d, 0x43,
	0x70, 0x31, 0x8e, 0x70, 0x3e, 0x24, 0xd3, 0x0e, 0x60, 0x41, 0xd8, 0x33, 0x94, 0x5f, 0x52, 0xef,
	0x3b, 0xbf, 0x14, 0xff, 0x92, 0x86, 0xf5, 0xf3, 0x27, 0x26, 0x90, 0xce, 0x0e, 0xae, 0xe9, 0xa9,
	0x5e, 0xdf, 0x45, 0xf7, 0x18, 0x69, 0xf2, 0x56, 0xe1, 0x56, 0xe0, 0x2b, 0xd7, 0xc6, 0x51, 0xe1,
	0xff, 0xf9, 0x33, 0xd0, 0x1f, 0x53, 0xb0, 0x14, 0xe7, 0xde, 0xa8, 0xff, 0xfc, 0x75, 0xe0, 0x2b,
	0xa7, 0x97, 0xa4, 0xf6, 0x91, 0x85, 0xed, 0x08, 0x27, 0x9a, 0x6e, 0x61, 0xfe, 0x8c, 0xc0, 0x6b,
	0x3a, 0xad, 0xf9, 0x89, 0xe3, 0xd8, 0x0e, 0x65, 0x6b, 0x56, 0xf3, 0xf3, 0x11, 0xfa, 0x5d, 0x6a,
	0x54, 0x38, 0xa6, 0xa7, 0x85, 0xe3, 0xd6, 0x97, 0x81, 0xaf, 0x3c, 0x9d, 0xf2, 0x6d, 0xbb, 0xaa,
	0xeb, 0x15, 0x44, 0xc8, 0x16, 0xa8, 0x6a, 0x94, 0x50, 0xcf, 0x05, 0xf3, 0x8f, 0x21, 0x1f, 0xde,
	0x19, 0x4b, 0x5d, 0x73, 0x3c, 0xee, 0x84, 0x8c, 0xa5, 0xa7, 0xff, 0xe6, 0x41, 0x19, 0xe1, 0x2d,
	0x47, 0x44, 0x3b, 0xc6, 0xc4, 0xed, 0x9b, 0xde, 0xd4, 0xda, 0x7d, 0x0b, 0x32, 0x1a, 0x5d, 0x7e,
	0x76, 0x43, 0xe3, 0x2a, 0x00, 0x86, 0x1a, 0x7f, 0x1f, 0x58, 0x60, 0x8a, 0x35, 0xf6, 0xce, 0xc0,
	0x31, 0x62, 0xc5, 0xfe, 0xb8, 0x77, 0x06, 0x8e, 0x92, 0xa8, 0xf6, 0xb3, 0x4c, 0x99, 0x25, 0xd4,
	0xa4, 0x9b, 0xa6, 0xdf, 0xbb, 0x9b, 0x0e, 0x1f, 0xf1, 0xf2, 0xb9, 0x23, 0x46, 0x03, 0x98, 0x77,
	0x59, 0xb0, 0x89, 0xe6, 0xe1, 0xf1, 0x04, 0x62, 0x4b, 0x1c, 0x7c, 0x89, 0x8d, 0x78, 0xa0, 0x6e,
	0x7d, 0x1c, 0xf8, 0xca, 0xbd, 0xb1, 0xc7, 0xc1, 0x17, 0x45, 0x07, 0x22, 0x36, 0x4c, 0x74, 0x2e,
	0xf3, 0xef, 0xa3, 0x73, 0x19, 0x22, 0xf6, 0x85, 0xf3, 0xc4, 0x5e, 0x84, 0xbc, 0x41, 0x0f, 0xb1,
	0xaf, 0x85, 0x6f, 0x00, 0x74, 0x49, 0x42, 0x86, 0x76, 0xa3, 0x7a, 0x23, 0xcb, 0xa8, 0xff, 0xe2,
	0x27, 0x34, 0xaa, 0xd2, 0xf8, 0x65, 0xb2, 0xd2, 0xe0, 0xfd, 0xff, 0xe7, 0x17, 0x86, 0x9c, 0x5c,
	0x63, 0x7c, 0x3b, 0x94, 0xa6, 0x73, 0x53, 0x63, 0xfa, 0x69, 0xe0, 0x2b, 0x8f, 0xc7, 0x5e, 0x1c,
	0xdf, 0xb6, 0xb0, 0xcd, 0x31, 0x93, 0xf1, 0x9c, 0x48, 0xf0, 0xf1, 0x42, 0x3b, 0xff, 0x5d, 0x0b,
	0x6d, 0x19, 0x16, 0x44, 0x61, 0xcd, 0x6a, 0x83, 0x2c, 0x0e, 0x87, 0xe8, 0xb7, 0x29, 0xc8, 0x32,
	0xba, 0x65, 0x5e, 0xcc, 0xbb, 0xa0, 0xc3, 0xc0, 0x57, 0xde, 0x8c, 0x63, 0xa3, 0xf3, 0x4f, 0x74,
	0x63, 0x49, 0xf6, 0xa0, 0xbc, 0x19, 0x41, 0x8f, 0x7c, 0x42, 0xa0, 0xb3, 0x2c, 0x54, 0xfe, 0x9c,
	0x1a, 0x51, 0x70, 0x4b, 0xcc, 0x98, 0x5f, 0x05, 0xbe, 0x72, 0x72, 0xf1, 0x82, 0xfb, 0xc2, 0x8c,
	0x3f, 0xbc, 0xdd, 0x28, 0x03, 0x87, 0xab, 0xf6, 0x44, 0x6b, 0x7b, 0x75, 0xb8, 0xb5, 0x5d, 0x05,
	0xe0, 0x4f, 0x3f, 0xfb, 0x2e, 0xd1, 0x65, 0xc4, 0x12, 0x43, 0x4c, 0x42, 0xbb, 0x1d, 0xd1, 0xae,
	0xba, 0xf2, 0x35, 0xde, 0xed, 0x84, 0xe3, 0x0f, 0x58, 0xb2, 0x14, 0x5d, 0xc8, 0xc5, 0xf8, 0x06,
	0x65, 0x61, 0x6e, 0xbf, 0xde, 0xac, 0xb6, 0xa4, 0x2b, 0x28, 0x03, 0xe9, 0xbd, 0x4a, 0xb3, 0x29,
	0xa5, 0xe8, 0xaf, 0x9d, 0x4a, 0x6d, 0x57, 0x9a, 0xa1, 0xd3, 0x55, 0x8c, 0x1b, 0x58, 0x9a, 0xa5,
	0xc2, 0x5a, 0x7d, 0xa7, 0x21, 0xa5, 0x11, 0xc0, 0xfc, 0xab, 0x4a, 0x7d, 0xbf, 0xb2, 0x2b, 0xcd,
	0x21, 0x04, 0x4b, 0xf5, 0x46, 0xab, 0x5d, 0xd9, 0xdb, 0xdb, 0xad, 0x6d, 0x57, 0xb6, 0x76, 0xab,
	0xd2, 0x3c, 0x92, 0x20, 0x5f, 0xab, 0x6f, 0x37, 0xea, 0xcd, 0x5a, 0xb3, 0x55, 0xad, 0xb7, 0xa4,
	0x85, 0xe2, 0x01, 0x00, 0xbd, 0x14, 0xb1, 0xe7, 0x75, 0x98, 0xeb, 0x1d, 0xa9, 0xae, 0x78, 0xa3,
	0xc2, 0x7c, 0x40, 0x93, 0xac, 0xc3, 0xa2, 0x24, 0x7a, 0x1b, 0x62, 0xa3, 0xc4, 0x39, 0xce, 0xf2,
	0x36, 0x36, 0x1c, 0x17, 0xff, 0x9d, 0x19, 0xf5, 0x16, 0xc5, 0x9d, 0x71, 0x4a, 0x3e, 0x1b, 0xe9,
	0x83, 0x33, 0x3f, 0x38, 0x1f, 0xac, 0xc6, 0xe3, 0x75, 0x71, 0xe2, 0xdb, 0xd6, 0xb9, 0x78, 0x8d,
	0x45, 0x5c, 0x32, 0x5f, 0xce, 0xbe, 0xf7, 0x7c, 0x79, 0x56, 0x31, 0xa5, 0xf9, 0x65, 0x8a, 0x8a,
	0x29, 0x7e, 0x99, 0x4b, 0xc9, 0xcb, 0x44, 0x25, 0xf6, 0x12, 0x4b, 0x3b, 0x36, 0x96, 0x41, 0xc7,
	0xb5, 0x75, 0xe1, 0xa2, 0x4b, 0x3d, 0xa7, 0x71, 0x97, 0xf8, 0x7e, 0x9e, 0xd3, 0x04, 0xd6, 0xe4,
	0x04, 0x52, 0x12, 0xf7, 0xc6, 0xde, 0x24, 0x33, 0x2c, 0x27, 0xc7, 0xfb, 0x42, 0xd5, 0x6a, 0x0d,
	0x7a, 0x84, 0x5f, 0x10, 0x7b, 0x7c, 0x7c, 0x02, 0x8b, 0xbc, 0xd3, 0x24, 0x26, 0xd1, 0x3c, 0xdb,
	0x61, 0x4f, 0x6d, 0x23, 0xbb, 0xd4, 0x3c, 0x5d, 0xd7, 0x14, 0xcb, 0xd0, 0xc7, 0x51, 0xd5, 0x01,
	0xec, 0xcc, 0xae, 0x25, 0x36, 0xe1, 0x01, 0x17, 0xd5, 0x09, 0xcf, 0x2e, 0x9b, 0xd5, 0x92, 0xb9,
	0xe9, 0x37, 0x29, 0x40, 0xac, 0xdc, 0x25, 0xa7, 0x44, 0xeb, 0x5f, 0xbc, 0x85, 0xdd, 0x7a, 0x12,
	0xf8, 0x4a, 0x79, 0x9c, 0xc7, 0xb2, 0x42, 0xb7, 0x2a, 0x10, 0x93, 0x99, 0x51, 0xa2, 0xbb, 0x85,
	0x53, 0x74, 0xe6, 0x43, 0x12, 0xe7, 0x7f, 0x66, 0xe0, 0xce, 0xd8, 0x3f, 0x15, 0x44, 0x8c, 0x54,
	0xf5, 0x0e, 0x41, 0x4a, 0x8c, 0x77, 0x96, 0x03, 0x5f, 0xc9, 0x85, 0xbd, 0x96, 0xa1, 0x73, 0xea,
	0x79, 0x3b, 0xe2, 0xef, 0x36, 0x1c, 0xf8, 0x4a, 0xfd, 0x42, 0x7f, 0xb7, 0x5d, 0xf4, 0x0f, 0xc3,
	0xc4, 0xff, 0x72, 0x2f, 0x21, 0x17, 0xfd, 0x6b, 0x15, 0x11, 0xc0, 0xfd, 0xc0, 0x57, 0x7e, 0x3a,
	0x61, 0xcf, 0xfd, 0x58, 0x0d, 0x1f, 0x5a, 0xbc, 0x7f, 0xae, 0x47, 0x7c, 0xef, 0xc5, 0xf7, 0xfd,
	0x0d, 0xc8, 0x84, 0xde, 0x4f, 0x13, 0x52, 0xad, 0xbe, 0x83, 0x2b, 0xd2, 0x15, 0x9a, 0x86, 0x5e,
	0x37, 0xf0, 0xcb, 0x2a, 0x96, 0x52, 0x3c, 0x25, 0x35, 0x5b, 0x55, 0x2c, 0xcd, 0xdc, 0xaf, 0x40,
	0x26, 0x0c, 0x30, 0x74, 0x0d, 0x96, 0x59, 0x7a, 0x6b, 0x37, 0xb7, 0x2b, 0xf5, 0x76, 0xeb, 0x9b,
	0xbd, 0xaa, 0x74, 0x05, 0x2d, 0x42, 0xb6, 0xde, 0x78, 0x5e, 0x65, 0x32, 0x29, 0x85, 0xae, 0xc2,
	0xe2, 0xde, 0x6e, 0xa5, 0xb5, 0xd3, 0xc0, 0xaf, 0xb8, 0x68, 0xe6, 0xfe, 0x9f, 0x52, 0xfc, 0xfd,
	0x33, 0x2c, 0x89, 0xd1, 0x2d, 0xb8, 0xc6, 0x71, 0xf0, 0xfe, 0x6e, 0xb5, 0xdd, 0xac, 0x1e, 0x54,
	0x71, 0xad, 0xf5, 0x8d, 0x74, 0x05, 0x7d, 0x04, 0x37, 0xf6, 0xeb, 0x2f, 0xeb, 0x8d, 0xd7, 0xf5,
	0xa1, 0xa9, 0x14, 0xba, 0x09, 0x88, 0x26, 0xcc, 0x21, 0xf9, 0x0c, 0xba, 0x01, 0x57, 0x77, 0x1b,
	0xaf, 0x87, 0xc4, 0xb3, 0x48, 0x86, 0xeb, 0xaf, 0xaa, 0xcf, 0x6b, 0xfb, 0xaf, 0x86, 0x66, 0xd2,
	0x14, 0xe8, 0x45, 0xed, 0xeb, 0x17, 0x43, 0xf2, 0xb9, 0xad, 0xc7, 0x7f, 0x7b, 0xb7, 0x9a, 0xfa,
	0xfb, 0xbb, 0xd5, 0xd4, 0x3f, 0xde, 0xad, 0xa6, 0xfe, 0xf0, 0xcf, 0xd5, 0x2b, 0xf0, 0x91, 0x61,
	0x97, 0x5c, 0x4f, 0xd5, 0x8e, 0x1d, 0xfb, 0x94, 0x07, 0x62, 0x48, 0x07, 0xbf, 0x08, 0xff, 0x05,
	0x7f, 0x33, 0xcf, 0xe4, 0x9f, 0xfe, 0x2f, 0x00, 0x00, 0xff, 0xff, 0x45, 0xed, 0xdb, 0x4e, 0x3d,
	0x1f, 0x00, 0x00,
>>>>>>> c289036eb5 (X-Smart-Squash: Squashed 39 commits:)
}

func (m *ProfileShim) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ProfileShim) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ProfileShim) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.ProfileId) > 0 {
		i -= len(m.ProfileId)
		copy(dAtA[i:], m.ProfileId)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ProfileId)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *ComplianceOperatorProfileV2) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ComplianceOperatorProfileV2) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ComplianceOperatorProfileV2) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.Values) > 0 {
		for iNdEx := len(m.Values) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Values[iNdEx])
			copy(dAtA[i:], m.Values[iNdEx])
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Values[iNdEx])))
			i--
			dAtA[i] = 0x6a
		}
	}
	if len(m.Title) > 0 {
		i -= len(m.Title)
		copy(dAtA[i:], m.Title)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Title)))
		i--
		dAtA[i] = 0x62
	}
	if len(m.Product) > 0 {
		i -= len(m.Product)
		copy(dAtA[i:], m.Product)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Product)))
		i--
		dAtA[i] = 0x5a
	}
	if len(m.Rules) > 0 {
		for iNdEx := len(m.Rules) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Rules[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x52
		}
	}
	if len(m.Description) > 0 {
		i -= len(m.Description)
		copy(dAtA[i:], m.Description)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Description)))
		i--
		dAtA[i] = 0x4a
	}
	if len(m.Annotations) > 0 {
		for k := range m.Annotations {
			v := m.Annotations[k]
			baseI := i
			i -= len(v)
			copy(dAtA[i:], v)
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(v)))
			i--
			dAtA[i] = 0x12
			i -= len(k)
			copy(dAtA[i:], k)
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(k)))
			i--
			dAtA[i] = 0xa
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(baseI-i))
			i--
			dAtA[i] = 0x42
		}
	}
	if len(m.Labels) > 0 {
		for k := range m.Labels {
			v := m.Labels[k]
			baseI := i
			i -= len(v)
			copy(dAtA[i:], v)
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(v)))
			i--
			dAtA[i] = 0x12
			i -= len(k)
			copy(dAtA[i:], k)
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(k)))
			i--
			dAtA[i] = 0xa
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(baseI-i))
			i--
			dAtA[i] = 0x3a
		}
	}
	if len(m.Standard) > 0 {
		i -= len(m.Standard)
		copy(dAtA[i:], m.Standard)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Standard)))
		i--
		dAtA[i] = 0x32
	}
	if len(m.ProductType) > 0 {
		i -= len(m.ProductType)
		copy(dAtA[i:], m.ProductType)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ProductType)))
		i--
		dAtA[i] = 0x2a
	}
	if len(m.ProfileVersion) > 0 {
		i -= len(m.ProfileVersion)
		copy(dAtA[i:], m.ProfileVersion)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ProfileVersion)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.Name) > 0 {
		i -= len(m.Name)
		copy(dAtA[i:], m.Name)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Name)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.ProfileId) > 0 {
		i -= len(m.ProfileId)
		copy(dAtA[i:], m.ProfileId)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ProfileId)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Id) > 0 {
		i -= len(m.Id)
		copy(dAtA[i:], m.Id)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Id)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *ComplianceOperatorProfileV2_Rule) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ComplianceOperatorProfileV2_Rule) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ComplianceOperatorProfileV2_Rule) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.RuleName) > 0 {
		i -= len(m.RuleName)
		copy(dAtA[i:], m.RuleName)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.RuleName)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *ComplianceOperatorRuleV2) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ComplianceOperatorRuleV2) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ComplianceOperatorRuleV2) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.ClusterId) > 0 {
		i -= len(m.ClusterId)
		copy(dAtA[i:], m.ClusterId)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ClusterId)))
		i--
		dAtA[i] = 0x72
	}
	if len(m.Controls) > 0 {
		for iNdEx := len(m.Controls) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Controls[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x6a
		}
	}
	if len(m.Warning) > 0 {
		i -= len(m.Warning)
		copy(dAtA[i:], m.Warning)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Warning)))
		i--
		dAtA[i] = 0x62
	}
	if len(m.Fixes) > 0 {
		for iNdEx := len(m.Fixes) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Fixes[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x5a
		}
	}
	if len(m.Rationale) > 0 {
		i -= len(m.Rationale)
		copy(dAtA[i:], m.Rationale)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Rationale)))
		i--
		dAtA[i] = 0x52
	}
	if len(m.Description) > 0 {
		i -= len(m.Description)
		copy(dAtA[i:], m.Description)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Description)))
		i--
		dAtA[i] = 0x4a
	}
	if len(m.Title) > 0 {
		i -= len(m.Title)
		copy(dAtA[i:], m.Title)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Title)))
		i--
		dAtA[i] = 0x42
	}
	if len(m.Annotations) > 0 {
		for k := range m.Annotations {
			v := m.Annotations[k]
			baseI := i
			i -= len(v)
			copy(dAtA[i:], v)
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(v)))
			i--
			dAtA[i] = 0x12
			i -= len(k)
			copy(dAtA[i:], k)
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(k)))
			i--
			dAtA[i] = 0xa
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(baseI-i))
			i--
			dAtA[i] = 0x3a
		}
	}
	if len(m.Labels) > 0 {
		for k := range m.Labels {
			v := m.Labels[k]
			baseI := i
			i -= len(v)
			copy(dAtA[i:], v)
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(v)))
			i--
			dAtA[i] = 0x12
			i -= len(k)
			copy(dAtA[i:], k)
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(k)))
			i--
			dAtA[i] = 0xa
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(baseI-i))
			i--
			dAtA[i] = 0x32
		}
	}
	if m.Severity != 0 {
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(m.Severity))
		i--
		dAtA[i] = 0x28
	}
	if len(m.RuleType) > 0 {
		i -= len(m.RuleType)
		copy(dAtA[i:], m.RuleType)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.RuleType)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.Name) > 0 {
		i -= len(m.Name)
		copy(dAtA[i:], m.Name)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Name)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.RuleId) > 0 {
		i -= len(m.RuleId)
		copy(dAtA[i:], m.RuleId)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.RuleId)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Id) > 0 {
		i -= len(m.Id)
		copy(dAtA[i:], m.Id)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Id)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *ComplianceOperatorRuleV2_Fix) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ComplianceOperatorRuleV2_Fix) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ComplianceOperatorRuleV2_Fix) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.Disruption) > 0 {
		i -= len(m.Disruption)
		copy(dAtA[i:], m.Disruption)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Disruption)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Platform) > 0 {
		i -= len(m.Platform)
		copy(dAtA[i:], m.Platform)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Platform)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *RuleControls) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *RuleControls) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *RuleControls) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.Controls) > 0 {
		for iNdEx := len(m.Controls) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Controls[iNdEx])
			copy(dAtA[i:], m.Controls[iNdEx])
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Controls[iNdEx])))
			i--
			dAtA[i] = 0x12
		}
	}
	if len(m.Standard) > 0 {
		i -= len(m.Standard)
		copy(dAtA[i:], m.Standard)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Standard)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *ComplianceOperatorScanConfigurationV2) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ComplianceOperatorScanConfigurationV2) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ComplianceOperatorScanConfigurationV2) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.Clusters) > 0 {
		for iNdEx := len(m.Clusters) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Clusters[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x1
			i--
			dAtA[i] = 0x82
		}
	}
	if len(m.Description) > 0 {
		i -= len(m.Description)
		copy(dAtA[i:], m.Description)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Description)))
		i--
		dAtA[i] = 0x7a
	}
	if m.ModifiedBy != nil {
		{
			size, err := m.ModifiedBy.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x72
	}
	if m.LastUpdatedTime != nil {
		{
			size, err := m.LastUpdatedTime.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x6a
	}
	if m.CreatedTime != nil {
		{
			size, err := m.CreatedTime.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x62
	}
	if m.Schedule != nil {
		{
			size, err := m.Schedule.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x5a
	}
	if m.StrictNodeScan {
		i--
		if m.StrictNodeScan {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x50
	}
	if len(m.NodeRoles) > 0 {
		dAtA6 := make([]byte, len(m.NodeRoles)*10)
		var j5 int
		for _, num := range m.NodeRoles {
			for num >= 1<<7 {
				dAtA6[j5] = uint8(uint64(num)&0x7f | 0x80)
				num >>= 7
				j5++
			}
			dAtA6[j5] = uint8(num)
			j5++
		}
		i -= j5
		copy(dAtA[i:], dAtA6[:j5])
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(j5))
		i--
		dAtA[i] = 0x4a
	}
	if len(m.Profiles) > 0 {
		for iNdEx := len(m.Profiles) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Profiles[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x42
		}
	}
	if len(m.Annotations) > 0 {
		for k := range m.Annotations {
			v := m.Annotations[k]
			baseI := i
			i -= len(v)
			copy(dAtA[i:], v)
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(v)))
			i--
			dAtA[i] = 0x12
			i -= len(k)
			copy(dAtA[i:], k)
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(k)))
			i--
			dAtA[i] = 0xa
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(baseI-i))
			i--
			dAtA[i] = 0x3a
		}
	}
	if len(m.Labels) > 0 {
		for k := range m.Labels {
			v := m.Labels[k]
			baseI := i
			i -= len(v)
			copy(dAtA[i:], v)
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(v)))
			i--
			dAtA[i] = 0x12
			i -= len(k)
			copy(dAtA[i:], k)
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(k)))
			i--
			dAtA[i] = 0xa
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(baseI-i))
			i--
			dAtA[i] = 0x32
		}
	}
	if m.OneTimeScan {
		i--
		if m.OneTimeScan {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x28
	}
	if m.AutoUpdateRemediations {
		i--
		if m.AutoUpdateRemediations {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x20
	}
	if m.AutoApplyRemediations {
		i--
		if m.AutoApplyRemediations {
			dAtA[i] = 1
		} else {
			dAtA[i] = 0
		}
		i--
		dAtA[i] = 0x18
	}
	if len(m.ScanConfigName) > 0 {
		i -= len(m.ScanConfigName)
		copy(dAtA[i:], m.ScanConfigName)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ScanConfigName)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Id) > 0 {
		i -= len(m.Id)
		copy(dAtA[i:], m.Id)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Id)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *ComplianceOperatorScanConfigurationV2_Cluster) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ComplianceOperatorScanConfigurationV2_Cluster) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ComplianceOperatorScanConfigurationV2_Cluster) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.ClusterId) > 0 {
		i -= len(m.ClusterId)
		copy(dAtA[i:], m.ClusterId)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ClusterId)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *ComplianceOperatorClusterScanConfigStatus) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ComplianceOperatorClusterScanConfigStatus) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ComplianceOperatorClusterScanConfigStatus) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.Id) > 0 {
		i -= len(m.Id)
		copy(dAtA[i:], m.Id)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Id)))
		i--
		dAtA[i] = 0x32
	}
	if len(m.ClusterName) > 0 {
		i -= len(m.ClusterName)
		copy(dAtA[i:], m.ClusterName)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ClusterName)))
		i--
		dAtA[i] = 0x2a
	}
	if m.LastUpdatedTime != nil {
		{
			size, err := m.LastUpdatedTime.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x22
	}
	if len(m.Errors) > 0 {
		for iNdEx := len(m.Errors) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Errors[iNdEx])
			copy(dAtA[i:], m.Errors[iNdEx])
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Errors[iNdEx])))
			i--
			dAtA[i] = 0x1a
		}
	}
	if len(m.ScanConfigId) > 0 {
		i -= len(m.ScanConfigId)
		copy(dAtA[i:], m.ScanConfigId)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ScanConfigId)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.ClusterId) > 0 {
		i -= len(m.ClusterId)
		copy(dAtA[i:], m.ClusterId)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ClusterId)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *ComplianceOperatorCheckResultV2) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ComplianceOperatorCheckResultV2) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ComplianceOperatorCheckResultV2) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.Warnings) > 0 {
		for iNdEx := len(m.Warnings) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Warnings[iNdEx])
			copy(dAtA[i:], m.Warnings[iNdEx])
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Warnings[iNdEx])))
			i--
			dAtA[i] = 0x1
			i--
			dAtA[i] = 0x9a
		}
	}
	if len(m.ValuesUsed) > 0 {
		for iNdEx := len(m.ValuesUsed) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.ValuesUsed[iNdEx])
			copy(dAtA[i:], m.ValuesUsed[iNdEx])
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ValuesUsed[iNdEx])))
			i--
			dAtA[i] = 0x1
			i--
			dAtA[i] = 0x92
		}
	}
	if len(m.Rationale) > 0 {
		i -= len(m.Rationale)
		copy(dAtA[i:], m.Rationale)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Rationale)))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0x8a
	}
	if len(m.ScanConfigName) > 0 {
		i -= len(m.ScanConfigName)
		copy(dAtA[i:], m.ScanConfigName)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ScanConfigName)))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0x82
	}
	if len(m.ClusterName) > 0 {
		i -= len(m.ClusterName)
		copy(dAtA[i:], m.ClusterName)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ClusterName)))
		i--
		dAtA[i] = 0x7a
	}
	if len(m.ScanName) > 0 {
		i -= len(m.ScanName)
		copy(dAtA[i:], m.ScanName)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ScanName)))
		i--
		dAtA[i] = 0x72
	}
	if len(m.Control) > 0 {
		i -= len(m.Control)
		copy(dAtA[i:], m.Control)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Control)))
		i--
		dAtA[i] = 0x6a
	}
	if len(m.Standard) > 0 {
		i -= len(m.Standard)
		copy(dAtA[i:], m.Standard)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Standard)))
		i--
		dAtA[i] = 0x62
	}
	if m.CreatedTime != nil {
		{
			size, err := m.CreatedTime.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x5a
	}
	if len(m.Annotations) > 0 {
		for k := range m.Annotations {
			v := m.Annotations[k]
			baseI := i
			i -= len(v)
			copy(dAtA[i:], v)
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(v)))
			i--
			dAtA[i] = 0x12
			i -= len(k)
			copy(dAtA[i:], k)
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(k)))
			i--
			dAtA[i] = 0xa
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(baseI-i))
			i--
			dAtA[i] = 0x52
		}
	}
	if len(m.Labels) > 0 {
		for k := range m.Labels {
			v := m.Labels[k]
			baseI := i
			i -= len(v)
			copy(dAtA[i:], v)
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(v)))
			i--
			dAtA[i] = 0x12
			i -= len(k)
			copy(dAtA[i:], k)
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(k)))
			i--
			dAtA[i] = 0xa
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(baseI-i))
			i--
			dAtA[i] = 0x4a
		}
	}
	if len(m.Instructions) > 0 {
		i -= len(m.Instructions)
		copy(dAtA[i:], m.Instructions)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Instructions)))
		i--
		dAtA[i] = 0x42
	}
	if len(m.Description) > 0 {
		i -= len(m.Description)
		copy(dAtA[i:], m.Description)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Description)))
		i--
		dAtA[i] = 0x3a
	}
	if m.Severity != 0 {
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(m.Severity))
		i--
		dAtA[i] = 0x30
	}
	if m.Status != 0 {
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(m.Status))
		i--
		dAtA[i] = 0x28
	}
	if len(m.ClusterId) > 0 {
		i -= len(m.ClusterId)
		copy(dAtA[i:], m.ClusterId)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ClusterId)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.CheckName) > 0 {
		i -= len(m.CheckName)
		copy(dAtA[i:], m.CheckName)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.CheckName)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.CheckId) > 0 {
		i -= len(m.CheckId)
		copy(dAtA[i:], m.CheckId)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.CheckId)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Id) > 0 {
		i -= len(m.Id)
		copy(dAtA[i:], m.Id)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Id)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *ScanStatus) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ScanStatus) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ScanStatus) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.Warnings) > 0 {
		i -= len(m.Warnings)
		copy(dAtA[i:], m.Warnings)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Warnings)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Result) > 0 {
		i -= len(m.Result)
		copy(dAtA[i:], m.Result)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Result)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Phase) > 0 {
		i -= len(m.Phase)
		copy(dAtA[i:], m.Phase)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Phase)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *ComplianceOperatorScanV2) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ComplianceOperatorScanV2) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ComplianceOperatorScanV2) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.Warnings) > 0 {
		i -= len(m.Warnings)
		copy(dAtA[i:], m.Warnings)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Warnings)))
		i--
		dAtA[i] = 0x72
	}
	if len(m.ScanName) > 0 {
		i -= len(m.ScanName)
		copy(dAtA[i:], m.ScanName)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ScanName)))
		i--
		dAtA[i] = 0x6a
	}
	if m.LastExecutedTime != nil {
		{
			size, err := m.LastExecutedTime.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x62
	}
	if m.CreatedTime != nil {
		{
			size, err := m.CreatedTime.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x5a
	}
	if m.Status != nil {
		{
			size, err := m.Status.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x52
	}
	if m.NodeSelector != 0 {
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(m.NodeSelector))
		i--
		dAtA[i] = 0x48
	}
	if m.ScanType != 0 {
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(m.ScanType))
		i--
		dAtA[i] = 0x40
	}
	if len(m.Annotations) > 0 {
		for k := range m.Annotations {
			v := m.Annotations[k]
			baseI := i
			i -= len(v)
			copy(dAtA[i:], v)
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(v)))
			i--
			dAtA[i] = 0x12
			i -= len(k)
			copy(dAtA[i:], k)
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(k)))
			i--
			dAtA[i] = 0xa
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(baseI-i))
			i--
			dAtA[i] = 0x3a
		}
	}
	if len(m.Labels) > 0 {
		for k := range m.Labels {
			v := m.Labels[k]
			baseI := i
			i -= len(v)
			copy(dAtA[i:], v)
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(v)))
			i--
			dAtA[i] = 0x12
			i -= len(k)
			copy(dAtA[i:], k)
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(k)))
			i--
			dAtA[i] = 0xa
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(baseI-i))
			i--
			dAtA[i] = 0x32
		}
	}
	if m.Profile != nil {
		{
			size, err := m.Profile.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x2a
	}
	if len(m.Errors) > 0 {
		i -= len(m.Errors)
		copy(dAtA[i:], m.Errors)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Errors)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.ClusterId) > 0 {
		i -= len(m.ClusterId)
		copy(dAtA[i:], m.ClusterId)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ClusterId)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.ScanConfigName) > 0 {
		i -= len(m.ScanConfigName)
		copy(dAtA[i:], m.ScanConfigName)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ScanConfigName)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Id) > 0 {
		i -= len(m.Id)
		copy(dAtA[i:], m.Id)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Id)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *ComplianceOperatorProfileClusterEdge) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ComplianceOperatorProfileClusterEdge) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ComplianceOperatorProfileClusterEdge) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if len(m.ClusterId) > 0 {
		i -= len(m.ClusterId)
		copy(dAtA[i:], m.ClusterId)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ClusterId)))
		i--
		dAtA[i] = 0x22
	}
	if len(m.ProfileUid) > 0 {
		i -= len(m.ProfileUid)
		copy(dAtA[i:], m.ProfileUid)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ProfileUid)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.ProfileId) > 0 {
		i -= len(m.ProfileId)
		copy(dAtA[i:], m.ProfileId)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ProfileId)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Id) > 0 {
		i -= len(m.Id)
		copy(dAtA[i:], m.Id)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Id)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintComplianceOperatorV2(dAtA []byte, offset int, v uint64) int {
	offset -= sovComplianceOperatorV2(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *ProfileShim) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.ProfileId)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *ComplianceOperatorProfileV2) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Id)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.ProfileId)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.Name)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.ProfileVersion)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.ProductType)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.Standard)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if len(m.Labels) > 0 {
		for k, v := range m.Labels {
			_ = k
			_ = v
			mapEntrySize := 1 + len(k) + sovComplianceOperatorV2(uint64(len(k))) + 1 + len(v) + sovComplianceOperatorV2(uint64(len(v)))
			n += mapEntrySize + 1 + sovComplianceOperatorV2(uint64(mapEntrySize))
		}
	}
	if len(m.Annotations) > 0 {
		for k, v := range m.Annotations {
			_ = k
			_ = v
			mapEntrySize := 1 + len(k) + sovComplianceOperatorV2(uint64(len(k))) + 1 + len(v) + sovComplianceOperatorV2(uint64(len(v)))
			n += mapEntrySize + 1 + sovComplianceOperatorV2(uint64(mapEntrySize))
		}
	}
	l = len(m.Description)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if len(m.Rules) > 0 {
		for _, e := range m.Rules {
			l = e.Size()
			n += 1 + l + sovComplianceOperatorV2(uint64(l))
		}
	}
	l = len(m.Product)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.Title)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if len(m.Values) > 0 {
		for _, s := range m.Values {
			l = len(s)
			n += 1 + l + sovComplianceOperatorV2(uint64(l))
		}
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *ComplianceOperatorProfileV2_Rule) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.RuleName)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *ComplianceOperatorRuleV2) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Id)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.RuleId)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.Name)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.RuleType)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if m.Severity != 0 {
		n += 1 + sovComplianceOperatorV2(uint64(m.Severity))
	}
	if len(m.Labels) > 0 {
		for k, v := range m.Labels {
			_ = k
			_ = v
			mapEntrySize := 1 + len(k) + sovComplianceOperatorV2(uint64(len(k))) + 1 + len(v) + sovComplianceOperatorV2(uint64(len(v)))
			n += mapEntrySize + 1 + sovComplianceOperatorV2(uint64(mapEntrySize))
		}
	}
	if len(m.Annotations) > 0 {
		for k, v := range m.Annotations {
			_ = k
			_ = v
			mapEntrySize := 1 + len(k) + sovComplianceOperatorV2(uint64(len(k))) + 1 + len(v) + sovComplianceOperatorV2(uint64(len(v)))
			n += mapEntrySize + 1 + sovComplianceOperatorV2(uint64(mapEntrySize))
		}
	}
	l = len(m.Title)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.Description)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.Rationale)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if len(m.Fixes) > 0 {
		for _, e := range m.Fixes {
			l = e.Size()
			n += 1 + l + sovComplianceOperatorV2(uint64(l))
		}
	}
	l = len(m.Warning)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if len(m.Controls) > 0 {
		for _, e := range m.Controls {
			l = e.Size()
			n += 1 + l + sovComplianceOperatorV2(uint64(l))
		}
	}
	l = len(m.ClusterId)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *ComplianceOperatorRuleV2_Fix) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Platform)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.Disruption)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *RuleControls) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Standard)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if len(m.Controls) > 0 {
		for _, s := range m.Controls {
			l = len(s)
			n += 1 + l + sovComplianceOperatorV2(uint64(l))
		}
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *ComplianceOperatorScanConfigurationV2) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Id)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.ScanConfigName)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if m.AutoApplyRemediations {
		n += 2
	}
	if m.AutoUpdateRemediations {
		n += 2
	}
	if m.OneTimeScan {
		n += 2
	}
	if len(m.Labels) > 0 {
		for k, v := range m.Labels {
			_ = k
			_ = v
			mapEntrySize := 1 + len(k) + sovComplianceOperatorV2(uint64(len(k))) + 1 + len(v) + sovComplianceOperatorV2(uint64(len(v)))
			n += mapEntrySize + 1 + sovComplianceOperatorV2(uint64(mapEntrySize))
		}
	}
	if len(m.Annotations) > 0 {
		for k, v := range m.Annotations {
			_ = k
			_ = v
			mapEntrySize := 1 + len(k) + sovComplianceOperatorV2(uint64(len(k))) + 1 + len(v) + sovComplianceOperatorV2(uint64(len(v)))
			n += mapEntrySize + 1 + sovComplianceOperatorV2(uint64(mapEntrySize))
		}
	}
	if len(m.Profiles) > 0 {
		for _, e := range m.Profiles {
			l = e.Size()
			n += 1 + l + sovComplianceOperatorV2(uint64(l))
		}
	}
	if len(m.NodeRoles) > 0 {
		l = 0
		for _, e := range m.NodeRoles {
			l += sovComplianceOperatorV2(uint64(e))
		}
		n += 1 + sovComplianceOperatorV2(uint64(l)) + l
	}
	if m.StrictNodeScan {
		n += 2
	}
	if m.Schedule != nil {
		l = m.Schedule.Size()
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if m.CreatedTime != nil {
		l = m.CreatedTime.Size()
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if m.LastUpdatedTime != nil {
		l = m.LastUpdatedTime.Size()
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if m.ModifiedBy != nil {
		l = m.ModifiedBy.Size()
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.Description)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if len(m.Clusters) > 0 {
		for _, e := range m.Clusters {
			l = e.Size()
			n += 2 + l + sovComplianceOperatorV2(uint64(l))
		}
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *ComplianceOperatorScanConfigurationV2_Cluster) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.ClusterId)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *ComplianceOperatorClusterScanConfigStatus) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.ClusterId)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.ScanConfigId)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if len(m.Errors) > 0 {
		for _, s := range m.Errors {
			l = len(s)
			n += 1 + l + sovComplianceOperatorV2(uint64(l))
		}
	}
	if m.LastUpdatedTime != nil {
		l = m.LastUpdatedTime.Size()
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.ClusterName)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.Id)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *ComplianceOperatorCheckResultV2) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Id)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.CheckId)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.CheckName)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.ClusterId)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if m.Status != 0 {
		n += 1 + sovComplianceOperatorV2(uint64(m.Status))
	}
	if m.Severity != 0 {
		n += 1 + sovComplianceOperatorV2(uint64(m.Severity))
	}
	l = len(m.Description)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.Instructions)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if len(m.Labels) > 0 {
		for k, v := range m.Labels {
			_ = k
			_ = v
			mapEntrySize := 1 + len(k) + sovComplianceOperatorV2(uint64(len(k))) + 1 + len(v) + sovComplianceOperatorV2(uint64(len(v)))
			n += mapEntrySize + 1 + sovComplianceOperatorV2(uint64(mapEntrySize))
		}
	}
	if len(m.Annotations) > 0 {
		for k, v := range m.Annotations {
			_ = k
			_ = v
			mapEntrySize := 1 + len(k) + sovComplianceOperatorV2(uint64(len(k))) + 1 + len(v) + sovComplianceOperatorV2(uint64(len(v)))
			n += mapEntrySize + 1 + sovComplianceOperatorV2(uint64(mapEntrySize))
		}
	}
	if m.CreatedTime != nil {
		l = m.CreatedTime.Size()
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.Standard)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.Control)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.ScanName)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.ClusterName)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.ScanConfigName)
	if l > 0 {
		n += 2 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.Rationale)
	if l > 0 {
		n += 2 + l + sovComplianceOperatorV2(uint64(l))
	}
	if len(m.ValuesUsed) > 0 {
		for _, s := range m.ValuesUsed {
			l = len(s)
			n += 2 + l + sovComplianceOperatorV2(uint64(l))
		}
	}
	if len(m.Warnings) > 0 {
		for _, s := range m.Warnings {
			l = len(s)
			n += 2 + l + sovComplianceOperatorV2(uint64(l))
		}
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *ScanStatus) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Phase)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.Result)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.Warnings)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *ComplianceOperatorScanV2) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Id)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.ScanConfigName)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.ClusterId)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.Errors)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if m.Profile != nil {
		l = m.Profile.Size()
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if len(m.Labels) > 0 {
		for k, v := range m.Labels {
			_ = k
			_ = v
			mapEntrySize := 1 + len(k) + sovComplianceOperatorV2(uint64(len(k))) + 1 + len(v) + sovComplianceOperatorV2(uint64(len(v)))
			n += mapEntrySize + 1 + sovComplianceOperatorV2(uint64(mapEntrySize))
		}
	}
	if len(m.Annotations) > 0 {
		for k, v := range m.Annotations {
			_ = k
			_ = v
			mapEntrySize := 1 + len(k) + sovComplianceOperatorV2(uint64(len(k))) + 1 + len(v) + sovComplianceOperatorV2(uint64(len(v)))
			n += mapEntrySize + 1 + sovComplianceOperatorV2(uint64(mapEntrySize))
		}
	}
	if m.ScanType != 0 {
		n += 1 + sovComplianceOperatorV2(uint64(m.ScanType))
	}
	if m.NodeSelector != 0 {
		n += 1 + sovComplianceOperatorV2(uint64(m.NodeSelector))
	}
	if m.Status != nil {
		l = m.Status.Size()
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if m.CreatedTime != nil {
		l = m.CreatedTime.Size()
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if m.LastExecutedTime != nil {
		l = m.LastExecutedTime.Size()
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.ScanName)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.Warnings)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *ComplianceOperatorProfileClusterEdge) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Id)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.ProfileId)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.ProfileUid)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.ClusterId)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovComplianceOperatorV2(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozComplianceOperatorV2(x uint64) (n int) {
	return sovComplianceOperatorV2(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *ProfileShim) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowComplianceOperatorV2
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ProfileShim: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ProfileShim: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ProfileId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ProfileId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ComplianceOperatorProfileV2) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowComplianceOperatorV2
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ComplianceOperatorProfileV2: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ComplianceOperatorProfileV2: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Id = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ProfileId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ProfileId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Name", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Name = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ProfileVersion", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ProfileVersion = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ProductType", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ProductType = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Standard", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Standard = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Labels", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Labels == nil {
				m.Labels = make(map[string]string)
			}
			var mapkey string
			var mapvalue string
			for iNdEx < postIndex {
				entryPreIndex := iNdEx
				var wire uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowComplianceOperatorV2
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					wire |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				fieldNum := int32(wire >> 3)
				if fieldNum == 1 {
					var stringLenmapkey uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowComplianceOperatorV2
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapkey |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapkey := int(stringLenmapkey)
					if intStringLenmapkey < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					postStringIndexmapkey := iNdEx + intStringLenmapkey
					if postStringIndexmapkey < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if postStringIndexmapkey > l {
						return io.ErrUnexpectedEOF
					}
					mapkey = string(dAtA[iNdEx:postStringIndexmapkey])
					iNdEx = postStringIndexmapkey
				} else if fieldNum == 2 {
					var stringLenmapvalue uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowComplianceOperatorV2
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapvalue |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapvalue := int(stringLenmapvalue)
					if intStringLenmapvalue < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					postStringIndexmapvalue := iNdEx + intStringLenmapvalue
					if postStringIndexmapvalue < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if postStringIndexmapvalue > l {
						return io.ErrUnexpectedEOF
					}
					mapvalue = string(dAtA[iNdEx:postStringIndexmapvalue])
					iNdEx = postStringIndexmapvalue
				} else {
					iNdEx = entryPreIndex
					skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
					if err != nil {
						return err
					}
					if (skippy < 0) || (iNdEx+skippy) < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if (iNdEx + skippy) > postIndex {
						return io.ErrUnexpectedEOF
					}
					iNdEx += skippy
				}
			}
			m.Labels[mapkey] = mapvalue
			iNdEx = postIndex
		case 8:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Annotations", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Annotations == nil {
				m.Annotations = make(map[string]string)
			}
			var mapkey string
			var mapvalue string
			for iNdEx < postIndex {
				entryPreIndex := iNdEx
				var wire uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowComplianceOperatorV2
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					wire |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				fieldNum := int32(wire >> 3)
				if fieldNum == 1 {
					var stringLenmapkey uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowComplianceOperatorV2
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapkey |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapkey := int(stringLenmapkey)
					if intStringLenmapkey < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					postStringIndexmapkey := iNdEx + intStringLenmapkey
					if postStringIndexmapkey < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if postStringIndexmapkey > l {
						return io.ErrUnexpectedEOF
					}
					mapkey = string(dAtA[iNdEx:postStringIndexmapkey])
					iNdEx = postStringIndexmapkey
				} else if fieldNum == 2 {
					var stringLenmapvalue uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowComplianceOperatorV2
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapvalue |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapvalue := int(stringLenmapvalue)
					if intStringLenmapvalue < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					postStringIndexmapvalue := iNdEx + intStringLenmapvalue
					if postStringIndexmapvalue < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if postStringIndexmapvalue > l {
						return io.ErrUnexpectedEOF
					}
					mapvalue = string(dAtA[iNdEx:postStringIndexmapvalue])
					iNdEx = postStringIndexmapvalue
				} else {
					iNdEx = entryPreIndex
					skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
					if err != nil {
						return err
					}
					if (skippy < 0) || (iNdEx+skippy) < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if (iNdEx + skippy) > postIndex {
						return io.ErrUnexpectedEOF
					}
					iNdEx += skippy
				}
			}
			m.Annotations[mapkey] = mapvalue
			iNdEx = postIndex
		case 9:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Description", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Description = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 10:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Rules", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Rules = append(m.Rules, &ComplianceOperatorProfileV2_Rule{})
			if err := m.Rules[len(m.Rules)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 11:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Product", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Product = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 12:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Title", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Title = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 13:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Values", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Values = append(m.Values, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ComplianceOperatorProfileV2_Rule) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowComplianceOperatorV2
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Rule: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Rule: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RuleName", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.RuleName = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ComplianceOperatorRuleV2) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowComplianceOperatorV2
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ComplianceOperatorRuleV2: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ComplianceOperatorRuleV2: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Id = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RuleId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.RuleId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Name", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Name = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RuleType", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.RuleType = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Severity", wireType)
			}
			m.Severity = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Severity |= RuleSeverity(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Labels", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Labels == nil {
				m.Labels = make(map[string]string)
			}
			var mapkey string
			var mapvalue string
			for iNdEx < postIndex {
				entryPreIndex := iNdEx
				var wire uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowComplianceOperatorV2
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					wire |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				fieldNum := int32(wire >> 3)
				if fieldNum == 1 {
					var stringLenmapkey uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowComplianceOperatorV2
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapkey |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapkey := int(stringLenmapkey)
					if intStringLenmapkey < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					postStringIndexmapkey := iNdEx + intStringLenmapkey
					if postStringIndexmapkey < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if postStringIndexmapkey > l {
						return io.ErrUnexpectedEOF
					}
					mapkey = string(dAtA[iNdEx:postStringIndexmapkey])
					iNdEx = postStringIndexmapkey
				} else if fieldNum == 2 {
					var stringLenmapvalue uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowComplianceOperatorV2
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapvalue |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapvalue := int(stringLenmapvalue)
					if intStringLenmapvalue < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					postStringIndexmapvalue := iNdEx + intStringLenmapvalue
					if postStringIndexmapvalue < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if postStringIndexmapvalue > l {
						return io.ErrUnexpectedEOF
					}
					mapvalue = string(dAtA[iNdEx:postStringIndexmapvalue])
					iNdEx = postStringIndexmapvalue
				} else {
					iNdEx = entryPreIndex
					skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
					if err != nil {
						return err
					}
					if (skippy < 0) || (iNdEx+skippy) < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if (iNdEx + skippy) > postIndex {
						return io.ErrUnexpectedEOF
					}
					iNdEx += skippy
				}
			}
			m.Labels[mapkey] = mapvalue
			iNdEx = postIndex
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Annotations", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Annotations == nil {
				m.Annotations = make(map[string]string)
			}
			var mapkey string
			var mapvalue string
			for iNdEx < postIndex {
				entryPreIndex := iNdEx
				var wire uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowComplianceOperatorV2
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					wire |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				fieldNum := int32(wire >> 3)
				if fieldNum == 1 {
					var stringLenmapkey uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowComplianceOperatorV2
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapkey |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapkey := int(stringLenmapkey)
					if intStringLenmapkey < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					postStringIndexmapkey := iNdEx + intStringLenmapkey
					if postStringIndexmapkey < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if postStringIndexmapkey > l {
						return io.ErrUnexpectedEOF
					}
					mapkey = string(dAtA[iNdEx:postStringIndexmapkey])
					iNdEx = postStringIndexmapkey
				} else if fieldNum == 2 {
					var stringLenmapvalue uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowComplianceOperatorV2
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapvalue |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapvalue := int(stringLenmapvalue)
					if intStringLenmapvalue < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					postStringIndexmapvalue := iNdEx + intStringLenmapvalue
					if postStringIndexmapvalue < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if postStringIndexmapvalue > l {
						return io.ErrUnexpectedEOF
					}
					mapvalue = string(dAtA[iNdEx:postStringIndexmapvalue])
					iNdEx = postStringIndexmapvalue
				} else {
					iNdEx = entryPreIndex
					skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
					if err != nil {
						return err
					}
					if (skippy < 0) || (iNdEx+skippy) < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if (iNdEx + skippy) > postIndex {
						return io.ErrUnexpectedEOF
					}
					iNdEx += skippy
				}
			}
			m.Annotations[mapkey] = mapvalue
			iNdEx = postIndex
		case 8:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Title", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Title = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 9:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Description", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Description = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 10:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Rationale", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Rationale = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 11:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Fixes", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Fixes = append(m.Fixes, &ComplianceOperatorRuleV2_Fix{})
			if err := m.Fixes[len(m.Fixes)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 12:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Warning", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Warning = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 13:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Controls", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Controls = append(m.Controls, &RuleControls{})
			if err := m.Controls[len(m.Controls)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 14:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ClusterId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ClusterId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ComplianceOperatorRuleV2_Fix) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowComplianceOperatorV2
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Fix: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Fix: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Platform", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Platform = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Disruption", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Disruption = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *RuleControls) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowComplianceOperatorV2
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: RuleControls: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: RuleControls: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Standard", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Standard = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Controls", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Controls = append(m.Controls, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ComplianceOperatorScanConfigurationV2) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowComplianceOperatorV2
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ComplianceOperatorScanConfigurationV2: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ComplianceOperatorScanConfigurationV2: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Id = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ScanConfigName", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ScanConfigName = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field AutoApplyRemediations", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.AutoApplyRemediations = bool(v != 0)
		case 4:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field AutoUpdateRemediations", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.AutoUpdateRemediations = bool(v != 0)
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field OneTimeScan", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.OneTimeScan = bool(v != 0)
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Labels", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Labels == nil {
				m.Labels = make(map[string]string)
			}
			var mapkey string
			var mapvalue string
			for iNdEx < postIndex {
				entryPreIndex := iNdEx
				var wire uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowComplianceOperatorV2
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					wire |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				fieldNum := int32(wire >> 3)
				if fieldNum == 1 {
					var stringLenmapkey uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowComplianceOperatorV2
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapkey |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapkey := int(stringLenmapkey)
					if intStringLenmapkey < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					postStringIndexmapkey := iNdEx + intStringLenmapkey
					if postStringIndexmapkey < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if postStringIndexmapkey > l {
						return io.ErrUnexpectedEOF
					}
					mapkey = string(dAtA[iNdEx:postStringIndexmapkey])
					iNdEx = postStringIndexmapkey
				} else if fieldNum == 2 {
					var stringLenmapvalue uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowComplianceOperatorV2
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapvalue |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapvalue := int(stringLenmapvalue)
					if intStringLenmapvalue < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					postStringIndexmapvalue := iNdEx + intStringLenmapvalue
					if postStringIndexmapvalue < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if postStringIndexmapvalue > l {
						return io.ErrUnexpectedEOF
					}
					mapvalue = string(dAtA[iNdEx:postStringIndexmapvalue])
					iNdEx = postStringIndexmapvalue
				} else {
					iNdEx = entryPreIndex
					skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
					if err != nil {
						return err
					}
					if (skippy < 0) || (iNdEx+skippy) < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if (iNdEx + skippy) > postIndex {
						return io.ErrUnexpectedEOF
					}
					iNdEx += skippy
				}
			}
			m.Labels[mapkey] = mapvalue
			iNdEx = postIndex
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Annotations", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Annotations == nil {
				m.Annotations = make(map[string]string)
			}
			var mapkey string
			var mapvalue string
			for iNdEx < postIndex {
				entryPreIndex := iNdEx
				var wire uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowComplianceOperatorV2
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					wire |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				fieldNum := int32(wire >> 3)
				if fieldNum == 1 {
					var stringLenmapkey uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowComplianceOperatorV2
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapkey |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapkey := int(stringLenmapkey)
					if intStringLenmapkey < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					postStringIndexmapkey := iNdEx + intStringLenmapkey
					if postStringIndexmapkey < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if postStringIndexmapkey > l {
						return io.ErrUnexpectedEOF
					}
					mapkey = string(dAtA[iNdEx:postStringIndexmapkey])
					iNdEx = postStringIndexmapkey
				} else if fieldNum == 2 {
					var stringLenmapvalue uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowComplianceOperatorV2
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapvalue |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapvalue := int(stringLenmapvalue)
					if intStringLenmapvalue < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					postStringIndexmapvalue := iNdEx + intStringLenmapvalue
					if postStringIndexmapvalue < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if postStringIndexmapvalue > l {
						return io.ErrUnexpectedEOF
					}
					mapvalue = string(dAtA[iNdEx:postStringIndexmapvalue])
					iNdEx = postStringIndexmapvalue
				} else {
					iNdEx = entryPreIndex
					skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
					if err != nil {
						return err
					}
					if (skippy < 0) || (iNdEx+skippy) < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if (iNdEx + skippy) > postIndex {
						return io.ErrUnexpectedEOF
					}
					iNdEx += skippy
				}
			}
			m.Annotations[mapkey] = mapvalue
			iNdEx = postIndex
		case 8:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Profiles", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Profiles = append(m.Profiles, &ProfileShim{})
			if err := m.Profiles[len(m.Profiles)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 9:
			if wireType == 0 {
				var v NodeRole
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowComplianceOperatorV2
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					v |= NodeRole(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				m.NodeRoles = append(m.NodeRoles, v)
			} else if wireType == 2 {
				var packedLen int
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowComplianceOperatorV2
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					packedLen |= int(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				if packedLen < 0 {
					return ErrInvalidLengthComplianceOperatorV2
				}
				postIndex := iNdEx + packedLen
				if postIndex < 0 {
					return ErrInvalidLengthComplianceOperatorV2
				}
				if postIndex > l {
					return io.ErrUnexpectedEOF
				}
				var elementCount int
				if elementCount != 0 && len(m.NodeRoles) == 0 {
					m.NodeRoles = make([]NodeRole, 0, elementCount)
				}
				for iNdEx < postIndex {
					var v NodeRole
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowComplianceOperatorV2
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						v |= NodeRole(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					m.NodeRoles = append(m.NodeRoles, v)
				}
			} else {
				return fmt.Errorf("proto: wrong wireType = %d for field NodeRoles", wireType)
			}
		case 10:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field StrictNodeScan", wireType)
			}
			var v int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				v |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			m.StrictNodeScan = bool(v != 0)
		case 11:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Schedule", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Schedule == nil {
				m.Schedule = &Schedule{}
			}
			if err := m.Schedule.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 12:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field CreatedTime", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.CreatedTime == nil {
				m.CreatedTime = &types.Timestamp{}
			}
			if err := m.CreatedTime.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 13:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field LastUpdatedTime", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.LastUpdatedTime == nil {
				m.LastUpdatedTime = &types.Timestamp{}
			}
			if err := m.LastUpdatedTime.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 14:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ModifiedBy", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.ModifiedBy == nil {
				m.ModifiedBy = &SlimUser{}
			}
			if err := m.ModifiedBy.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 15:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Description", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Description = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 16:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Clusters", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Clusters = append(m.Clusters, &ComplianceOperatorScanConfigurationV2_Cluster{})
			if err := m.Clusters[len(m.Clusters)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ComplianceOperatorScanConfigurationV2_Cluster) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowComplianceOperatorV2
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: Cluster: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Cluster: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ClusterId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ClusterId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ComplianceOperatorClusterScanConfigStatus) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowComplianceOperatorV2
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ComplianceOperatorClusterScanConfigStatus: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ComplianceOperatorClusterScanConfigStatus: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ClusterId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ClusterId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ScanConfigId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ScanConfigId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Errors", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Errors = append(m.Errors, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field LastUpdatedTime", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.LastUpdatedTime == nil {
				m.LastUpdatedTime = &types.Timestamp{}
			}
			if err := m.LastUpdatedTime.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ClusterName", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ClusterName = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Id = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ComplianceOperatorCheckResultV2) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowComplianceOperatorV2
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ComplianceOperatorCheckResultV2: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ComplianceOperatorCheckResultV2: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Id = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field CheckId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.CheckId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field CheckName", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.CheckName = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ClusterId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ClusterId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Status", wireType)
			}
			m.Status = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Status |= ComplianceOperatorCheckResultV2_CheckStatus(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 6:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Severity", wireType)
			}
			m.Severity = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Severity |= RuleSeverity(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Description", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Description = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 8:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Instructions", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Instructions = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 9:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Labels", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Labels == nil {
				m.Labels = make(map[string]string)
			}
			var mapkey string
			var mapvalue string
			for iNdEx < postIndex {
				entryPreIndex := iNdEx
				var wire uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowComplianceOperatorV2
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					wire |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				fieldNum := int32(wire >> 3)
				if fieldNum == 1 {
					var stringLenmapkey uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowComplianceOperatorV2
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapkey |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapkey := int(stringLenmapkey)
					if intStringLenmapkey < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					postStringIndexmapkey := iNdEx + intStringLenmapkey
					if postStringIndexmapkey < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if postStringIndexmapkey > l {
						return io.ErrUnexpectedEOF
					}
					mapkey = string(dAtA[iNdEx:postStringIndexmapkey])
					iNdEx = postStringIndexmapkey
				} else if fieldNum == 2 {
					var stringLenmapvalue uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowComplianceOperatorV2
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapvalue |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapvalue := int(stringLenmapvalue)
					if intStringLenmapvalue < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					postStringIndexmapvalue := iNdEx + intStringLenmapvalue
					if postStringIndexmapvalue < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if postStringIndexmapvalue > l {
						return io.ErrUnexpectedEOF
					}
					mapvalue = string(dAtA[iNdEx:postStringIndexmapvalue])
					iNdEx = postStringIndexmapvalue
				} else {
					iNdEx = entryPreIndex
					skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
					if err != nil {
						return err
					}
					if (skippy < 0) || (iNdEx+skippy) < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if (iNdEx + skippy) > postIndex {
						return io.ErrUnexpectedEOF
					}
					iNdEx += skippy
				}
			}
			m.Labels[mapkey] = mapvalue
			iNdEx = postIndex
		case 10:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Annotations", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Annotations == nil {
				m.Annotations = make(map[string]string)
			}
			var mapkey string
			var mapvalue string
			for iNdEx < postIndex {
				entryPreIndex := iNdEx
				var wire uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowComplianceOperatorV2
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					wire |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				fieldNum := int32(wire >> 3)
				if fieldNum == 1 {
					var stringLenmapkey uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowComplianceOperatorV2
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapkey |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapkey := int(stringLenmapkey)
					if intStringLenmapkey < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					postStringIndexmapkey := iNdEx + intStringLenmapkey
					if postStringIndexmapkey < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if postStringIndexmapkey > l {
						return io.ErrUnexpectedEOF
					}
					mapkey = string(dAtA[iNdEx:postStringIndexmapkey])
					iNdEx = postStringIndexmapkey
				} else if fieldNum == 2 {
					var stringLenmapvalue uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowComplianceOperatorV2
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapvalue |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapvalue := int(stringLenmapvalue)
					if intStringLenmapvalue < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					postStringIndexmapvalue := iNdEx + intStringLenmapvalue
					if postStringIndexmapvalue < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if postStringIndexmapvalue > l {
						return io.ErrUnexpectedEOF
					}
					mapvalue = string(dAtA[iNdEx:postStringIndexmapvalue])
					iNdEx = postStringIndexmapvalue
				} else {
					iNdEx = entryPreIndex
					skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
					if err != nil {
						return err
					}
					if (skippy < 0) || (iNdEx+skippy) < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if (iNdEx + skippy) > postIndex {
						return io.ErrUnexpectedEOF
					}
					iNdEx += skippy
				}
			}
			m.Annotations[mapkey] = mapvalue
			iNdEx = postIndex
		case 11:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field CreatedTime", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.CreatedTime == nil {
				m.CreatedTime = &types.Timestamp{}
			}
			if err := m.CreatedTime.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 12:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Standard", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Standard = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 13:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Control", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Control = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 14:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ScanName", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ScanName = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 15:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ClusterName", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ClusterName = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 16:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ScanConfigName", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ScanConfigName = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 17:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Rationale", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Rationale = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 18:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ValuesUsed", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ValuesUsed = append(m.ValuesUsed, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		case 19:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Warnings", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Warnings = append(m.Warnings, string(dAtA[iNdEx:postIndex]))
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ScanStatus) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowComplianceOperatorV2
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ScanStatus: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ScanStatus: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Phase", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Phase = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Result", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Result = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Warnings", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Warnings = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ComplianceOperatorScanV2) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowComplianceOperatorV2
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ComplianceOperatorScanV2: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ComplianceOperatorScanV2: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Id = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ScanConfigName", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ScanConfigName = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ClusterId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ClusterId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Errors", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Errors = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Profile", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Profile == nil {
				m.Profile = &ProfileShim{}
			}
			if err := m.Profile.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 6:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Labels", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Labels == nil {
				m.Labels = make(map[string]string)
			}
			var mapkey string
			var mapvalue string
			for iNdEx < postIndex {
				entryPreIndex := iNdEx
				var wire uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowComplianceOperatorV2
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					wire |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				fieldNum := int32(wire >> 3)
				if fieldNum == 1 {
					var stringLenmapkey uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowComplianceOperatorV2
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapkey |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapkey := int(stringLenmapkey)
					if intStringLenmapkey < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					postStringIndexmapkey := iNdEx + intStringLenmapkey
					if postStringIndexmapkey < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if postStringIndexmapkey > l {
						return io.ErrUnexpectedEOF
					}
					mapkey = string(dAtA[iNdEx:postStringIndexmapkey])
					iNdEx = postStringIndexmapkey
				} else if fieldNum == 2 {
					var stringLenmapvalue uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowComplianceOperatorV2
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapvalue |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapvalue := int(stringLenmapvalue)
					if intStringLenmapvalue < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					postStringIndexmapvalue := iNdEx + intStringLenmapvalue
					if postStringIndexmapvalue < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if postStringIndexmapvalue > l {
						return io.ErrUnexpectedEOF
					}
					mapvalue = string(dAtA[iNdEx:postStringIndexmapvalue])
					iNdEx = postStringIndexmapvalue
				} else {
					iNdEx = entryPreIndex
					skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
					if err != nil {
						return err
					}
					if (skippy < 0) || (iNdEx+skippy) < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if (iNdEx + skippy) > postIndex {
						return io.ErrUnexpectedEOF
					}
					iNdEx += skippy
				}
			}
			m.Labels[mapkey] = mapvalue
			iNdEx = postIndex
		case 7:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Annotations", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Annotations == nil {
				m.Annotations = make(map[string]string)
			}
			var mapkey string
			var mapvalue string
			for iNdEx < postIndex {
				entryPreIndex := iNdEx
				var wire uint64
				for shift := uint(0); ; shift += 7 {
					if shift >= 64 {
						return ErrIntOverflowComplianceOperatorV2
					}
					if iNdEx >= l {
						return io.ErrUnexpectedEOF
					}
					b := dAtA[iNdEx]
					iNdEx++
					wire |= uint64(b&0x7F) << shift
					if b < 0x80 {
						break
					}
				}
				fieldNum := int32(wire >> 3)
				if fieldNum == 1 {
					var stringLenmapkey uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowComplianceOperatorV2
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapkey |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapkey := int(stringLenmapkey)
					if intStringLenmapkey < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					postStringIndexmapkey := iNdEx + intStringLenmapkey
					if postStringIndexmapkey < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if postStringIndexmapkey > l {
						return io.ErrUnexpectedEOF
					}
					mapkey = string(dAtA[iNdEx:postStringIndexmapkey])
					iNdEx = postStringIndexmapkey
				} else if fieldNum == 2 {
					var stringLenmapvalue uint64
					for shift := uint(0); ; shift += 7 {
						if shift >= 64 {
							return ErrIntOverflowComplianceOperatorV2
						}
						if iNdEx >= l {
							return io.ErrUnexpectedEOF
						}
						b := dAtA[iNdEx]
						iNdEx++
						stringLenmapvalue |= uint64(b&0x7F) << shift
						if b < 0x80 {
							break
						}
					}
					intStringLenmapvalue := int(stringLenmapvalue)
					if intStringLenmapvalue < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					postStringIndexmapvalue := iNdEx + intStringLenmapvalue
					if postStringIndexmapvalue < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if postStringIndexmapvalue > l {
						return io.ErrUnexpectedEOF
					}
					mapvalue = string(dAtA[iNdEx:postStringIndexmapvalue])
					iNdEx = postStringIndexmapvalue
				} else {
					iNdEx = entryPreIndex
					skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
					if err != nil {
						return err
					}
					if (skippy < 0) || (iNdEx+skippy) < 0 {
						return ErrInvalidLengthComplianceOperatorV2
					}
					if (iNdEx + skippy) > postIndex {
						return io.ErrUnexpectedEOF
					}
					iNdEx += skippy
				}
			}
			m.Annotations[mapkey] = mapvalue
			iNdEx = postIndex
		case 8:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field ScanType", wireType)
			}
			m.ScanType = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.ScanType |= ScanType(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 9:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field NodeSelector", wireType)
			}
			m.NodeSelector = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.NodeSelector |= NodeRole(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 10:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Status", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Status == nil {
				m.Status = &ScanStatus{}
			}
			if err := m.Status.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 11:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field CreatedTime", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.CreatedTime == nil {
				m.CreatedTime = &types.Timestamp{}
			}
			if err := m.CreatedTime.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 12:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field LastExecutedTime", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.LastExecutedTime == nil {
				m.LastExecutedTime = &types.Timestamp{}
			}
			if err := m.LastExecutedTime.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 13:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ScanName", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ScanName = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 14:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Warnings", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Warnings = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ComplianceOperatorProfileClusterEdge) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowComplianceOperatorV2
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ComplianceOperatorProfileClusterEdge: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ComplianceOperatorProfileClusterEdge: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Id", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Id = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ProfileId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ProfileId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ProfileUid", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ProfileUid = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 4:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ClusterId", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ClusterId = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipComplianceOperatorV2(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthComplianceOperatorV2
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipComplianceOperatorV2(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowComplianceOperatorV2
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowComplianceOperatorV2
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthComplianceOperatorV2
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupComplianceOperatorV2
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthComplianceOperatorV2
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthComplianceOperatorV2        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowComplianceOperatorV2          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupComplianceOperatorV2 = fmt.Errorf("proto: unexpected end of group")
)
