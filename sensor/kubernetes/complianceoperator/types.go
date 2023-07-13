package complianceoperator

import (
	"github.com/stackrox/rox/sensor/common"
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
