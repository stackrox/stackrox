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
	return fileDescriptor_26c2ec1f62102154, []int{5, 0}
}

type ProfileShim struct {
	ProfileId            string   `protobuf:"bytes,1,opt,name=profile_id,json=profileId,proto3" json:"profile_id,omitempty" search:"-" sql:"fk(ComplianceOperatorProfileV2:id),no-fk-constraint"`
	ProfileName          string   `protobuf:"bytes,2,opt,name=profile_name,json=profileName,proto3" json:"profile_name,omitempty"`
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

func (m *ProfileShim) GetProfileName() string {
	if m != nil {
		return m.ProfileName
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
	Id                   string                              `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty" sql:"pk"`
	ProfileId            string                              `protobuf:"bytes,2,opt,name=profile_id,json=profileId,proto3" json:"profile_id,omitempty"`
	Name                 string                              `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty" search:"Compliance Profile Name,hidden,store" sql:"index=category:unique;name:profile_unique_indicator"`
<<<<<<< HEAD
	ProfileVersion       string                              `protobuf:"bytes,4,opt,name=profile_version,json=profileVersion,proto3" json:"profile_version,omitempty" search:"Compliance Profile Version,hidden,store" sql:"index=category:unique;name:profile_unique_indicator"`
	ProductType          []string                            `protobuf:"bytes,5,rep,name=product_type,json=productType,proto3" json:"product_type,omitempty" search:"Compliance Profile Product Type,hidden,store"`
	Standard             string                              `protobuf:"bytes,6,opt,name=standard,proto3" json:"standard,omitempty" search:"Compliance Standard,hidden,store"`
	Labels               map[string]string                   `protobuf:"bytes,7,rep,name=labels,proto3" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Annotations          map[string]string                   `protobuf:"bytes,8,rep,name=annotations,proto3" json:"annotations,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Description          string                              `protobuf:"bytes,9,opt,name=description,proto3" json:"description,omitempty"`
	Rules                []*ComplianceOperatorProfileV2_Rule `protobuf:"bytes,10,rep,name=rules,proto3" json:"rules,omitempty"`
	Product              string                              `protobuf:"bytes,11,opt,name=product,proto3" json:"product,omitempty"`
	Title                string                              `protobuf:"bytes,12,opt,name=title,proto3" json:"title,omitempty"`
=======
	OperatorVersion      string                              `protobuf:"bytes,4,opt,name=operator_version,json=operatorVersion,proto3" json:"operator_version,omitempty"`
	ProfileVersion       string                              `protobuf:"bytes,5,opt,name=profile_version,json=profileVersion,proto3" json:"profile_version,omitempty" search:"Compliance Profile Version,hidden,store" sql:"index=category:unique;name:profile_unique_indicator"`
	ProductType          []string                            `protobuf:"bytes,6,rep,name=product_type,json=productType,proto3" json:"product_type,omitempty" search:"Compliance Profile Product Type,hidden,store"`
	Standard             string                              `protobuf:"bytes,7,opt,name=standard,proto3" json:"standard,omitempty" search:"Compliance Standard,hidden,store"`
	Labels               map[string]string                   `protobuf:"bytes,8,rep,name=labels,proto3" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Annotations          map[string]string                   `protobuf:"bytes,9,rep,name=annotations,proto3" json:"annotations,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Description          string                              `protobuf:"bytes,10,opt,name=description,proto3" json:"description,omitempty"`
	Rules                []*ComplianceOperatorProfileV2_Rule `protobuf:"bytes,11,rep,name=rules,proto3" json:"rules,omitempty"`
	Product              string                              `protobuf:"bytes,12,opt,name=product,proto3" json:"product,omitempty"`
>>>>>>> 4aa7f423fb (X-Smart-Squash: Squashed 5 commits:)
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

<<<<<<< HEAD
=======
func (m *ComplianceOperatorProfileV2) GetOperatorVersion() string {
	if m != nil {
		return m.OperatorVersion
	}
	return ""
}

>>>>>>> 4aa7f423fb (X-Smart-Squash: Squashed 5 commits:)
func (m *ComplianceOperatorProfileV2) GetProfileVersion() string {
	if m != nil {
		return m.ProfileVersion
	}
	return ""
}

func (m *ComplianceOperatorProfileV2) GetProductType() []string {
	if m != nil {
		return m.ProductType
	}
	return nil
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

func (m *ComplianceOperatorProfileV2) MessageClone() proto.Message {
	return m.Clone()
}
func (m *ComplianceOperatorProfileV2) Clone() *ComplianceOperatorProfileV2 {
	if m == nil {
		return nil
	}
	cloned := new(ComplianceOperatorProfileV2)
	*cloned = *m

	if m.ProductType != nil {
		cloned.ProductType = make([]string, len(m.ProductType))
		copy(cloned.ProductType, m.ProductType)
	}
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

// Next Tag: 12
type ComplianceOperatorRuleV2 struct {
	Name                 string            `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty" search:"Compliance Rule Name,hidden,store" sql:"pk"`
	OperatorVersion      string            `protobuf:"bytes,2,opt,name=operator_version,json=operatorVersion,proto3" json:"operator_version,omitempty" search:"Compliance Operator Version,hidden,store"`
	RuleVersion          string            `protobuf:"bytes,3,opt,name=rule_version,json=ruleVersion,proto3" json:"rule_version,omitempty" search:"Compliance Rule Version,hidden,store"`
	RuleType             string            `protobuf:"bytes,4,opt,name=rule_type,json=ruleType,proto3" json:"rule_type,omitempty" search:"Compliance Rule Type,hidden,store"`
	Severity             RuleSeverity      `protobuf:"varint,5,opt,name=severity,proto3,enum=storage.RuleSeverity" json:"severity,omitempty" search:"Compliance Rule Severity,hidden,store"`
	Labels               map[string]string `protobuf:"bytes,6,rep,name=labels,proto3" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Annotations          map[string]string `protobuf:"bytes,7,rep,name=annotations,proto3" json:"annotations,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Title                string            `protobuf:"bytes,8,opt,name=title,proto3" json:"title,omitempty"`
	Description          string            `protobuf:"bytes,9,opt,name=description,proto3" json:"description,omitempty"`
	Rationale            string            `protobuf:"bytes,10,opt,name=rationale,proto3" json:"rationale,omitempty"`
	Fixes                string            `protobuf:"bytes,11,opt,name=fixes,proto3" json:"fixes,omitempty"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
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

func (m *ComplianceOperatorRuleV2) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *ComplianceOperatorRuleV2) GetOperatorVersion() string {
	if m != nil {
		return m.OperatorVersion
	}
	return ""
}

func (m *ComplianceOperatorRuleV2) GetRuleVersion() string {
	if m != nil {
		return m.RuleVersion
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

func (m *ComplianceOperatorRuleV2) GetFixes() string {
	if m != nil {
		return m.Fixes
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
	return cloned
}

// Next Tag: 15
type ComplianceOperatorScanConfigurationV2 struct {
	Id                     string            `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty" search:"Compliance Scan Config ID,hidden,store" sql:"pk,type(uuid)"`
	ScanName               string            `protobuf:"bytes,2,opt,name=scan_name,json=scanName,proto3" json:"scan_name,omitempty" search:"Compliance Scan Name,store" sql:"unique"`
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
	ModifiedBy           *SlimUser `protobuf:"bytes,14,opt,name=modified_by,json=modifiedBy,proto3" json:"modified_by,omitempty" sql:"ignore_labels(User ID)"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *ComplianceOperatorScanConfigurationV2) Reset()         { *m = ComplianceOperatorScanConfigurationV2{} }
func (m *ComplianceOperatorScanConfigurationV2) String() string { return proto.CompactTextString(m) }
func (*ComplianceOperatorScanConfigurationV2) ProtoMessage()    {}
func (*ComplianceOperatorScanConfigurationV2) Descriptor() ([]byte, []int) {
	return fileDescriptor_26c2ec1f62102154, []int{3}
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

func (m *ComplianceOperatorScanConfigurationV2) GetScanName() string {
	if m != nil {
		return m.ScanName
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
	return cloned
}

// Next Tag: 6
// Cluster and an error if necessary to handle cases where the scan configuration is
// unable to be applied to a cluster for whatever reason.
type ComplianceOperatorClusterScanConfigStatus struct {
	ClusterId            string           `protobuf:"bytes,1,opt,name=cluster_id,json=clusterId,proto3" json:"cluster_id,omitempty" search:"Cluster ID,hidden,store" sql:"pk,fk(Cluster:id),no-fk-constraint,type(uuid)"`
	ScanId               string           `protobuf:"bytes,2,opt,name=scan_id,json=scanId,proto3" json:"scan_id,omitempty" search:"Compliance Scan Config ID,hidden,store" sql:"fk(ComplianceOperatorScanConfigurationV2:id),no-fk-constraint"`
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
	return fileDescriptor_26c2ec1f62102154, []int{4}
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

func (m *ComplianceOperatorClusterScanConfigStatus) GetClusterId() string {
	if m != nil {
		return m.ClusterId
	}
	return ""
}

func (m *ComplianceOperatorClusterScanConfigStatus) GetScanId() string {
	if m != nil {
		return m.ScanId
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

// Next Tag: 18
// This object has been flattened vs joining with rule.  The rationale is to spend the time to query rule
// while processing results vs reporting them to the user.  Additionally, flattening it helps with the historical data
// as the rules can change without impacting the historical result.
type ComplianceOperatorCheckResultV2 struct {
	Id                   string                                      `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty" sql:"pk"`
	CheckId              string                                      `protobuf:"bytes,2,opt,name=check_id,json=checkId,proto3" json:"check_id,omitempty" search:"Compliance Check ID,hidden,store"`
	CheckName            string                                      `protobuf:"bytes,3,opt,name=check_name,json=checkName,proto3" json:"check_name,omitempty" search:"Compliance Check Name,hidden,store"`
	ClusterId            string                                      `protobuf:"bytes,4,opt,name=cluster_id,json=clusterId,proto3" json:"cluster_id,omitempty" search:"Cluster ID,hidden,store" sql:"fk(Cluster:id),no-fk-constraint,type(uuid)"`
	ClusterName          string                                      `protobuf:"bytes,15,opt,name=cluster_name,json=clusterName,proto3" json:"cluster_name,omitempty"`
	Status               ComplianceOperatorCheckResultV2_CheckStatus `protobuf:"varint,5,opt,name=status,proto3,enum=storage.ComplianceOperatorCheckResultV2_CheckStatus" json:"status,omitempty" search:"Compliance Check Status,hidden,store"`
	Severity             RuleSeverity                                `protobuf:"varint,6,opt,name=severity,proto3,enum=storage.RuleSeverity" json:"severity,omitempty" search:"Compliance Rule Severity,hidden,store"`
	Description          string                                      `protobuf:"bytes,7,opt,name=description,proto3" json:"description,omitempty"`
	Instructions         string                                      `protobuf:"bytes,8,opt,name=instructions,proto3" json:"instructions,omitempty"`
	Labels               map[string]string                           `protobuf:"bytes,9,rep,name=labels,proto3" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	Annotations          map[string]string                           `protobuf:"bytes,10,rep,name=annotations,proto3" json:"annotations,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	CreatedTime          *types.Timestamp                            `protobuf:"bytes,11,opt,name=created_time,json=createdTime,proto3" json:"created_time,omitempty" search:"Compliance Check Result Created Time,hidden"`
	Standard             string                                      `protobuf:"bytes,12,opt,name=standard,proto3" json:"standard,omitempty" search:"Compliance Standard,hidden,store"`
	Control              string                                      `protobuf:"bytes,13,opt,name=control,proto3" json:"control,omitempty"`
	ScanId               string                                      `protobuf:"bytes,14,opt,name=scan_id,json=scanId,proto3" json:"scan_id,omitempty" search:"-" sql:"fk(ComplianceOperatorScanV2:id),no-fk-constraint"`
	ScanConfigId         string                                      `protobuf:"bytes,16,opt,name=scan_config_id,json=scanConfigId,proto3" json:"scan_config_id,omitempty" search:"Compliance Scan Config ID,hidden,store" sql:"fk(ComplianceOperatorScanConfigurationV2:id),no-fk-constraint,type(uuid)"`
	ScanConfigName       string                                      `protobuf:"bytes,17,opt,name=scan_config_name,json=scanConfigName,proto3" json:"scan_config_name,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                                    `json:"-"`
	XXX_unrecognized     []byte                                      `json:"-"`
	XXX_sizecache        int32                                       `json:"-"`
}

func (m *ComplianceOperatorCheckResultV2) Reset()         { *m = ComplianceOperatorCheckResultV2{} }
func (m *ComplianceOperatorCheckResultV2) String() string { return proto.CompactTextString(m) }
func (*ComplianceOperatorCheckResultV2) ProtoMessage()    {}
func (*ComplianceOperatorCheckResultV2) Descriptor() ([]byte, []int) {
	return fileDescriptor_26c2ec1f62102154, []int{5}
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

func (m *ComplianceOperatorCheckResultV2) GetScanId() string {
	if m != nil {
		return m.ScanId
	}
	return ""
}

func (m *ComplianceOperatorCheckResultV2) GetScanConfigId() string {
	if m != nil {
		return m.ScanConfigId
	}
	return ""
}

func (m *ComplianceOperatorCheckResultV2) GetScanConfigName() string {
	if m != nil {
		return m.ScanConfigName
	}
	return ""
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
	return cloned
}

type ScanStatus struct {
	Phase                string   `protobuf:"bytes,1,opt,name=phase,proto3" json:"phase,omitempty"`
	Result               string   `protobuf:"bytes,2,opt,name=result,proto3" json:"result,omitempty"`
	Warnings             []string `protobuf:"bytes,3,rep,name=warnings,proto3" json:"warnings,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ScanStatus) Reset()         { *m = ScanStatus{} }
func (m *ScanStatus) String() string { return proto.CompactTextString(m) }
func (*ScanStatus) ProtoMessage()    {}
func (*ScanStatus) Descriptor() ([]byte, []int) {
	return fileDescriptor_26c2ec1f62102154, []int{6}
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

func (m *ScanStatus) GetWarnings() []string {
	if m != nil {
		return m.Warnings
	}
	return nil
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

	if m.Warnings != nil {
		cloned.Warnings = make([]string, len(m.Warnings))
		copy(cloned.Warnings, m.Warnings)
	}
	return cloned
}

// Next Tag: 15
// Scan object per cluster
type ComplianceOperatorScanV2 struct {
	Id                   string            `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty" sql:"pk"`
	ScanConfigId         string            `protobuf:"bytes,2,opt,name=scan_config_id,json=scanConfigId,proto3" json:"scan_config_id,omitempty" search:"-" sql:"fk(ComplianceOperatorScanConfigurationV2:id),no-fk-constraint,type(uuid)"`
	ScanName             string            `protobuf:"bytes,14,opt,name=scan_name,json=scanName,proto3" json:"scan_name,omitempty"`
	ClusterId            string            `protobuf:"bytes,3,opt,name=cluster_id,json=clusterId,proto3" json:"cluster_id,omitempty" search:"Cluster ID,hidden,store" sql:"fk(Cluster:id),no-fk-constraint,type(uuid),index=category:unique;name:scan_unique_indicator"`
	ClusterName          string            `protobuf:"bytes,13,opt,name=cluster_name,json=clusterName,proto3" json:"cluster_name,omitempty"`
	Errors               []string          `protobuf:"bytes,4,rep,name=errors,proto3" json:"errors,omitempty"`
	Profile              []*ProfileShim    `protobuf:"bytes,5,rep,name=profile,proto3" json:"profile,omitempty"`
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
	return fileDescriptor_26c2ec1f62102154, []int{7}
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

func (m *ComplianceOperatorScanV2) GetScanConfigId() string {
	if m != nil {
		return m.ScanConfigId
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

func (m *ComplianceOperatorScanV2) GetClusterName() string {
	if m != nil {
		return m.ClusterName
	}
	return ""
}

func (m *ComplianceOperatorScanV2) GetErrors() []string {
	if m != nil {
		return m.Errors
	}
	return nil
}

func (m *ComplianceOperatorScanV2) GetProfile() []*ProfileShim {
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

	if m.Errors != nil {
		cloned.Errors = make([]string, len(m.Errors))
		copy(cloned.Errors, m.Errors)
	}
	if m.Profile != nil {
		cloned.Profile = make([]*ProfileShim, len(m.Profile))
		for idx, v := range m.Profile {
			cloned.Profile[idx] = v.Clone()
		}
	}
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
	ProfileId            string   `protobuf:"bytes,2,opt,name=profile_id,json=profileId,proto3" json:"profile_id,omitempty" search:"Compliance Profile ID,store,hidden" sql:"fk(ComplianceOperatorProfileV2:id)"`
	ClusterId            string   `protobuf:"bytes,3,opt,name=cluster_id,json=clusterId,proto3" json:"cluster_id,omitempty" search:"Cluster ID,hidden,store" sql:"fk(Cluster:id),no-fk-constraint,type(uuid)"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ComplianceOperatorProfileClusterEdge) Reset()         { *m = ComplianceOperatorProfileClusterEdge{} }
func (m *ComplianceOperatorProfileClusterEdge) String() string { return proto.CompactTextString(m) }
func (*ComplianceOperatorProfileClusterEdge) ProtoMessage()    {}
func (*ComplianceOperatorProfileClusterEdge) Descriptor() ([]byte, []int) {
	return fileDescriptor_26c2ec1f62102154, []int{8}
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
	proto.RegisterType((*ComplianceOperatorScanConfigurationV2)(nil), "storage.ComplianceOperatorScanConfigurationV2")
	proto.RegisterMapType((map[string]string)(nil), "storage.ComplianceOperatorScanConfigurationV2.AnnotationsEntry")
	proto.RegisterMapType((map[string]string)(nil), "storage.ComplianceOperatorScanConfigurationV2.LabelsEntry")
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
	// 2226 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xc4, 0x59, 0x4b, 0x73, 0xdb, 0xd6,
	0xf5, 0x37, 0xf4, 0xa0, 0xc8, 0x43, 0x4a, 0x82, 0xaf, 0x1f, 0x61, 0x14, 0xff, 0x0d, 0x86, 0x93,
	0xff, 0x0c, 0xe5, 0x58, 0xb4, 0xc3, 0xd8, 0xae, 0xad, 0xd6, 0xcd, 0x90, 0x12, 0x1d, 0x73, 0x4c,
	0x93, 0x0a, 0x48, 0x49, 0x75, 0x5f, 0x2c, 0x0c, 0x5c, 0x51, 0xa8, 0x40, 0x80, 0x01, 0x40, 0xcb,
	0xdc, 0x34, 0x9d, 0xb4, 0x33, 0xdd, 0x76, 0xd7, 0x76, 0xdd, 0x45, 0xa7, 0x1f, 0xa0, 0xdd, 0x75,
	0xdf, 0x5d, 0xbb, 0xeb, 0x0e, 0xed, 0xb8, 0xdf, 0x00, 0xbb, 0xee, 0x3a, 0xf7, 0x01, 0x0a, 0x20,
	0x41, 0x8a, 0x4e, 0x9c, 0x7a, 0xc7, 0x7b, 0xee, 0x3d, 0xbf, 0x73, 0x71, 0xef, 0xf9, 0x9d, 0xc7,
	0x25, 0x7c, 0xe0, 0xb8, 0x96, 0xad, 0x74, 0xf1, 0x2d, 0xd5, 0xea, 0xf5, 0x0d, 0x5d, 0x31, 0x55,
	0xdc, 0xb1, 0xfa, 0xd8, 0x56, 0x5c, 0xcb, 0xee, 0xbc, 0x28, 0x15, 0xfb, 0xb6, 0xe5, 0x5a, 0x68,
	0x85, 0xaf, 0xda, 0xb8, 0xdc, 0xb5, 0xba, 0x16, 0x95, 0xdd, 0x22, 0xbf, 0xd8, 0xf4, 0x86, 0xd4,
	0xb5, 0xac, 0xae, 0x81, 0x6f, 0xd1, 0xd1, 0xf3, 0xc1, 0xd1, 0x2d, 0x57, 0xef, 0x61, 0xc7, 0x55,
	0x7a, 0x7d, 0xbe, 0xe0, 0x6a, 0x60, 0xc5, 0x51, 0x8f, 0xb1, 0x36, 0x30, 0x30, 0x97, 0xa3, 0x40,
	0x3e, 0x70, 0xb0, 0xcd, 0x64, 0xf9, 0xdf, 0x09, 0x90, 0xde, 0xb3, 0xad, 0x23, 0xdd, 0xc0, 0xad,
	0x63, 0xbd, 0x87, 0xba, 0x00, 0x7d, 0x36, 0xec, 0xe8, 0x5a, 0x56, 0xc8, 0x09, 0x85, 0x54, 0xe5,
	0xb1, 0xef, 0x49, 0xbb, 0x0e, 0x56, 0x6c, 0xf5, 0x78, 0x3b, 0xbf, 0x95, 0xcf, 0x39, 0x9f, 0x1b,
	0xdb, 0xf9, 0xa3, 0x93, 0xc2, 0xce, 0xe8, 0x23, 0x9a, 0xfc, 0x1b, 0x38, 0xd2, 0x41, 0x69, 0x5b,
	0xd7, 0x36, 0x6f, 0x9a, 0xd6, 0xd6, 0xd1, 0xc9, 0x96, 0x6a, 0x99, 0x8e, 0x6b, 0x2b, 0xba, 0xe9,
	0xe6, 0xe5, 0x14, 0xc7, 0xae, 0x69, 0xe8, 0x7d, 0xc8, 0x04, 0x86, 0x4c, 0xa5, 0x87, 0xb3, 0x0b,
	0xc4, 0x94, 0x9c, 0xe6, 0xb2, 0x86, 0xd2, 0xc3, 0xf9, 0xbf, 0x24, 0xe1, 0xbd, 0x19, 0x36, 0xd0,
	0x35, 0x58, 0x18, 0xed, 0x31, 0xe3, 0x7b, 0x52, 0x92, 0x6e, 0xac, 0x7f, 0x92, 0x97, 0x17, 0x74,
	0x0d, 0xfd, 0x5f, 0xe4, 0x4b, 0x18, 0x7c, 0xc8, 0xfe, 0x17, 0xb0, 0x44, 0xed, 0x2e, 0x52, 0xf5,
	0x13, 0xdf, 0x93, 0xba, 0xc1, 0x27, 0x9e, 0xd9, 0xcc, 0x71, 0x63, 0x39, 0xb2, 0x9f, 0x9b, 0xc7,
	0xba, 0xa6, 0x61, 0xf3, 0x26, 0x39, 0x45, 0xcc, 0x4f, 0x41, 0x37, 0x35, 0xfc, 0xf2, 0xa1, 0xaa,
	0xb8, 0xb8, 0x6b, 0xd9, 0xc3, 0xed, 0x81, 0xa9, 0x7f, 0x3e, 0xc0, 0xdf, 0x26, 0xc0, 0xdb, 0x81,
	0x71, 0x26, 0xeb, 0xe8, 0xa6, 0xa6, 0xab, 0x64, 0xff, 0x79, 0x99, 0x1a, 0x46, 0xbf, 0x11, 0x60,
	0x3d, 0x58, 0xf3, 0x02, 0xdb, 0x8e, 0x6e, 0x99, 0xd9, 0x25, 0xba, 0x19, 0xd3, 0xf7, 0xa4, 0x9f,
	0xce, 0xd8, 0xcc, 0x01, 0x5b, 0xfd, 0xc6, 0xf6, 0xb3, 0xc6, 0xa7, 0x38, 0x2e, 0xfa, 0x21, 0xbd,
	0x1a, 0x6d, 0xa0, 0xba, 0x1d, 0x77, 0xd8, 0xc7, 0xd9, 0xe5, 0xdc, 0x62, 0x21, 0x55, 0x79, 0xe0,
	0x7b, 0xd2, 0xdd, 0x19, 0xbb, 0xda, 0x63, 0x2a, 0xb9, 0xf6, 0xb0, 0x3f, 0x76, 0x54, 0xf4, 0x56,
	0xc9, 0x1c, 0x99, 0x42, 0x35, 0x48, 0x3a, 0xae, 0x62, 0x6a, 0x8a, 0xad, 0x65, 0x13, 0xf4, 0x7b,
	0xb7, 0x7c, 0x4f, 0xda, 0x8c, 0x41, 0x6e, 0xf1, 0x65, 0x63, 0x68, 0x23, 0x75, 0xf4, 0x18, 0x12,
	0x86, 0xf2, 0x1c, 0x1b, 0x4e, 0x76, 0x25, 0xb7, 0x58, 0x48, 0x97, 0x6e, 0x17, 0xb9, 0x87, 0x17,
	0x67, 0xb8, 0x4d, 0xb1, 0x4e, 0x55, 0xaa, 0xa6, 0x6b, 0x0f, 0x65, 0xae, 0x8f, 0x0e, 0x21, 0xad,
	0x98, 0xa6, 0xe5, 0x2a, 0xae, 0x6e, 0x99, 0x4e, 0x36, 0x49, 0xe1, 0xee, 0xce, 0x05, 0x57, 0x3e,
	0xd3, 0x63, 0x98, 0x61, 0x24, 0x94, 0x83, 0xb4, 0x86, 0x1d, 0xd5, 0xd6, 0xfb, 0x64, 0x9c, 0x4d,
	0x31, 0x2f, 0x0f, 0x89, 0xd0, 0x27, 0xb0, 0x6c, 0x0f, 0x0c, 0xec, 0x64, 0x81, 0x1a, 0xdd, 0x9c,
	0xcb, 0xa8, 0x3c, 0x30, 0xb0, 0xcc, 0xf4, 0x50, 0x16, 0x56, 0xf8, 0xf9, 0x66, 0xd3, 0x14, 0x3e,
	0x18, 0xa2, 0xcb, 0xb0, 0xec, 0xea, 0xae, 0x81, 0xb3, 0x19, 0x2a, 0x67, 0x83, 0x8d, 0x07, 0x90,
	0x0e, 0x1d, 0x01, 0x12, 0x61, 0xf1, 0x04, 0x0f, 0x19, 0x8d, 0x64, 0xf2, 0x93, 0xa8, 0xbd, 0x50,
	0x8c, 0x41, 0xc0, 0x49, 0x36, 0xd8, 0x5e, 0xb8, 0x2f, 0x6c, 0x7c, 0x17, 0xc4, 0xf1, 0xcf, 0x7d,
	0x2d, 0x7d, 0x03, 0x96, 0xc8, 0xce, 0x91, 0x06, 0x29, 0xb2, 0x77, 0xc6, 0x7c, 0x46, 0xe0, 0x4f,
	0x7d, 0x4f, 0xda, 0x99, 0x2b, 0xc8, 0x10, 0x84, 0x83, 0xd2, 0x36, 0xd1, 0x8d, 0x8b, 0x31, 0x49,
	0x82, 0x4c, 0xe3, 0xc7, 0x3f, 0x12, 0x90, 0x9d, 0xa6, 0x8e, 0x9e, 0x70, 0xfe, 0x33, 0xeb, 0xdf,
	0xf2, 0x3d, 0xe9, 0xe3, 0x18, 0x17, 0x24, 0x8b, 0xa7, 0x92, 0x9f, 0x44, 0x1a, 0xc6, 0xe5, 0x9f,
	0x80, 0x78, 0x16, 0xc6, 0x39, 0x97, 0xe9, 0xc7, 0x57, 0xee, 0xfa, 0x9e, 0xf4, 0x51, 0x0c, 0x70,
	0xb0, 0x9b, 0x78, 0x32, 0xcb, 0xeb, 0x01, 0x5c, 0xc0, 0xc9, 0x36, 0x64, 0xe8, 0x89, 0x05, 0xe8,
	0x2c, 0x6c, 0x7d, 0xe4, 0x7b, 0xd2, 0xd6, 0xb4, 0x6d, 0xc7, 0x23, 0xa7, 0x09, 0x4c, 0x80, 0xfa,
	0x84, 0xdf, 0x03, 0xa5, 0x39, 0x0b, 0x3e, 0x45, 0xdf, 0x93, 0x6e, 0x4c, 0x83, 0x8c, 0xe1, 0x36,
	0x3d, 0x6e, 0x4a, 0x6c, 0x0c, 0x49, 0x07, 0xbf, 0xc0, 0xb6, 0xee, 0x0e, 0xb3, 0xcb, 0x39, 0xa1,
	0xb0, 0x56, 0xba, 0x32, 0xf2, 0x65, 0xa2, 0xdd, 0xe2, 0x93, 0x95, 0x92, 0xef, 0x49, 0xc5, 0x69,
	0x26, 0x82, 0x55, 0x13, 0xa4, 0xe7, 0x72, 0x54, 0x1d, 0x91, 0x3e, 0x41, 0x09, 0xb3, 0x35, 0x83,
	0x30, 0xec, 0xae, 0x63, 0x19, 0xdf, 0x8e, 0x32, 0x9e, 0x05, 0x90, 0xd2, 0xf9, 0x58, 0xb3, 0xe9,
	0x3e, 0x62, 0x5c, 0x32, 0xc4, 0xb8, 0x39, 0x82, 0xc0, 0x35, 0x48, 0xd9, 0x14, 0x42, 0x31, 0x70,
	0x16, 0x58, 0xae, 0x1a, 0x09, 0x08, 0xea, 0x91, 0xfe, 0x12, 0x3b, 0x9c, 0xdf, 0x6c, 0xf0, 0x16,
	0x79, 0x9c, 0xff, 0x53, 0x12, 0xfe, 0x7f, 0xf2, 0x84, 0x5a, 0xaa, 0x62, 0xee, 0x58, 0xe6, 0x91,
	0xde, 0x1d, 0xb0, 0xbd, 0x1f, 0x94, 0xd0, 0x61, 0x28, 0x47, 0x47, 0x28, 0x1e, 0x8e, 0xf3, 0xaa,
	0x62, 0xe6, 0x98, 0x62, 0xae, 0xb6, 0x1b, 0xcf, 0xb4, 0x9b, 0xc4, 0x45, 0x0b, 0x83, 0x81, 0xae,
	0x6d, 0xb2, 0xf4, 0xfe, 0x19, 0xa4, 0x1c, 0x55, 0x31, 0x43, 0xc5, 0x43, 0xe5, 0x8e, 0xef, 0x49,
	0xb7, 0xa7, 0xe1, 0x53, 0x12, 0x87, 0x31, 0x59, 0x1e, 0x24, 0x9e, 0xa5, 0x2a, 0x26, 0x99, 0x45,
	0xf7, 0xe0, 0x1d, 0x65, 0xe0, 0x5a, 0x1d, 0xa5, 0xdf, 0x37, 0x86, 0x1d, 0x1b, 0xf7, 0xb0, 0xa6,
	0x73, 0xf7, 0x20, 0x74, 0x4b, 0xca, 0x57, 0xc8, 0x74, 0x99, 0xcc, 0xca, 0xa1, 0x49, 0x74, 0x1f,
	0xb2, 0x54, 0x6f, 0xd0, 0xd7, 0x14, 0x17, 0x47, 0x15, 0x97, 0xa8, 0xe2, 0x55, 0x32, 0xbf, 0x4f,
	0xa7, 0x23, 0x9a, 0x79, 0x58, 0xb5, 0x4c, 0xdc, 0x21, 0x05, 0x5c, 0x87, 0x6c, 0x83, 0xf2, 0x26,
	0x29, 0xa7, 0x2d, 0x13, 0xb7, 0xf5, 0x1e, 0x26, 0x1b, 0x47, 0xf2, 0x98, 0xbf, 0x6f, 0xcf, 0xf0,
	0xd1, 0x98, 0x1b, 0x88, 0x75, 0x7e, 0x25, 0xce, 0xf9, 0x3f, 0x79, 0x4d, 0xe0, 0xd9, 0x4c, 0xb8,
	0x0d, 0x49, 0x5e, 0x56, 0x04, 0xe9, 0xf4, 0xf2, 0x08, 0x3f, 0x54, 0x70, 0xca, 0xa3, 0x55, 0xe8,
	0x36, 0x80, 0x69, 0x69, 0xb8, 0x63, 0x5b, 0x44, 0x27, 0x95, 0x5b, 0x2c, 0xac, 0x95, 0x2e, 0x8e,
	0x74, 0x1a, 0x96, 0x86, 0x65, 0xcb, 0xc0, 0x72, 0xca, 0xe4, 0xbf, 0x1c, 0x54, 0x00, 0xd1, 0x71,
	0x6d, 0x5d, 0x75, 0x3b, 0x54, 0x91, 0x9e, 0x20, 0xd0, 0x13, 0x5c, 0x63, 0x72, 0xa2, 0x44, 0x0f,
	0x71, 0x0b, 0x92, 0x41, 0x31, 0x4c, 0x49, 0x94, 0x0e, 0x21, 0xb7, 0xf8, 0x84, 0x3c, 0x5a, 0x82,
	0x1e, 0x42, 0x46, 0xb5, 0xb1, 0xe2, 0x62, 0x8d, 0xde, 0x0d, 0xcd, 0x9f, 0xe9, 0xd2, 0x46, 0x91,
	0x55, 0xde, 0xc5, 0xa0, 0xf2, 0x2e, 0xb6, 0x83, 0xca, 0x5b, 0x4e, 0xf3, 0xf5, 0x44, 0x82, 0x1e,
	0xc1, 0x45, 0x43, 0x71, 0x5c, 0xee, 0x10, 0x1c, 0x63, 0xf5, 0x5c, 0x8c, 0x75, 0xa2, 0xc4, 0xbc,
	0x84, 0xe1, 0xb4, 0x21, 0xdd, 0xb3, 0x34, 0xfd, 0x48, 0xc7, 0x5a, 0xe7, 0xf9, 0x30, 0xbb, 0x36,
	0xbe, 0x71, 0x43, 0xef, 0xed, 0x3b, 0xd8, 0xae, 0xe4, 0x7c, 0x4f, 0xba, 0xc6, 0xea, 0xbf, 0xae,
	0x69, 0xd9, 0xb8, 0xc3, 0xae, 0xb9, 0x40, 0x26, 0x73, 0xb5, 0xdd, 0xcd, 0xbc, 0x0c, 0x01, 0x4e,
	0x65, 0xf8, 0x36, 0xe3, 0xc6, 0x7f, 0x16, 0x61, 0x73, 0xd2, 0xb9, 0x76, 0x8c, 0x81, 0xe3, 0xe2,
	0x90, 0x8f, 0xb5, 0x5c, 0xc5, 0x1d, 0x38, 0xe8, 0x14, 0x40, 0x65, 0x53, 0x67, 0xbd, 0xc8, 0xf7,
	0x7c, 0x4f, 0x6a, 0x8f, 0x38, 0xce, 0x66, 0xa7, 0x07, 0x0d, 0x52, 0x3f, 0xb0, 0x35, 0xb1, 0x0d,
	0x49, 0x24, 0xa8, 0xa4, 0xb8, 0xad, 0x9a, 0x86, 0x7e, 0x2d, 0xc0, 0x0a, 0x0d, 0x2e, 0x41, 0xe3,
	0x50, 0x39, 0xf5, 0x3d, 0xc9, 0xf9, 0x4a, 0xa1, 0x2b, 0xb6, 0x84, 0x89, 0xa1, 0xd3, 0x94, 0x8e,
	0x29, 0x41, 0xf6, 0x51, 0xd3, 0xd0, 0x55, 0x48, 0x60, 0xdb, 0xb6, 0x6c, 0x12, 0x8a, 0x16, 0x0b,
	0x29, 0x99, 0x8f, 0xd0, 0x2f, 0x85, 0x38, 0x5f, 0x5b, 0x3a, 0xcf, 0xd7, 0x2a, 0xdf, 0xf1, 0x3d,
	0xe9, 0xfe, 0x39, 0x1f, 0x54, 0x57, 0x1c, 0x37, 0xc7, 0xfd, 0x31, 0x47, 0x54, 0xf9, 0xf7, 0xe5,
	0x27, 0x3d, 0xf5, 0x7d, 0xc8, 0x04, 0x57, 0x45, 0x03, 0xf2, 0x32, 0x4b, 0x71, 0x5c, 0x46, 0xab,
	0xb1, 0xbf, 0x65, 0x40, 0x8a, 0xb9, 0xfb, 0x63, 0xac, 0x9e, 0xc8, 0xd8, 0x19, 0x18, 0xee, 0xb9,
	0x1d, 0xdd, 0x63, 0x48, 0xaa, 0x64, 0xf9, 0xd9, 0xb5, 0x4c, 0xeb, 0x1c, 0x28, 0xea, 0xc4, 0x85,
	0xc8, 0x2b, 0x54, 0xbd, 0xa6, 0xa1, 0x06, 0x00, 0x43, 0x0a, 0xb5, 0x80, 0xb7, 0x7c, 0x4f, 0xfa,
	0x70, 0x2a, 0xd6, 0x64, 0x0d, 0x28, 0xa7, 0x28, 0x04, 0xcd, 0x1c, 0x6e, 0xc4, 0x53, 0x59, 0x21,
	0xb5, 0xef, 0x7b, 0xd2, 0x67, 0xf3, 0x79, 0xea, 0x57, 0x74, 0xd3, 0xf1, 0x43, 0x5f, 0x9f, 0x38,
	0x74, 0xf4, 0x05, 0x24, 0x1c, 0x4a, 0x26, 0x5e, 0x91, 0xdd, 0x99, 0x11, 0xe3, 0x23, 0x57, 0x51,
	0xa4, 0x23, 0x46, 0xc4, 0xa9, 0x65, 0x26, 0x3b, 0x1a, 0xb6, 0x68, 0xec, 0x70, 0xb8, 0xd9, 0x48,
	0x51, 0x98, 0xf8, 0xe6, 0x8a, 0xc2, 0xb1, 0x0a, 0x6b, 0x65, 0xb2, 0xc2, 0xca, 0x43, 0x46, 0x27,
	0x07, 0x3a, 0x50, 0x83, 0x16, 0x8f, 0x2c, 0x89, 0xc8, 0x50, 0x7d, 0x94, 0x6a, 0x53, 0x34, 0x63,
	0xcd, 0x7f, 0x5a, 0x71, 0x49, 0xf6, 0x07, 0xd1, 0x24, 0xcb, 0xda, 0xbb, 0x07, 0x73, 0x43, 0xce,
	0x4e, 0xaf, 0xa7, 0x63, 0x19, 0x2a, 0x7d, 0x2e, 0xe3, 0xef, 0xfb, 0x9e, 0x74, 0x67, 0xea, 0x25,
	0x32, 0xb3, 0xb9, 0x1d, 0x86, 0x19, 0x65, 0x7b, 0x24, 0xb7, 0x85, 0xdb, 0xf7, 0xcc, 0xd7, 0x6b,
	0xdf, 0xb3, 0xb0, 0xa2, 0x5a, 0xa6, 0x6b, 0x5b, 0x06, 0x4d, 0x8e, 0x29, 0x39, 0x18, 0xa2, 0x1f,
	0x9f, 0xc5, 0xdf, 0x35, 0x6a, 0xa3, 0xea, 0x7b, 0x52, 0x79, 0xae, 0xee, 0x90, 0x04, 0xb0, 0x73,
	0xa3, 0xe9, 0x1f, 0x05, 0x58, 0xa3, 0x06, 0x54, 0x1a, 0xe9, 0x88, 0x1d, 0x91, 0xda, 0xf9, 0x52,
	0xf0, 0x3d, 0xe9, 0x67, 0xff, 0xfb, 0x40, 0x1f, 0xa1, 0x78, 0xc6, 0x19, 0x69, 0xd5, 0x34, 0x5a,
	0xe4, 0x84, 0xb6, 0x4a, 0x99, 0x7e, 0x91, 0x1e, 0xd7, 0xda, 0xd9, 0x3a, 0x42, 0xf6, 0xb7, 0x99,
	0xd8, 0x1d, 0x48, 0x87, 0x02, 0x06, 0x4a, 0xc1, 0xf2, 0x7e, 0xa3, 0x55, 0x6d, 0x8b, 0x17, 0x50,
	0x12, 0x96, 0xf6, 0xca, 0xad, 0x96, 0x28, 0x90, 0x5f, 0x8f, 0xca, 0xb5, 0xba, 0xb8, 0x40, 0xa6,
	0xab, 0xb2, 0xdc, 0x94, 0xc5, 0x45, 0x22, 0xac, 0x35, 0x1e, 0x35, 0xc5, 0x25, 0x04, 0x90, 0x78,
	0x5a, 0x6e, 0xec, 0x97, 0xeb, 0xe2, 0x32, 0x42, 0xb0, 0xd6, 0x68, 0xb6, 0x3b, 0xe5, 0xbd, 0xbd,
	0x7a, 0x6d, 0xa7, 0x5c, 0xa9, 0x57, 0xc5, 0x04, 0x12, 0x21, 0x53, 0x6b, 0xec, 0x34, 0x1b, 0xad,
	0x5a, 0xab, 0x5d, 0x6d, 0xb4, 0xc5, 0x95, 0xfc, 0x01, 0x00, 0x39, 0x5f, 0x6e, 0xf3, 0x32, 0x2c,
	0xf7, 0x8f, 0x15, 0x87, 0x77, 0xf4, 0x32, 0x1b, 0x90, 0xbc, 0x69, 0x53, 0xd7, 0xe6, 0x7b, 0xe6,
	0x23, 0xb4, 0x01, 0xc9, 0x53, 0xc5, 0x36, 0x75, 0xb3, 0x1b, 0x64, 0xd4, 0xd1, 0x38, 0xff, 0xcf,
	0x64, 0xdc, 0xbb, 0x01, 0x73, 0xac, 0x73, 0x52, 0xd4, 0x2f, 0x26, 0x1d, 0x8b, 0x65, 0xaa, 0x1f,
	0xf9, 0x9e, 0xf4, 0x6c, 0x6e, 0x07, 0xfe, 0x9a, 0x2e, 0xf3, 0x5e, 0xb8, 0x37, 0xa2, 0x04, 0x0a,
	0x75, 0x39, 0x7f, 0x10, 0x22, 0xc9, 0x8a, 0x25, 0xbf, 0x5f, 0x11, 0xbf, 0xff, 0x52, 0x78, 0xd3,
	0xe9, 0xea, 0xe6, 0x8c, 0xf7, 0x48, 0xba, 0xc3, 0xc9, 0xc7, 0xc8, 0x19, 0xf9, 0x6d, 0x75, 0x32,
	0xbf, 0x9d, 0x95, 0x45, 0x4b, 0x91, 0xb2, 0xa8, 0x48, 0xdf, 0xc4, 0x48, 0x5f, 0x41, 0x5f, 0x2f,
	0xa7, 0x35, 0x1f, 0xc1, 0xa2, 0xd7, 0x7a, 0x54, 0x60, 0x8e, 0xf0, 0x66, 0x1e, 0x15, 0x38, 0xd6,
	0xec, 0x58, 0x5f, 0xe4, 0xd7, 0x49, 0x5f, 0x69, 0x92, 0x34, 0x89, 0x86, 0xbb, 0x17, 0xc5, 0x6c,
	0x0f, 0xfb, 0x98, 0xdd, 0x30, 0x7d, 0x88, 0xb9, 0x07, 0xab, 0xac, 0x1f, 0xc2, 0x06, 0x56, 0x5d,
	0xcb, 0xa6, 0x0f, 0x0e, 0xb1, 0xbd, 0x54, 0x86, 0xac, 0x6b, 0xf1, 0x65, 0xe8, 0xc3, 0x51, 0xb1,
	0x00, 0x34, 0x9b, 0x5c, 0x8a, 0x18, 0x61, 0x34, 0x1b, 0x25, 0xf6, 0x87, 0xaf, 0x9b, 0x80, 0xa2,
	0x69, 0xe4, 0xe7, 0x02, 0x20, 0x5a, 0xb7, 0xe2, 0x97, 0x58, 0x1d, 0xcc, 0xdf, 0x68, 0x55, 0xee,
	0xf9, 0x9e, 0x54, 0x9a, 0x16, 0xa0, 0x69, 0xc5, 0x5a, 0xe5, 0x88, 0xd1, 0x24, 0x26, 0x12, 0x6b,
	0xc1, 0x14, 0x99, 0x79, 0x9b, 0xe1, 0xf2, 0xcf, 0x0b, 0xf0, 0xc1, 0xd4, 0xe7, 0x5d, 0xce, 0xad,
	0xaa, 0xd6, 0xc5, 0x48, 0x0a, 0x45, 0x9b, 0x75, 0xdf, 0x93, 0xd2, 0x41, 0x67, 0xa3, 0x6b, 0x2c,
	0xe0, 0x9c, 0x4e, 0xfe, 0xcb, 0x31, 0xd6, 0x23, 0x4d, 0xbe, 0xd4, 0xd7, 0x76, 0x19, 0x9d, 0x83,
	0xb3, 0x99, 0xf7, 0x0f, 0x9d, 0xc8, 0xff, 0x37, 0x6e, 0x4c, 0x14, 0xf9, 0xc6, 0x4b, 0xde, 0x1b,
	0x5b, 0x90, 0x0c, 0x9c, 0x97, 0x64, 0x91, 0x5a, 0xe3, 0x91, 0x5c, 0x16, 0x2f, 0x90, 0xdc, 0x71,
	0xd8, 0x94, 0x9f, 0x54, 0x65, 0x51, 0x60, 0x79, 0xa4, 0xd5, 0xae, 0xca, 0xe2, 0xc2, 0x8d, 0x32,
	0x24, 0x03, 0x7e, 0xa0, 0x4b, 0xb0, 0x4e, 0x73, 0x52, 0xa7, 0xb5, 0x53, 0x6e, 0x74, 0xda, 0xcf,
	0xf6, 0xaa, 0xe2, 0x05, 0xb4, 0x0a, 0xa9, 0x46, 0x73, 0xb7, 0x4a, 0x65, 0xa2, 0x80, 0x2e, 0xc2,
	0xea, 0x5e, 0xbd, 0xdc, 0x7e, 0xd4, 0x94, 0x9f, 0x32, 0xd1, 0xc2, 0x8d, 0xdf, 0x0b, 0x90, 0x09,
	0x17, 0xaa, 0xe8, 0x1d, 0xb8, 0xc4, 0x70, 0xe4, 0xfd, 0x7a, 0xb5, 0xd3, 0xaa, 0x1e, 0x54, 0xe5,
	0x5a, 0xfb, 0x99, 0x78, 0x01, 0xbd, 0x0b, 0x57, 0xf6, 0x1b, 0x4f, 0x1a, 0xcd, 0xc3, 0xc6, 0xd8,
	0x94, 0x80, 0xae, 0x02, 0x22, 0x59, 0x6e, 0x4c, 0xbe, 0x80, 0xae, 0xc0, 0xc5, 0x7a, 0xf3, 0x70,
	0x4c, 0xbc, 0x88, 0xb2, 0x70, 0xf9, 0x69, 0x75, 0xb7, 0xb6, 0xff, 0x74, 0x6c, 0x66, 0x89, 0x00,
	0x3d, 0xae, 0x7d, 0xfa, 0x78, 0x4c, 0xbe, 0x5c, 0xb9, 0xf3, 0xd7, 0x57, 0xd7, 0x85, 0xbf, 0xbf,
	0xba, 0x2e, 0xfc, 0xeb, 0xd5, 0x75, 0xe1, 0xb7, 0xff, 0xbe, 0x7e, 0x01, 0xde, 0xd5, 0xad, 0xa2,
	0xe3, 0x2a, 0xea, 0x89, 0x6d, 0xbd, 0x64, 0x3c, 0x0a, 0xd8, 0xfc, 0xfd, 0xe0, 0xff, 0xc5, 0xe7,
	0x09, 0x2a, 0xff, 0xf8, 0xbf, 0x01, 0x00, 0x00, 0xff, 0xff, 0x1a, 0xb5, 0x30, 0x16, 0x97, 0x1c,
	0x00, 0x00,
=======
	// 2227 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xc4, 0x59, 0x4b, 0x73, 0xdb, 0xd6,
	0xf5, 0x37, 0x24, 0x91, 0x22, 0x0f, 0x29, 0x09, 0xbe, 0x96, 0x1d, 0x46, 0xf1, 0xdf, 0x60, 0x38,
	0xf9, 0xcf, 0x48, 0x8e, 0x45, 0x3b, 0x8c, 0xed, 0xda, 0x6a, 0xdd, 0x0c, 0x29, 0xd1, 0x31, 0xc7,
	0x34, 0xa9, 0x80, 0x94, 0x5c, 0xf7, 0xc5, 0xc2, 0xc0, 0x15, 0x85, 0x0a, 0x04, 0x18, 0x00, 0xb4,
	0xcc, 0x4d, 0xd3, 0x49, 0x3b, 0xd3, 0x6d, 0x77, 0x6d, 0xd7, 0x5d, 0x74, 0xfa, 0x01, 0xda, 0xcf,
	0xd0, 0x5d, 0x3b, 0xdd, 0x74, 0x87, 0x76, 0xdc, 0x6f, 0x80, 0x5d, 0x77, 0x9d, 0xfb, 0x00, 0x05,
	0x90, 0x20, 0x45, 0x27, 0x4e, 0xbd, 0xe3, 0x7d, 0x9c, 0xdf, 0x39, 0xb8, 0xf7, 0xfc, 0xce, 0xe3,
	0x12, 0x3e, 0x70, 0x5c, 0xcb, 0x56, 0xba, 0xf8, 0xa6, 0x6a, 0xf5, 0xfa, 0x86, 0xae, 0x98, 0x2a,
	0xee, 0x58, 0x7d, 0x6c, 0x2b, 0xae, 0x65, 0x77, 0x5e, 0x94, 0x8a, 0x7d, 0xdb, 0x72, 0x2d, 0xb4,
	0xcc, 0x77, 0x6d, 0xac, 0x77, 0xad, 0xae, 0x45, 0xe7, 0x6e, 0x92, 0x5f, 0x6c, 0x79, 0x43, 0xea,
	0x5a, 0x56, 0xd7, 0xc0, 0x37, 0xe9, 0xe8, 0xf9, 0xe0, 0xe8, 0xa6, 0xab, 0xf7, 0xb0, 0xe3, 0x2a,
	0xbd, 0x3e, 0xdf, 0x70, 0x25, 0xd0, 0xe2, 0xa8, 0xc7, 0x58, 0x1b, 0x18, 0x98, 0xcf, 0xa3, 0x60,
	0x7e, 0xe0, 0x60, 0x9b, 0xcd, 0x15, 0x7e, 0x27, 0x40, 0x66, 0xdf, 0xb6, 0x8e, 0x74, 0x03, 0xb7,
	0x8e, 0xf5, 0x1e, 0xea, 0x02, 0xf4, 0xd9, 0xb0, 0xa3, 0x6b, 0x39, 0x21, 0x2f, 0x6c, 0xa6, 0x2b,
	0x8f, 0x7c, 0x4f, 0xda, 0x73, 0xb0, 0x62, 0xab, 0xc7, 0x3b, 0x85, 0xed, 0x42, 0xde, 0xf9, 0xdc,
	0xd8, 0x29, 0x1c, 0x9d, 0x6c, 0xee, 0x8e, 0x3e, 0xa2, 0xc9, 0xbf, 0x81, 0x23, 0x1d, 0x96, 0x76,
	0x74, 0x6d, 0xeb, 0x86, 0x69, 0x6d, 0x1f, 0x9d, 0x6c, 0xab, 0x96, 0xe9, 0xb8, 0xb6, 0xa2, 0x9b,
	0x6e, 0x41, 0x4e, 0x73, 0xec, 0x9a, 0x86, 0xde, 0x87, 0x6c, 0xa0, 0xc8, 0x54, 0x7a, 0x38, 0xb7,
	0x40, 0x54, 0xc9, 0x19, 0x3e, 0xd7, 0x50, 0x7a, 0xb8, 0xf0, 0xf7, 0x14, 0xbc, 0x37, 0x43, 0x07,
	0xba, 0x0a, 0x0b, 0x23, 0x1b, 0xb3, 0xbe, 0x27, 0xa5, 0xa8, 0x61, 0xfd, 0x93, 0x82, 0xbc, 0xa0,
	0x6b, 0xe8, 0xff, 0x22, 0x5f, 0xc2, 0xe0, 0x43, 0xfa, 0xbf, 0x80, 0x25, 0xaa, 0x77, 0x91, 0x8a,
	0x9f, 0xf8, 0x9e, 0xd4, 0x0d, 0x3e, 0xf1, 0x4c, 0x67, 0x9e, 0x2b, 0xcb, 0x13, 0x7b, 0x6e, 0x1c,
	0xeb, 0x9a, 0x86, 0xcd, 0x1b, 0xe4, 0x14, 0x31, 0x3f, 0x05, 0xdd, 0xd4, 0xf0, 0xcb, 0x07, 0xaa,
	0xe2, 0xe2, 0xae, 0x65, 0x0f, 0x77, 0x06, 0xa6, 0xfe, 0xf9, 0x00, 0x7f, 0x9b, 0x00, 0xef, 0x04,
	0xca, 0xd9, 0x5c, 0x47, 0x37, 0x35, 0x5d, 0x25, 0xf6, 0x17, 0x64, 0xaa, 0x18, 0x6d, 0x81, 0x78,
	0x76, 0xf5, 0xd8, 0x76, 0x74, 0xcb, 0xcc, 0x2d, 0x51, 0x2b, 0xd7, 0x82, 0xf9, 0x43, 0x36, 0x8d,
	0x7e, 0x23, 0xc0, 0x5a, 0x00, 0x17, 0x6c, 0x4d, 0x50, 0xbb, 0x4d, 0xdf, 0x93, 0x7e, 0x3a, 0xc3,
	0x6e, 0x8e, 0xf0, 0xc6, 0x4c, 0x5f, 0xe5, 0x4b, 0x81, 0x65, 0x3f, 0xa4, 0xb7, 0xa8, 0x0d, 0x54,
	0xb7, 0xe3, 0x0e, 0xfb, 0x38, 0x97, 0xcc, 0x2f, 0x6e, 0xa6, 0x2b, 0xf7, 0x7d, 0x4f, 0xba, 0x33,
	0xc3, 0xaa, 0x7d, 0x26, 0x92, 0x6f, 0x0f, 0xfb, 0x63, 0xa7, 0x4a, 0x1d, 0x80, 0xac, 0x91, 0x25,
	0x54, 0x83, 0x94, 0xe3, 0x2a, 0xa6, 0xa6, 0xd8, 0x5a, 0x6e, 0x99, 0x7e, 0xef, 0xb6, 0xef, 0x49,
	0x5b, 0x31, 0xc8, 0x2d, 0xbe, 0x6d, 0x0c, 0x6d, 0x24, 0x8e, 0x1e, 0x41, 0xd2, 0x50, 0x9e, 0x63,
	0xc3, 0xc9, 0xa5, 0xf2, 0x8b, 0x9b, 0x99, 0xd2, 0xad, 0x22, 0x27, 0x43, 0x71, 0x86, 0x87, 0x15,
	0xeb, 0x54, 0xa4, 0x6a, 0xba, 0xf6, 0x50, 0xe6, 0xf2, 0xe8, 0x29, 0x64, 0x14, 0xd3, 0xb4, 0x5c,
	0xc5, 0xd5, 0x2d, 0xd3, 0xc9, 0xa5, 0x29, 0xdc, 0x9d, 0xb9, 0xe0, 0xca, 0x67, 0x72, 0x0c, 0x33,
	0x8c, 0x84, 0xf2, 0x90, 0xd1, 0xb0, 0xa3, 0xda, 0x7a, 0x9f, 0x8c, 0x73, 0xc0, 0x08, 0x11, 0x9a,
	0x42, 0x9f, 0x40, 0xc2, 0x1e, 0x18, 0xd8, 0xc9, 0x65, 0xa8, 0xd2, 0xad, 0xb9, 0x94, 0xca, 0x03,
	0x03, 0xcb, 0x4c, 0x0e, 0xe5, 0x60, 0x99, 0x9f, 0x6f, 0x2e, 0x4b, 0xe1, 0x83, 0xe1, 0xc6, 0x7d,
	0xc8, 0x84, 0x3e, 0x16, 0x89, 0xb0, 0x78, 0x82, 0x87, 0x8c, 0x5b, 0x32, 0xf9, 0x89, 0xd6, 0x21,
	0xf1, 0x42, 0x31, 0x06, 0x01, 0x51, 0xd9, 0x60, 0x67, 0xe1, 0x9e, 0xb0, 0xf1, 0x5d, 0x10, 0xc7,
	0x3f, 0xec, 0xb5, 0xe4, 0x0d, 0x58, 0x22, 0x36, 0x22, 0x0d, 0xd2, 0xc4, 0x4a, 0x16, 0x0e, 0x18,
	0xab, 0x3f, 0xf5, 0x3d, 0x69, 0x77, 0xae, 0xc8, 0x43, 0x10, 0x0e, 0x4b, 0x3b, 0x44, 0x36, 0x2e,
	0xf0, 0xa4, 0x08, 0x32, 0x0d, 0x2a, 0xff, 0x48, 0x42, 0x6e, 0x9a, 0x38, 0x7a, 0xcc, 0x83, 0x02,
	0xd3, 0xfe, 0x2d, 0xdf, 0x93, 0x3e, 0x8e, 0x71, 0x36, 0xb2, 0x79, 0x6a, 0x44, 0x20, 0xe1, 0x87,
	0x11, 0xfc, 0x27, 0x31, 0x04, 0xa7, 0x1f, 0x5f, 0xb9, 0xe3, 0x7b, 0xd2, 0x47, 0x31, 0xc0, 0x81,
	0x35, 0xf1, 0xb4, 0x9d, 0x8c, 0x0b, 0x6d, 0xc8, 0xd2, 0x13, 0x0b, 0xd0, 0x59, 0x2c, 0xfb, 0xc8,
	0xf7, 0xa4, 0xed, 0x69, 0x66, 0xc7, 0x23, 0x67, 0x08, 0x4c, 0x80, 0xfa, 0x98, 0xdf, 0x03, 0x25,
	0x34, 0x8d, 0x48, 0x95, 0xa2, 0xef, 0x49, 0xd7, 0xa7, 0x41, 0xc6, 0xb0, 0x98, 0x1e, 0x37, 0xa5,
	0x30, 0x86, 0x94, 0x83, 0x5f, 0x60, 0x5b, 0x77, 0x87, 0x34, 0x64, 0xad, 0x96, 0x2e, 0x8f, 0xbc,
	0x96, 0x48, 0xb7, 0xf8, 0x62, 0xa5, 0xe4, 0x7b, 0x52, 0x71, 0x9a, 0x8a, 0x60, 0xd7, 0x04, 0xbd,
	0xf9, 0x3c, 0xaa, 0x8e, 0xe8, 0x9d, 0xa4, 0xd4, 0xd8, 0x9e, 0x41, 0x0d, 0x76, 0xd7, 0xb1, 0xdc,
	0x6e, 0x47, 0xb9, 0xbd, 0x4c, 0xb1, 0x4a, 0xe7, 0x63, 0xcd, 0x26, 0xf6, 0x3a, 0x24, 0x5c, 0xdd,
	0x35, 0x70, 0x2e, 0xc5, 0x5c, 0x9f, 0x0e, 0xc6, 0xe9, 0x9e, 0x9e, 0xa4, 0xfb, 0x55, 0x48, 0xdb,
	0x14, 0x42, 0x31, 0x30, 0x0f, 0x07, 0x67, 0x13, 0x04, 0xf5, 0x48, 0x7f, 0x49, 0x83, 0x01, 0x45,
	0xa5, 0x83, 0xb7, 0xc8, 0xe3, 0xc2, 0x9f, 0x52, 0xf0, 0xff, 0x93, 0x27, 0xd4, 0x52, 0x15, 0x73,
	0xd7, 0x32, 0x8f, 0xf4, 0xee, 0x80, 0xd9, 0x7e, 0x58, 0x42, 0x4f, 0x43, 0x89, 0x3b, 0x42, 0xf1,
	0x70, 0x44, 0x57, 0x15, 0x33, 0xcf, 0x04, 0xf3, 0xb5, 0xbd, 0x78, 0xa6, 0xdd, 0x20, 0x2e, 0xba,
	0x39, 0x18, 0xe8, 0xda, 0x16, 0xcb, 0xf9, 0x9f, 0x41, 0xda, 0x51, 0x15, 0x33, 0x54, 0x51, 0x54,
	0x6e, 0xfb, 0x9e, 0x74, 0x6b, 0x1a, 0x3e, 0x25, 0x71, 0x18, 0x93, 0x65, 0x3c, 0xe2, 0x59, 0xaa,
	0x62, 0x92, 0x55, 0x74, 0x17, 0xde, 0x51, 0x06, 0xae, 0xd5, 0x51, 0xfa, 0x7d, 0x63, 0xd8, 0xb1,
	0x71, 0x0f, 0x6b, 0x3a, 0x77, 0x0f, 0x42, 0xb7, 0x94, 0x7c, 0x99, 0x2c, 0x97, 0xc9, 0xaa, 0x1c,
	0x5a, 0x44, 0xf7, 0x20, 0x47, 0xe5, 0x06, 0x7d, 0x4d, 0x71, 0x71, 0x54, 0x70, 0x89, 0x0a, 0x5e,
	0x21, 0xeb, 0x07, 0x74, 0x39, 0x22, 0x59, 0x80, 0x15, 0xcb, 0xc4, 0x1d, 0x52, 0xd5, 0x75, 0x88,
	0x19, 0x94, 0x37, 0x29, 0x39, 0x63, 0x99, 0xb8, 0xad, 0xf7, 0x30, 0x31, 0x1c, 0xc9, 0x63, 0xfe,
	0xbe, 0x33, 0xc3, 0x47, 0x63, 0x6e, 0x20, 0xd6, 0xf9, 0x95, 0x38, 0xe7, 0xff, 0xe4, 0x35, 0x81,
	0x67, 0x33, 0xe1, 0x16, 0xa4, 0x78, 0x01, 0x11, 0xe4, 0xe1, 0xf5, 0x11, 0x7e, 0xa8, 0x0a, 0x95,
	0x47, 0xbb, 0xd0, 0x2d, 0x00, 0xd3, 0xd2, 0x70, 0xc7, 0xb6, 0x88, 0x0c, 0x49, 0xb6, 0xab, 0xa5,
	0x8b, 0x23, 0x99, 0x86, 0xa5, 0x61, 0xd9, 0x32, 0xb0, 0x9c, 0x36, 0xf9, 0x2f, 0x07, 0x6d, 0x82,
	0xe8, 0xb8, 0xb6, 0xae, 0xba, 0x1d, 0x2a, 0x48, 0x4f, 0x10, 0xe8, 0x09, 0xae, 0xb2, 0x79, 0x22,
	0x44, 0x0f, 0x71, 0x1b, 0x52, 0x41, 0x85, 0x4c, 0x49, 0x94, 0x09, 0x21, 0xb7, 0xf8, 0x82, 0x3c,
	0xda, 0x82, 0x1e, 0x40, 0x56, 0xb5, 0xb1, 0xe2, 0x62, 0x8d, 0xde, 0x0d, 0xcd, 0xa0, 0x99, 0xd2,
	0x46, 0x91, 0x95, 0xe3, 0xc5, 0xa0, 0x1c, 0x2f, 0xb6, 0x83, 0x72, 0x5c, 0xce, 0xf0, 0xfd, 0x64,
	0x06, 0x3d, 0x84, 0x8b, 0x86, 0xe2, 0xb8, 0xdc, 0x21, 0x38, 0xc6, 0xca, 0xb9, 0x18, 0x6b, 0x44,
	0x88, 0x79, 0x09, 0xc3, 0x69, 0x43, 0xa6, 0x67, 0x69, 0xfa, 0x91, 0x8e, 0xb5, 0xce, 0xf3, 0x61,
	0x6e, 0x75, 0xdc, 0x70, 0x43, 0xef, 0x1d, 0x38, 0xd8, 0xae, 0xe4, 0x7d, 0x4f, 0xba, 0xca, 0x2a,
	0xbd, 0xae, 0x69, 0xd9, 0xb8, 0xc3, 0xae, 0x79, 0x93, 0x2c, 0xe6, 0x6b, 0x7b, 0x5b, 0x05, 0x19,
	0x02, 0x9c, 0xca, 0xf0, 0x6d, 0xc6, 0x8d, 0xff, 0x2c, 0xc2, 0xd6, 0xa4, 0x73, 0xed, 0x1a, 0x03,
	0xc7, 0xc5, 0x21, 0x1f, 0x6b, 0xb9, 0x8a, 0x3b, 0x70, 0xd0, 0x29, 0x80, 0xca, 0x96, 0xce, 0x1a,
	0x94, 0xef, 0xf9, 0x9e, 0xd4, 0x1e, 0x71, 0x9c, 0xad, 0x4e, 0x0f, 0x1a, 0xa4, 0x7e, 0x60, 0x7b,
	0x62, 0xbb, 0x94, 0x48, 0x50, 0x49, 0x73, 0x5d, 0x35, 0x0d, 0xfd, 0x5a, 0x80, 0x65, 0x1a, 0x5c,
	0x82, 0x6e, 0xa2, 0x72, 0xea, 0x7b, 0x92, 0xf3, 0x95, 0x42, 0x57, 0x6c, 0x09, 0x13, 0x43, 0xa7,
	0x29, 0x6d, 0x54, 0x92, 0xd8, 0x51, 0xd3, 0xd0, 0x15, 0x48, 0x62, 0xdb, 0xb6, 0x6c, 0x12, 0x8a,
	0x16, 0x37, 0xd3, 0x32, 0x1f, 0xa1, 0x5f, 0x0a, 0x71, 0xbe, 0xb6, 0x74, 0x9e, 0xaf, 0x55, 0xbe,
	0xe3, 0x7b, 0xd2, 0xbd, 0x73, 0x3e, 0xa8, 0xae, 0x38, 0x6e, 0x9e, 0xfb, 0x63, 0x9e, 0x88, 0xf2,
	0xef, 0x2b, 0x4c, 0x7a, 0xea, 0xfb, 0x90, 0x0d, 0xae, 0x8a, 0x06, 0xe4, 0x04, 0x4b, 0x71, 0x7c,
	0x8e, 0x56, 0x63, 0x7f, 0xcd, 0x82, 0x14, 0x73, 0xf7, 0xc7, 0x58, 0x3d, 0x91, 0xb1, 0x33, 0x30,
	0xdc, 0x73, 0xdb, 0xbc, 0x47, 0x90, 0x52, 0xc9, 0xf6, 0xb3, 0x6b, 0x99, 0xd6, 0x23, 0x50, 0xd4,
	0x89, 0x0b, 0x91, 0x97, 0xa9, 0x78, 0x4d, 0x43, 0x0d, 0x00, 0x86, 0x14, 0xea, 0x0b, 0x6f, 0xfa,
	0x9e, 0xf4, 0xe1, 0x54, 0xac, 0xc9, 0x1a, 0x50, 0x4e, 0x53, 0x08, 0x9a, 0x39, 0xdc, 0x88, 0xa7,
	0xb2, 0x42, 0xea, 0xc0, 0xf7, 0xa4, 0xcf, 0xe6, 0xf3, 0xd4, 0xaf, 0xe8, 0xa6, 0xe3, 0x87, 0xbe,
	0x36, 0x71, 0xe8, 0xe8, 0x0b, 0x48, 0x3a, 0x94, 0x4c, 0xbc, 0x22, 0xbb, 0x3d, 0x23, 0xc6, 0x47,
	0xae, 0xa2, 0x48, 0x47, 0x8c, 0x88, 0x53, 0xcb, 0x4c, 0x76, 0x34, 0x6c, 0xd3, 0xd8, 0xe1, 0x70,
	0xb5, 0x91, 0xa2, 0x30, 0xf9, 0xcd, 0x15, 0x85, 0x63, 0x15, 0xd6, 0xf2, 0x64, 0x85, 0x55, 0x80,
	0xac, 0x4e, 0x0e, 0x74, 0xa0, 0xb2, 0x9c, 0xc7, 0x0a, 0xb4, 0xc8, 0x1c, 0xaa, 0x8f, 0x52, 0x2d,
	0x6b, 0xf5, 0xe6, 0x3f, 0xad, 0xb8, 0x24, 0xfb, 0x83, 0x68, 0x92, 0x05, 0x0a, 0x79, 0x7f, 0x6e,
	0xc8, 0xd9, 0xe9, 0xf5, 0x74, 0x2c, 0x43, 0x65, 0xce, 0x65, 0xfc, 0x3d, 0xdf, 0x93, 0x6e, 0x4f,
	0xbd, 0x44, 0xa6, 0x36, 0xbf, 0xcb, 0x30, 0xa3, 0x6c, 0x8f, 0xe4, 0xb6, 0x70, 0xa3, 0x9e, 0xfd,
	0x7a, 0x8d, 0x7a, 0x0e, 0x96, 0x55, 0xcb, 0x74, 0x6d, 0xcb, 0xa0, 0xc9, 0x31, 0x2d, 0x07, 0x43,
	0xf4, 0xe3, 0xb3, 0xf8, 0xbb, 0x4a, 0x75, 0x54, 0x7d, 0x4f, 0x2a, 0xcf, 0xd5, 0x1d, 0x92, 0x00,
	0x76, 0x6e, 0x34, 0xfd, 0xa3, 0x00, 0xab, 0x54, 0x81, 0x4a, 0x23, 0x1d, 0xd1, 0x23, 0x52, 0x3d,
	0x5f, 0x0a, 0xbe, 0x27, 0xfd, 0xec, 0x7f, 0x1f, 0xe8, 0x23, 0x14, 0xcf, 0x3a, 0x23, 0xa9, 0x9a,
	0x46, 0x8b, 0x9c, 0x90, 0xa9, 0x94, 0xe9, 0x17, 0xe9, 0x71, 0xad, 0x9e, 0xed, 0x23, 0x64, 0x7f,
	0x9b, 0x89, 0xdd, 0x81, 0x4c, 0x28, 0x60, 0xa0, 0x34, 0x24, 0x0e, 0x1a, 0xad, 0x6a, 0x5b, 0xbc,
	0x80, 0x52, 0xb0, 0xb4, 0x5f, 0x6e, 0xb5, 0x44, 0x81, 0xfc, 0x7a, 0x58, 0xae, 0xd5, 0xc5, 0x05,
	0xb2, 0x5c, 0x95, 0xe5, 0xa6, 0x2c, 0x2e, 0x92, 0xc9, 0x5a, 0xe3, 0x61, 0x53, 0x5c, 0x42, 0x00,
	0xc9, 0x27, 0xe5, 0xc6, 0x41, 0xb9, 0x2e, 0x26, 0x10, 0x82, 0xd5, 0x46, 0xb3, 0xdd, 0x29, 0xef,
	0xef, 0xd7, 0x6b, 0xbb, 0xe5, 0x4a, 0xbd, 0x2a, 0x26, 0x91, 0x08, 0xd9, 0x5a, 0x63, 0xb7, 0xd9,
	0x68, 0xd5, 0x5a, 0xed, 0x6a, 0xa3, 0x2d, 0x2e, 0x17, 0x0e, 0x01, 0xc8, 0xf9, 0x72, 0x9d, 0xeb,
	0x90, 0xe8, 0x1f, 0x2b, 0x0e, 0xef, 0xe8, 0x65, 0x36, 0x20, 0x79, 0xd3, 0xa6, 0xae, 0xcd, 0x6d,
	0xe6, 0x23, 0xb4, 0x01, 0xa9, 0x53, 0xc5, 0x36, 0x75, 0xb3, 0x1b, 0x64, 0xd4, 0xd1, 0xb8, 0xf0,
	0xcf, 0x54, 0xdc, 0xbb, 0x01, 0x73, 0xac, 0x73, 0x52, 0xd4, 0x2f, 0x26, 0x1d, 0x8b, 0x65, 0xaa,
	0x1f, 0xf9, 0x9e, 0xf4, 0x6c, 0x6e, 0x07, 0xfe, 0x9a, 0x2e, 0xf3, 0x5e, 0xb8, 0x37, 0xa2, 0x04,
	0x0a, 0x75, 0x39, 0x7f, 0x10, 0x22, 0xc9, 0x8a, 0x25, 0xbf, 0x5f, 0x11, 0xbf, 0xff, 0x52, 0x78,
	0xd3, 0xe9, 0xea, 0xc6, 0x8c, 0x97, 0x47, 0x6a, 0xe1, 0xe4, 0xb3, 0xe3, 0x8c, 0xfc, 0xb6, 0x32,
	0x99, 0xdf, 0xce, 0xca, 0xa2, 0xa5, 0x48, 0x59, 0x54, 0xa4, 0xaf, 0x5f, 0xa4, 0xaf, 0xc8, 0x25,
	0x66, 0x34, 0x1f, 0xc1, 0xa6, 0xd7, 0x7a, 0x54, 0x60, 0x8e, 0xf0, 0x66, 0x1e, 0x15, 0x38, 0xd6,
	0xec, 0x58, 0x5f, 0xe4, 0xd7, 0x49, 0x5f, 0x69, 0x52, 0x34, 0x89, 0x86, 0xbb, 0x17, 0xc5, 0x6c,
	0x0f, 0xfb, 0x98, 0xdd, 0x30, 0x7d, 0x88, 0xb9, 0x0b, 0x2b, 0xac, 0x1f, 0xc2, 0x06, 0x56, 0x5d,
	0xcb, 0xa6, 0x0f, 0x0e, 0xb1, 0xbd, 0x54, 0x96, 0xec, 0x6b, 0xf1, 0x6d, 0xe8, 0xc3, 0x51, 0xb1,
	0x00, 0x34, 0x9b, 0x5c, 0x8a, 0x28, 0x61, 0x34, 0x1b, 0x25, 0xf6, 0x07, 0xaf, 0x9b, 0x80, 0xa2,
	0x69, 0xe4, 0xe7, 0x02, 0x20, 0x5a, 0xb7, 0xe2, 0x97, 0x58, 0x1d, 0xcc, 0xdf, 0x68, 0x55, 0xee,
	0xfa, 0x9e, 0x54, 0x9a, 0x16, 0xa0, 0x69, 0xc5, 0x5a, 0xe5, 0x88, 0xd1, 0x24, 0x26, 0x12, 0x6d,
	0xc1, 0x12, 0x59, 0x79, 0x9b, 0xe1, 0xf2, 0xcf, 0x0b, 0xf0, 0xc1, 0xd4, 0x87, 0x5c, 0xce, 0xad,
	0xaa, 0xd6, 0xc5, 0x48, 0x0a, 0x45, 0x9b, 0x35, 0xdf, 0x93, 0x32, 0x41, 0x67, 0xa3, 0x6b, 0x2c,
	0xe0, 0x9c, 0x4e, 0xfe, 0xf5, 0x31, 0xd6, 0x23, 0x4d, 0xbe, 0xc9, 0xd7, 0xf6, 0x18, 0x9d, 0x83,
	0xb3, 0x99, 0xf7, 0x5f, 0x9e, 0xc8, 0x9f, 0x3a, 0x6e, 0x4c, 0x14, 0xf9, 0xc6, 0x4b, 0xde, 0xeb,
	0xdb, 0x90, 0x0a, 0x9c, 0x97, 0x64, 0x91, 0x5a, 0xe3, 0xa1, 0x5c, 0x16, 0x2f, 0x90, 0xdc, 0xf1,
	0xb4, 0x29, 0x3f, 0xae, 0xca, 0xa2, 0xc0, 0xf2, 0x48, 0xab, 0x5d, 0x95, 0xc5, 0x85, 0xeb, 0x65,
	0x48, 0x05, 0xfc, 0x40, 0x97, 0x60, 0x8d, 0xe6, 0xa4, 0x4e, 0x6b, 0xb7, 0xdc, 0xe8, 0xb4, 0x9f,
	0xed, 0x57, 0xc5, 0x0b, 0x68, 0x05, 0xd2, 0x8d, 0xe6, 0x5e, 0x95, 0xce, 0x89, 0x02, 0xba, 0x08,
	0x2b, 0xfb, 0xf5, 0x72, 0xfb, 0x61, 0x53, 0x7e, 0xc2, 0xa6, 0x16, 0xae, 0xff, 0x5e, 0x80, 0x6c,
	0xb8, 0x50, 0x45, 0xef, 0xc0, 0x25, 0x86, 0x23, 0x1f, 0xd4, 0xab, 0x9d, 0x56, 0xf5, 0xb0, 0x2a,
	0xd7, 0xda, 0xcf, 0xc4, 0x0b, 0xe8, 0x5d, 0xb8, 0x7c, 0xd0, 0x78, 0xdc, 0x68, 0x3e, 0x6d, 0x8c,
	0x2d, 0x09, 0xe8, 0x0a, 0x20, 0x92, 0xe5, 0xc6, 0xe6, 0x17, 0xd0, 0x65, 0xb8, 0x58, 0x6f, 0x3e,
	0x1d, 0x9b, 0x5e, 0x44, 0x39, 0x58, 0x7f, 0x52, 0xdd, 0xab, 0x1d, 0x3c, 0x19, 0x5b, 0x59, 0x22,
	0x40, 0x8f, 0x6a, 0x9f, 0x3e, 0x1a, 0x9b, 0x4f, 0x54, 0x6e, 0xff, 0xe5, 0xd5, 0x35, 0xe1, 0x6f,
	0xaf, 0xae, 0x09, 0xff, 0x7a, 0x75, 0x4d, 0xf8, 0xed, 0xbf, 0xaf, 0x5d, 0x80, 0x77, 0x75, 0xab,
	0xe8, 0xb8, 0x8a, 0x7a, 0x62, 0x5b, 0x2f, 0x19, 0x8f, 0x02, 0x36, 0x7f, 0x3f, 0xf8, 0xd3, 0xf1,
	0x79, 0x92, 0xce, 0x7f, 0xfc, 0xdf, 0x00, 0x00, 0x00, 0xff, 0xff, 0x7f, 0x9a, 0x66, 0x68, 0xac,
	0x1c, 0x00, 0x00,
>>>>>>> 4aa7f423fb (X-Smart-Squash: Squashed 5 commits:)
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
	if len(m.ProfileName) > 0 {
		i -= len(m.ProfileName)
		copy(dAtA[i:], m.ProfileName)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ProfileName)))
		i--
		dAtA[i] = 0x12
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
		for iNdEx := len(m.ProductType) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.ProductType[iNdEx])
			copy(dAtA[i:], m.ProductType[iNdEx])
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ProductType[iNdEx])))
			i--
			dAtA[i] = 0x2a
		}
	}
	if len(m.ProfileVersion) > 0 {
		i -= len(m.ProfileVersion)
		copy(dAtA[i:], m.ProfileVersion)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ProfileVersion)))
		i--
<<<<<<< HEAD
=======
		dAtA[i] = 0x2a
	}
	if len(m.OperatorVersion) > 0 {
		i -= len(m.OperatorVersion)
		copy(dAtA[i:], m.OperatorVersion)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.OperatorVersion)))
		i--
>>>>>>> 4aa7f423fb (X-Smart-Squash: Squashed 5 commits:)
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
	if len(m.Fixes) > 0 {
		i -= len(m.Fixes)
		copy(dAtA[i:], m.Fixes)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Fixes)))
		i--
		dAtA[i] = 0x5a
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
	if len(m.RuleVersion) > 0 {
		i -= len(m.RuleVersion)
		copy(dAtA[i:], m.RuleVersion)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.RuleVersion)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.OperatorVersion) > 0 {
		i -= len(m.OperatorVersion)
		copy(dAtA[i:], m.OperatorVersion)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.OperatorVersion)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Name) > 0 {
		i -= len(m.Name)
		copy(dAtA[i:], m.Name)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Name)))
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
	if len(m.ScanName) > 0 {
		i -= len(m.ScanName)
		copy(dAtA[i:], m.ScanName)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ScanName)))
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
	if len(m.ScanId) > 0 {
		i -= len(m.ScanId)
		copy(dAtA[i:], m.ScanId)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ScanId)))
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
	if len(m.ScanConfigName) > 0 {
		i -= len(m.ScanConfigName)
		copy(dAtA[i:], m.ScanConfigName)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ScanConfigName)))
		i--
		dAtA[i] = 0x1
		i--
		dAtA[i] = 0x8a
	}
	if len(m.ScanConfigId) > 0 {
		i -= len(m.ScanConfigId)
		copy(dAtA[i:], m.ScanConfigId)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ScanConfigId)))
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
	if len(m.ScanId) > 0 {
		i -= len(m.ScanId)
		copy(dAtA[i:], m.ScanId)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ScanId)))
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
		for iNdEx := len(m.Warnings) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Warnings[iNdEx])
			copy(dAtA[i:], m.Warnings[iNdEx])
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Warnings[iNdEx])))
			i--
			dAtA[i] = 0x1a
		}
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
	if len(m.ScanName) > 0 {
		i -= len(m.ScanName)
		copy(dAtA[i:], m.ScanName)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ScanName)))
		i--
		dAtA[i] = 0x72
	}
	if len(m.ClusterName) > 0 {
		i -= len(m.ClusterName)
		copy(dAtA[i:], m.ClusterName)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ClusterName)))
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
	if len(m.Profile) > 0 {
		for iNdEx := len(m.Profile) - 1; iNdEx >= 0; iNdEx-- {
			{
				size, err := m.Profile[iNdEx].MarshalToSizedBuffer(dAtA[:i])
				if err != nil {
					return 0, err
				}
				i -= size
				i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(size))
			}
			i--
			dAtA[i] = 0x2a
		}
	}
	if len(m.Errors) > 0 {
		for iNdEx := len(m.Errors) - 1; iNdEx >= 0; iNdEx-- {
			i -= len(m.Errors[iNdEx])
			copy(dAtA[i:], m.Errors[iNdEx])
			i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.Errors[iNdEx])))
			i--
			dAtA[i] = 0x22
		}
	}
	if len(m.ClusterId) > 0 {
		i -= len(m.ClusterId)
		copy(dAtA[i:], m.ClusterId)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ClusterId)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.ScanConfigId) > 0 {
		i -= len(m.ScanConfigId)
		copy(dAtA[i:], m.ScanConfigId)
		i = encodeVarintComplianceOperatorV2(dAtA, i, uint64(len(m.ScanConfigId)))
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
	l = len(m.ProfileName)
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
<<<<<<< HEAD
=======
	l = len(m.OperatorVersion)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
>>>>>>> 4aa7f423fb (X-Smart-Squash: Squashed 5 commits:)
	l = len(m.ProfileVersion)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if len(m.ProductType) > 0 {
		for _, s := range m.ProductType {
			l = len(s)
			n += 1 + l + sovComplianceOperatorV2(uint64(l))
		}
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
	l = len(m.Name)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.OperatorVersion)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.RuleVersion)
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
	l = len(m.Fixes)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
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
	l = len(m.ScanName)
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
	l = len(m.ScanId)
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
	l = len(m.ScanId)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.ClusterName)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.ScanConfigId)
	if l > 0 {
		n += 2 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.ScanConfigName)
	if l > 0 {
		n += 2 + l + sovComplianceOperatorV2(uint64(l))
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
	if len(m.Warnings) > 0 {
		for _, s := range m.Warnings {
			l = len(s)
			n += 1 + l + sovComplianceOperatorV2(uint64(l))
		}
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
	l = len(m.ScanConfigId)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.ClusterId)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	if len(m.Errors) > 0 {
		for _, s := range m.Errors {
			l = len(s)
			n += 1 + l + sovComplianceOperatorV2(uint64(l))
		}
	}
	if len(m.Profile) > 0 {
		for _, e := range m.Profile {
			l = e.Size()
			n += 1 + l + sovComplianceOperatorV2(uint64(l))
		}
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
	l = len(m.ClusterName)
	if l > 0 {
		n += 1 + l + sovComplianceOperatorV2(uint64(l))
	}
	l = len(m.ScanName)
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
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ProfileName", wireType)
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
			m.ProfileName = string(dAtA[iNdEx:postIndex])
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
<<<<<<< HEAD
=======
				return fmt.Errorf("proto: wrong wireType = %d for field OperatorVersion", wireType)
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
			m.OperatorVersion = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 5:
			if wireType != 2 {
>>>>>>> 4aa7f423fb (X-Smart-Squash: Squashed 5 commits:)
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
			m.ProductType = append(m.ProductType, string(dAtA[iNdEx:postIndex]))
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
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field OperatorVersion", wireType)
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
			m.OperatorVersion = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field RuleVersion", wireType)
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
			m.RuleVersion = string(dAtA[iNdEx:postIndex])
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
			m.Fixes = string(dAtA[iNdEx:postIndex])
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
				return fmt.Errorf("proto: wrong wireType = %d for field ScanId", wireType)
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
			m.ScanId = string(dAtA[iNdEx:postIndex])
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
				return fmt.Errorf("proto: wrong wireType = %d for field ScanId", wireType)
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
			m.ScanId = string(dAtA[iNdEx:postIndex])
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
		case 17:
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
			m.Errors = append(m.Errors, string(dAtA[iNdEx:postIndex]))
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
			m.Profile = append(m.Profile, &ProfileShim{})
			if err := m.Profile[len(m.Profile)-1].Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
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
