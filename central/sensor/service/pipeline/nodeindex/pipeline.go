package nodeindex

import (
	"context"

	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/enrichment"
	nodeDatastore "github.com/stackrox/rox/central/node/datastore"
	riskManager "github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/nodes/enricher"
)

var (
	log = logging.LoggerForModule()

	_ pipeline.Fragment = (*pipelineImpl)(nil)
)

// GetPipeline returns an instantiation of this particular pipeline
func GetPipeline() pipeline.Fragment {
	return newPipeline(clusterDataStore.Singleton(), nodeDatastore.Singleton(), enrichment.NodeEnricherSingleton(), riskManager.Singleton())
}

// newPipeline returns a new instance of Pipeline.
func newPipeline(clusters clusterDataStore.DataStore, nodes nodeDatastore.DataStore, enricher enricher.NodeEnricher, riskManager riskManager.Manager) pipeline.Fragment {
	return &pipelineImpl{
		clusterStore:  clusters,
		nodeDatastore: nodes,
		enricher:      enricher,
		riskManager:   riskManager,
	}
}

type pipelineImpl struct {
	clusterStore  clusterDataStore.DataStore
	nodeDatastore nodeDatastore.DataStore
	enricher      enricher.NodeEnricher
	riskManager   riskManager.Manager
}

func (p pipelineImpl) OnFinish(_ string) {
}

func (p pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (p pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetIndexReport() != nil
}

func (p pipelineImpl) Run(ctx context.Context, _ string, msg *central.MsgFromSensor, _ common.MessageInjector) error {
	if !env.NodeIndexEnabled.BooleanSetting() || !features.ScannerV4.Enabled() {
		// Node Indexing only works correctly when both, itself and Scanner v4 are enabled
		log.Debugf("Skipping node index message (Node Indexing Enabled: %t, Scanner V4 Enabled: %t",
			env.NodeIndexEnabled.BooleanSetting(), features.ScannerV4.Enabled())
		return nil
	}
	event := msg.GetEvent()
	report := event.GetIndexReport()
	if report == nil {
		return errors.Errorf("unexpected resource type %T for index report", event.GetResource())
	}
	if event.GetAction() == central.ResourceAction_REMOVE_RESOURCE {
		log.Warn("Removal of node index is unsupported action")
		return nil
	}
	log.Debugf("received node index report for node %s with %d packages from %d content sets",
		event.GetId(), len(report.GetContents().Packages), len(report.GetContents().Repositories))
	report = report.CloneVT()

	// Query storage for the node this report comes from
	nodeId := event.GetId()
	node, found, err := p.nodeDatastore.GetNode(ctx, nodeId)
	if err != nil {
		return errors.WithMessagef(err, "failed to fetch node %s from database", nodeId)
	}
	if !found {
		return errors.WithMessagef(err, "node %s not found in datastore", nodeId)
	}

	// Send the Node and Index Report to Scanner for enrichment. The result will be persisted in node.NodeScan
	err = p.enricher.EnrichNodeWithVulnerabilities(node, nil, report)
	if err != nil {
		return errors.WithMessagef(err, "enriching node %s with index report", nodeId)
	}
	log.Infof("Scanned index report and found %d components for node %s",
		len(node.GetScan().GetComponents()), nodeDatastore.NodeString(node))

	// Update the whole node in the database with the new and previous information.
	err = p.riskManager.CalculateRiskAndUpsertNode(node)
	if err != nil {
		return errors.Wrapf(err, "failed calculating risk and upserting node %s", nodeDatastore.NodeString(node))
	}

	return nil
}

func (p pipelineImpl) Reconcile(_ context.Context, _ string, _ *reconciliation.StoreMap) error {
	return nil
}
