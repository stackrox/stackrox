package main

import (
	"context"
	"time"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/compliance/collection/compliance"
	"github.com/stackrox/rox/compliance/collection/intervals"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
)

// LoadGeneratingNodeScanner is a scanner that generates fake scans with high frequecy of the node-inventory messages.
// Its main purpose is to generate load for load-testing of Sensor
type LoadGeneratingNodeScanner struct {
	nodeProvider       compliance.NodeNameProvider
	generationInterval time.Duration
	initialScanDelay   time.Duration
}

// IsActive returns true if the scanner is ready to be used
func (n *LoadGeneratingNodeScanner) IsActive() bool {
	return true
}

// Connect is a dummy as this scanner does not connect to anything
func (n *LoadGeneratingNodeScanner) Connect(_ string) {}

// GetIntervals returns an object with delay-intervals between scans
func (n *LoadGeneratingNodeScanner) GetIntervals() *intervals.NodeScanIntervals {
	return intervals.NewNodeScanInterval(n.generationInterval, 0.0, n.initialScanDelay)
}

// ScanNode generates a MsgFromCompliance with node scan
func (n *LoadGeneratingNodeScanner) ScanNode(_ context.Context) (*sensor.MsgFromCompliance, error) {
	msg := &sensor.MsgFromCompliance{
		Node: n.nodeProvider.GetNodeName(),
		Msg: &sensor.MsgFromCompliance_NodeInventory{
			NodeInventory: &storage.NodeInventory{
				NodeId:   "",
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
	log.Info("Generating Node Inventory")
	return msg, nil
}
