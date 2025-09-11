package vm_relay

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
)

// VsockRelay provides functionality to relay virtual machine index reports from vsock to sensor
type VsockRelay interface {
	// Run starts the vsock relay loop, which waits for data and immediately sends it
	// In the real implementation, this would wait for vsock data. For now, it sends fake data every 5 seconds.
	Run(ctx context.Context)
}

// vsockRelayImpl implements VsockRelay
type vsockRelayImpl struct {
	vsockServer VsockServer
	handler     Handler
	sender      ReportSender
}

// NewVsockRelay creates a new vsock relay
func NewVsockRelay(client sensor.VirtualMachineIndexReportServiceClient) VsockRelay {
	return &vsockRelayImpl{
		vsockServer: NewVsockServer(),
		handler:     NewIndexReportHandler(),
		sender:      NewReportSender(client),
	}
}

// Run starts the vsock relay loop
// This waits for index reports from the server and immediately relays them to the sensor
func (v *vsockRelayImpl) Run(ctx context.Context) {
	log.Infof("Starting vsock relay")

	// Create a channel for results from the server
	resultChan := make(chan interface{}, 10) // Buffered to avoid blocking

	// Start the vsock server in a goroutine
	go v.vsockServer.Run(ctx, v.handler, resultChan)

	// Wait for results and relay them
	for {
		select {
		case <-ctx.Done():
			log.Infof("Vsock relay stopping")
			return
		case result := <-resultChan:
			// Type assert to IndexReport
			report, ok := result.(*v1.IndexReport)
			if !ok {
				log.Errorf("Received unexpected result type: %T", result)
				continue
			}

			// Relay the report to the sensor
			if err := v.sender.SendIndexReport(ctx, report); err != nil {
				log.Errorf("Error relaying VM index report: %v", err)
			}
		}
	}
}
