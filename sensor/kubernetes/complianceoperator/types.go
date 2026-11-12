package complianceoperator

import (
	"time"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/pkg/branding"
	"github.com/stackrox/rox/sensor/common"
)

const (
	masterRole                 = "master"
	workerRole                 = "worker"
	rescanAnnotation           = v1alpha1.ComplianceScanRescanAnnotation
	defaultMaxRetries          = 5
	defaultAPICallTimeout      = 5 * time.Second
	defaultRetryTimeout        = 30 * time.Second
	defaultMCPDebounceInterval = 5 * time.Second
)

type scanScheduleConfiguration struct {
	Suspend  *bool
	ScanName string
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
