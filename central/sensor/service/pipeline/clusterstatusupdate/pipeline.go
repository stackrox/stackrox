package clusterstatusupdate

import (
	"context"

	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/deploymentenvs"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Template design pattern. We define control flow here and defer logic to subclasses.
//////////////////////////////////////////////////////////////////////////////////////

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return NewPipeline(clusterDataStore.Singleton(), deploymentenvs.ManagerSingleton())
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(clusters clusterDataStore.DataStore, deploymentEnvsMgr deploymentenvs.Manager) pipeline.Fragment {
	return &pipelineImpl{
		clusters:          clusters,
		deploymentEnvsMgr: deploymentEnvsMgr,
	}
}

type pipelineImpl struct {
	clusters          clusterDataStore.DataStore
	deploymentEnvsMgr deploymentenvs.Manager
}

func (s *pipelineImpl) Reconcile(_ context.Context, _ string) error {
	// Nothing to reconcile
	return nil
}

func (s *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetClusterStatusUpdate() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (s *pipelineImpl) Run(ctx context.Context, clusterID string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	switch m := msg.GetClusterStatusUpdate().Msg.(type) {
	case *central.ClusterStatusUpdate_DeploymentEnvUpdate:
		s.deploymentEnvsMgr.UpdateDeploymentEnvironments(clusterID, m.DeploymentEnvUpdate.Environments)
		return nil
	case *central.ClusterStatusUpdate_Status:
		return s.clusters.UpdateClusterStatus(ctx, clusterID, m.Status)
	default:
		return errors.Errorf("unknown cluster status update message type %T", m)
	}
}

func (s *pipelineImpl) OnFinish(clusterID string) {
	s.deploymentEnvsMgr.MarkClusterInactive(clusterID)
}
