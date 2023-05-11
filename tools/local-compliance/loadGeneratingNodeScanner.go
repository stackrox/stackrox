package main

import (
	"context"
	"time"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/compliance/collection/compliance"
	"github.com/stackrox/rox/compliance/collection/intervals"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/uuid"
)

type LoadGeneratingNodeScanner struct {
	log          *logging.Logger
	nodeProvider compliance.NodeNameProvider
}

func (n *LoadGeneratingNodeScanner) IsActive() bool {
	return true
}

func (n *LoadGeneratingNodeScanner) Connect(address string) {}

func (n *LoadGeneratingNodeScanner) ManageNodeScanLoop(ctx context.Context, i intervals.NodeScanIntervals) <-chan *sensor.MsgFromCompliance {
	nodeInventoriesC := make(chan *sensor.MsgFromCompliance)
	nodeName := n.nodeProvider.GetNodeName()
	go func() {
		defer close(nodeInventoriesC)
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
				n.log.Infof("Scanning node %q", nodeName)
				msg, err := n.ScanNode(ctx)
				if err != nil {
					n.log.Errorf("error running node scan: %v", err)
				} else {
					nodeInventoriesC <- msg
				}
			}
		}
	}()
	return nodeInventoriesC
}

func (n *LoadGeneratingNodeScanner) ScanNode(_ context.Context) (*sensor.MsgFromCompliance, error) {
	msg := &sensor.MsgFromCompliance{
		Node: n.nodeProvider.GetNodeName(),
		Msg: &sensor.MsgFromCompliance_NodeInventory{
			NodeInventory: &storage.NodeInventory{
				NodeId:   uuid.NewDummy().String(),
				NodeName: n.nodeProvider.GetNodeName(),
				ScanTime: timestamp.TimestampNow(),
				Components: &storage.NodeInventory_Components{
					Namespace:       "rhcos:4.11",
					RhelContentSets: []string{"rhel-8-for-x86_64-appstream-rpms", "rhel-8-for-x86_64-baseos-rpms"},
					RhelComponents: []*storage.NodeInventory_Components_RHELComponent{
						{
							Id:        int64(1),
							Name:      "vim-minimal",
							Namespace: "rhel:8",
							Version:   "2:7.4.629-6.el8",
							Arch:      "x86_64",
							Module:    "",
							AddedBy:   "",
						},
					},
				},
				Notes: nil,
			}},
	}
	n.log.Infof("Generating Node Inventory")
	return msg, nil
}
