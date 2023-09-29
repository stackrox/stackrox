package nodeinventory

import (
	"context"
	"fmt"

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
	"github.com/stackrox/rox/pkg/centralsensor"
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

func (p *pipelineImpl) Capabilities() []centralsensor.CentralCapability {
	return nil
}

func (p *pipelineImpl) Reconcile(_ context.Context, _ string, _ *reconciliation.StoreMap) error {
	return nil
}

func (p *pipelineImpl) Match(msg *central.MsgFromSensor) bool {
	return msg.GetEvent().GetNodeInventory() != nil
}

// Run runs the pipeline template on the input and returns the output.
func (p *pipelineImpl) Run(ctx context.Context, _ string, msg *central.MsgFromSensor, injector common.MessageInjector) error {
	defer countMetrics.IncrementResourceProcessedCounter(pipeline.ActionToOperation(msg.GetEvent().GetAction()), metrics.NodeInventory)

	// Sanitize input.
	event := msg.GetEvent()
	ninv := event.GetNodeInventory()
	if ninv == nil {
		return errors.Errorf("unexpected resource type %T for node inventory", event.GetResource())
	}
	nodeStr := fmt.Sprintf("(node name: %q, node id: %q)", ninv.GetNodeName(), ninv.GetNodeId())
	log.Debugf("received inventory %s contains %d packages to scan from %d content sets", nodeStr,
		len(ninv.GetComponents().GetRhelComponents()), len(ninv.GetComponents().GetRhelContentSets()))
	if event.GetAction() != central.ResourceAction_UNSET_ACTION_RESOURCE {
		log.Errorf("inventory %s has unsupported action: %q", nodeStr, event.GetAction())
		return nil
	}
	ninv = ninv.Clone()

	// Read the node from the database, if not found we fail.
	node, found, err := p.nodeDatastore.GetNode(ctx, ninv.GetNodeId())
	if err != nil {
		log.Errorf("fetching node %s from the database: %v", nodeStr, err)
		return errors.WithMessagef(err, "fetching node: %s", ninv.GetNodeId())
	}
	if !found {
		log.Errorf("fetching node %s from the database: node does not exist", nodeStr)
		return errors.WithMessagef(err, "node does not exist: %s", ninv.GetNodeId())
	}

	// Call Scanner to enrich the node inventory and attach the results to the node object.
	err = p.enricher.EnrichNodeWithInventory(node, ninv)
	if err != nil {
		log.Errorf("enriching node %s: %v", nodeDatastore.NodeString(node), err)
		return errors.WithMessagef(err, "enrinching node %s", nodeDatastore.NodeString(node))
	}
	log.Infof("Scanned node inventory %s with %d components", nodeDatastore.NodeString(node),
		len(node.GetScan().GetComponents()))

	// Update the whole node in the database with the new and previous information.
	err = p.riskManager.CalculateRiskAndUpsertNode(node)
	if err != nil {
		log.Error(err)
		return err
	}

	if injector != nil {
		reply := replyCompliance(node.GetClusterId(), ninv.GetNodeName(), central.NodeInventoryACK_ACK)
		if err := injector.InjectMessage(ctx, reply); err != nil {
			log.Warnf("Failed sending node-scanning-ACK to Sensor for %s: %v", nodeDatastore.NodeString(node), err)
		} else {
			log.Debugf("Sent node-scanning-ACK for %s", nodeDatastore.NodeString(node))
		}
	}
	return nil
}

func replyCompliance(clusterID, nodeName string, t central.NodeInventoryACK_Action) *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_NodeInventoryAck{
			NodeInventoryAck: &central.NodeInventoryACK{
				ClusterId: clusterID,
				NodeName:  nodeName,
				Action:    t,
			},
		},
	}
}

func (p *pipelineImpl) OnFinish(_ string) {}
