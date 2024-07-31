package new

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
)

// ConvertComplianceOperatorScanV2FromProto converts a `*storage.ComplianceOperatorScanV2` to Gorm model
func ConvertComplianceOperatorScanV2FromProto(obj *storage.ComplianceOperatorScanV2) (*ComplianceOperatorScanV2, error) {
	serialized, err := obj.MarshalVT()
	if err != nil {
		return nil, err
	}
	model := &ComplianceOperatorScanV2{
		ID:                  obj.GetId(),
		ScanConfigName:      obj.GetScanConfigName(),
		ClusterID:           obj.GetClusterId(),
		ProfileProfileRefID: obj.GetProfile().GetProfileRefId(),
		StatusResult:        obj.GetStatus().GetResult(),
		LastExecutedTime:    protocompat.NilOrTime(obj.GetLastExecutedTime()),
		ScanName:            obj.GetScanName(),
		ScanRefID:           obj.GetScanRefId(),
		Serialized:          serialized,
	}
	return model, nil
}

// ConvertComplianceOperatorScanV2ToProto converts Gorm model `ComplianceOperatorScanV2` to its protobuf type object
func ConvertComplianceOperatorScanV2ToProto(m *ComplianceOperatorScanV2) (*storage.ComplianceOperatorScanV2, error) {
	var msg storage.ComplianceOperatorScanV2
	if err := msg.Unmarshal(m.Serialized); err != nil {
		return nil, err
	}
	return &msg, nil
}
