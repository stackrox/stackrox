package deploymentreconciler

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
)

var _ common.SensorComponent = (*deploymentReconcilerImpl)(nil)

type DeploymentReconciler interface {
	common.SensorComponent
	OnDeploymentRemove(*storage.Deployment)
}

func NewDeploymentReconciler() *deploymentReconcilerImpl {
	return &deploymentReconcilerImpl{
		deployments: make(map[string]*storage.Deployment),
		toCentral:   make(chan *message.ExpiringMessage),
		stopper:     concurrency.NewStopper(),
	}
}
