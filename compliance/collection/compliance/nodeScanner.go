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
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/mtls"
	scannerV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
)

type NodeInventoryComponentScanner struct {
	log              *logging.Logger
	nodeNameProvider NodeNameProvider
	client           scannerV1.NodeInventoryServiceClient
}

func NewNodeInventoryComponentScanner(log *logging.Logger, nnp NodeNameProvider) *NodeInventoryComponentScanner {
	return &NodeInventoryComponentScanner{
		log:              log,
		nodeNameProvider: nnp,
	}
}

func (n *NodeInventoryComponentScanner) IsActive() bool {
	return n.client != nil
}

func (n *NodeInventoryComponentScanner) Connect(address string) {
	if !env.NodeInventoryContainerEnabled.BooleanSetting() {
		n.log.Infof("Compliance will not call the node-inventory container, because this is not Openshift 4 cluster")
	} else if env.RHCOSNodeScanning.BooleanSetting() {
		// Start the prometheus metrics server
		metrics.NewDefaultHTTPServer(metrics.ComplianceSubsystem).RunForever()
		metrics.GatherThrottleMetricsForever(metrics.ComplianceSubsystem.String())

		// Set up Compliance <-> NodeInventory connection
		niConn, err := clientconn.AuthenticatedGRPCConnection(address, mtls.Subject{}, clientconn.UseInsecureNoTLS(true))
		if err != nil {
			n.log.Errorf("Disabling node scanning for this node: could not initialize connection to node-inventory container: %v", err)
		}
		if niConn != nil {
			n.log.Info("Initialized gRPC connection to node-inventory container")
			n.client = scannerV1.NewNodeInventoryServiceClient(niConn)
		}
	}
}

func (n *NodeInventoryComponentScanner) ManageNodeScanLoop(ctx context.Context, i intervals.NodeScanIntervals) <-chan *sensor.MsgFromCompliance {
	nodeInventoriesC := make(chan *sensor.MsgFromCompliance)
	nodeName := n.nodeNameProvider.GetNodeName()
	go func() {
		defer close(nodeInventoriesC)
		t := time.NewTicker(i.Initial())
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				n.log.Infof("Scanning node %q", nodeName)
				msg, err := n.ScanNode(ctx)
				if err != nil {
					n.log.Errorf("error running node scan: %v", err)
				} else {
					nodeInventoriesC <- msg
				}
				interval := i.Next()
				cmetrics.ObserveRescanInterval(interval, n.nodeNameProvider.GetNodeName())
				t.Reset(interval)
			}
		}
	}()
	return nodeInventoriesC
}

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
