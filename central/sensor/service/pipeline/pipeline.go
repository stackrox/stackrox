package pipeline

import (
	"context"

	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
)

// BasePipeline represents methods that are shared between Pipelines and fragments of pipelines.
type BasePipeline interface {
	OnFinish(clusterID string)

	Capabilities() []centralsensor.CentralCapability
}

// ClusterPipeline processes a message received from a given cluster.
//
//go:generate mockgen-wrapper
type ClusterPipeline interface {
	BasePipeline

	Reconcile(ctx context.Context, reconciliationStore *reconciliation.StoreMap) error
	Run(ctx context.Context, msg *central.MsgFromSensor, injector common.MessageInjector) error
}

// Factory returns a ClusterPipeline for the given cluster.
type Factory interface {
	PipelineForCluster(ctx context.Context, clusterID string) (ClusterPipeline, error)
}

// Fragment is a component of a Pipeline that only processes specific messages.
// Fragments can be either local or global across clusters;
// they are passed clusterIDs along with every event, which they are free to use.
//
//go:generate mockgen-wrapper
type Fragment interface {
	BasePipeline

	Match(msg *central.MsgFromSensor) bool

	Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, injector common.MessageInjector) error
	Reconcile(ctx context.Context, clusterID string, reconciliationStore *reconciliation.StoreMap) error
}

// FragmentFactory returns a Fragment for the given cluster.
type FragmentFactory interface {
	GetFragment(ctx context.Context, clusterID string) (Fragment, error)
}
