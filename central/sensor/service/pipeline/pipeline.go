package pipeline

import (
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
)

// BasePipeline represents methods that are shared between Pipelines and fragments of pipelines.
type BasePipeline interface {
	OnFinish()
}

// ClusterPipeline processes a message received from a given cluster.
//go:generate mockgen-wrapper ClusterPipeline
type ClusterPipeline interface {
	Run(msg *central.MsgFromSensor, injector common.MessageInjector) error
	BasePipeline
}

// Factory returns a ClusterPipeline for the given cluster.
type Factory interface {
	PipelineForCluster(clusterID string) (ClusterPipeline, error)
}

// Fragment is a component of a Pipeline that only processes specific messages.
// Fragments can be either local or global across clusters;
// they are passed clusterIDs along with every event, which they are free to use.
//go:generate mockgen-wrapper Fragment
type Fragment interface {
	BasePipeline
	Match(msg *central.MsgFromSensor) bool
	Run(clusterID string, msg *central.MsgFromSensor, injector common.MessageInjector) error
	Reconcile(clusterID string) error
}

// FragmentFactory returns a Fragment for the given cluster.
type FragmentFactory interface {
	GetFragment(clusterID string) (Fragment, error)
}
