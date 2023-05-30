package compliance

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
)

// NodeScanResend handles ACK/NACK messages from Sensor
type NodeScanResend struct {
	onResend       func(*sensor.MsgFromCompliance)
	resendInterval time.Duration
	ticker         *time.Ticker
	inventory      *sensor.MsgFromCompliance
}

// NewNodeScanResend returns new NodeScanResend
func NewNodeScanResend(resendInterval time.Duration) *NodeScanResend {
	nsr := &NodeScanResend{
		onResend:       func(*sensor.MsgFromCompliance) {},
		resendInterval: resendInterval,
		ticker:         time.NewTicker(resendInterval),
		inventory:      nil,
	}
	nsr.ticker.Stop()
	return nsr
}

// SetOnResend defines the function that should be called when resending is necessary
func (s *NodeScanResend) SetOnResend(fn func(*sensor.MsgFromCompliance)) {
	s.onResend = fn
}

// Run starts the ticker, so that resending can happen after defined time without ACK passes
func (s *NodeScanResend) Run(ctx context.Context) {
	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.onResend(s.inventory)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// RegisterSending should be called when a new node-inventory is sent
func (s *NodeScanResend) RegisterSending(msg *sensor.MsgFromCompliance) {
	s.inventory = msg
	s.ticker.Reset(s.resendInterval)
}

// RegisterACK should be called when an ACK for node-inventory is received
func (s *NodeScanResend) RegisterACK() {
	s.ticker.Stop()
	s.inventory = nil
}
