package complianceoperator

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/sensor/common"
)

const (
	rescanAnnotation = v1alpha1.ComplianceScanRescanAnnotation
)

// defaultNodeRoles returns the backward-compatible default roles used when
// Central does not specify any (e.g. old Central version).
// Central always populates BaseScanSettings.node_roles once compliance-v2 sync is
// supported; conversely, old Sensor builds that predate the node_roles field ignore
// it via normal protobuf forward-compatibility and fall back to this same default.
// See the node_roles comment in proto/internalapi/central/compliance_operator.proto.
// Keep in sync with central/complianceoperator/v2/scanconfigurations/service/convert.go
// and the UI default in ui/.../Schedules/compliance.scanConfigs.utils.tsx.
func defaultNodeRoles() []string {
	return []string{"master", "worker"}
}

var (
	defaultScanSettingName = "default-" + branding.GetProductNameShort()
)

// StatusInfo is an interface that provides functionality to fetch compliance operator info.
//
//go:generate mockgen-wrapper
type StatusInfo interface {
	GetNamespace() string
}

// InfoUpdater is an interface that provides functionality to periodically scan secured cluster for compliance operator info.
//
//go:generate mockgen-wrapper
type InfoUpdater interface {
	common.SensorComponent
	StatusInfo
}
