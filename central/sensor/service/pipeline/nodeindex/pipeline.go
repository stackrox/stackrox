package nodeindex

import (
	"context"

	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/enrichment"
	countMetrics "github.com/stackrox/rox/central/metrics"
	nodeDatastore "github.com/stackrox/rox/central/node/datastore"
	riskManager "github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
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

func (p *pipelineImpl) OnFinish(_ string) {
}

func (p *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (p *pipelineImpl) Reconcile(_ context.Context, _ string, _ *reconciliation.StoreMap) error {
	return nil
}

func (p *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetIndexReport() != nil
}

func (p *pipelineImpl) Run(ctx context.Context, _ string, msg *central.MsgFromSensor, injector common.MessageInjector) error {
	if !features.NodeIndexEnabled.Enabled() || !features.ScannerV4.Enabled() {
		// Node Indexing only works correctly when both, itself and Scanner v4 are enabled
		log.Debugf("Skipping node index message (Node Indexing Enabled: %t, Scanner V4 Enabled: %t",
			features.NodeIndexEnabled.Enabled(), features.ScannerV4.Enabled())
		// ACK the message to prevent frequent retries.
		// If support for NodeIndex is disabled on Central or Scanner V4 is missing, but NodeIndex msg arrives to Central,
		// then we must acknowledge the reception to prevent the compliance container resending the message, as
		// this could flood Sensor and Central unnecessarily.
		// Such situations may occur when some secured clusters have NodeIndexing enabled, while Central has it disabled,
		// or Scanner V4 is not deployed.
		sendComplianceAck(ctx, msg.GetEvent().GetNode(), injector)
		return nil
	}
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.NodeIndex)

	event := msg.GetEvent()
	report := event.GetIndexReport()
	if report == nil {
		return errors.Errorf("unexpected resource type %T for index report", event.GetResource())
	}
	if event.GetAction() == central.ResourceAction_REMOVE_RESOURCE {
		log.Warn("Removal of node index is unsupported action")
		return nil
	}
	log.Debugf("Received node index report for node %s with %d packages from %d content sets",
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

	sendComplianceAck(ctx, node, injector)
	return nil
}

func sendComplianceAck(ctx context.Context, node *storage.Node, injector common.MessageInjector) {
	if injector == nil {
		return
	}
	reply := replyCompliance(node.GetClusterId(), node.GetName(), central.NodeInventoryACK_ACK)
	if err := injector.InjectMessage(ctx, reply); err != nil {
		log.Warnf("Failed sending node-indexing-ACK to Sensor for %s: %v", nodeDatastore.NodeString(node), err)
	} else {
		log.Debugf("Sent node-indexing-ACK for %s", nodeDatastore.NodeString(node))
	}
}

func replyCompliance(clusterID, nodeName string, t central.NodeInventoryACK_Action) *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_NodeInventoryAck{
			NodeInventoryAck: &central.NodeInventoryACK{
				ClusterId:   clusterID,
				NodeName:    nodeName,
				Action:      t,
				MessageType: central.NodeInventoryACK_NodeIndexer,
			},
		},
	}
}
