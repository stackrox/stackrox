package inventory

import (
	"context"
	"time"

	cmetrics "github.com/stackrox/rox/compliance/collection/metrics"
	"github.com/stackrox/rox/compliance/node"
	"github.com/stackrox/rox/compliance/utils"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

var (
	log = logging.LoggerForModule()
)

// NodeInventoryComponentScanner connects to node-inventory container to provide node-inventory object
type NodeInventoryComponentScanner struct {
	nodeNameProvider node.NodeNameProvider
	client           scannerV1.NodeInventoryServiceClient
}

// NewNodeInventoryComponentScanner builds new NodeInventoryComponentScanner
func NewNodeInventoryComponentScanner(nnp node.NodeNameProvider) *NodeInventoryComponentScanner {
	return &NodeInventoryComponentScanner{nodeNameProvider: nnp}
}

// IsActive returns true if the connection to node-inventory is ready
func (n *NodeInventoryComponentScanner) IsActive() bool {
	return n.client != nil
}

// Connect connects to node-inventory and stores an active client
func (n *NodeInventoryComponentScanner) Connect(address string) {
	if !env.NodeInventoryContainerEnabled.BooleanSetting() {
		log.Info("Compliance will not call the node-inventory container, because this feature is disabled")
		return
	}
	// Set up Compliance <-> NodeInventory connection
	niConn, err := clientconn.AuthenticatedGRPCConnection(context.Background(), address, mtls.Subject{}, clientconn.UseInsecureNoTLS(true))
	if err != nil {
		log.Errorf("Disabling node scanning for this node: could not initialize connection to node-inventory container: %v", err)
	}
	if niConn != nil {
		log.Info("Initialized gRPC connection to node-inventory container")
		n.client = scannerV1.NewNodeInventoryServiceClient(niConn)
	}
}

// GetIntervals returns node scan intervals (initial scan delay, regular scan delay)
func (n *NodeInventoryComponentScanner) GetIntervals() *utils.NodeScanIntervals {
	i := utils.NewNodeScanIntervalFromEnv()
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
	inv := ToNodeInventory(result)
	msg := &sensor.MsgFromCompliance{
		Node: result.GetNodeName(),
		Msg:  &sensor.MsgFromCompliance_NodeInventory{NodeInventory: inv},
	}
	cmetrics.ObserveReportProtobufMessage(msg, cmetrics.ScannerVersionV2)
	return msg, nil
}
