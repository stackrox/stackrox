package compliance

import (
	"context"
	"time"

	"github.com/stackrox/rox/compliance/collection/intervals"
	"github.com/stackrox/rox/compliance/collection/inventory"
	cmetrics "github.com/stackrox/rox/compliance/collection/metrics"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/mtls"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

// NodeInventoryComponentScanner connects to node-inventory container to provide node-inventory object
type NodeInventoryComponentScanner struct {
	nodeNameProvider NodeNameProvider
	client           scannerV1.NodeInventoryServiceClient
}

// NewNodeInventoryComponentScanner builds new NodeInventoryComponentScanner
func NewNodeInventoryComponentScanner(nnp NodeNameProvider) *NodeInventoryComponentScanner {
	return &NodeInventoryComponentScanner{nodeNameProvider: nnp}
}

// IsActive returns true if the connection to node-inventory is ready
func (n *NodeInventoryComponentScanner) IsActive() bool {
	return n.client != nil
}

// Connect connects to node-inventory and stores an active client
func (n *NodeInventoryComponentScanner) Connect(address string) {
	if !env.NodeInventoryContainerEnabled.BooleanSetting() {
		log.Infof("Compliance will not call the node-inventory container, because this is not Openshift 4 cluster")
	} else if env.RHCOSNodeScanning.BooleanSetting() {
		// Start the prometheus metrics server
		metrics.NewServer(metrics.NodeInventorySubsystem, metrics.NewTLSConfigurerFromEnv()).RunForever()
		metrics.GatherThrottleMetricsForever(metrics.NodeInventorySubsystem.String())

		// Set up Compliance <-> NodeInventory connection
		niConn, err := clientconn.AuthenticatedGRPCConnection(address, mtls.Subject{}, clientconn.UseInsecureNoTLS(true))
		if err != nil {
			log.Errorf("Disabling node scanning for this node: could not initialize connection to node-inventory container: %v", err)
		}
		if niConn != nil {
			log.Info("Initialized gRPC connection to node-inventory container")
			n.client = scannerV1.NewNodeInventoryServiceClient(niConn)
		}
	}
}

// GetIntervals returns node scan intervals (initial scan delay, regular scan delay)
func (n *NodeInventoryComponentScanner) GetIntervals() *intervals.NodeScanIntervals {
	i := intervals.NewNodeScanIntervalFromEnv()
	return &i
}

// ScanNode returns a message with node-inventory
func (n *NodeInventoryComponentScanner) ScanNode(ctx context.Context) (*sensor.MsgFromCompliance, error) {
	ctx, cancel := context.WithTimeout(ctx, env.NodeAnalysisDeadline.DurationSetting())
	defer cancel()
	startCall := time.Now()
	result, err := n.client.GetNodeInventory(ctx, &scannerV1.GetNodeInventoryRequest{})
	if err != nil {
		return nil, err
	}
	cmetrics.ObserveNodeInventoryCallDuration(time.Since(startCall), result.GetNodeName(), err)
	inv := inventory.ToNodeInventory(result)
	msg := &sensor.MsgFromCompliance{
		Node: result.GetNodeName(),
		Msg:  &sensor.MsgFromCompliance_NodeInventory{NodeInventory: inv},
	}
	cmetrics.ObserveInventoryProtobufMessage(msg)
	return msg, nil
}
