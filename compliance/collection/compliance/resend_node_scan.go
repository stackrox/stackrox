package compliance

import (
	"context"
	"fmt"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
)

// NodeScanResend handles resending node scans based on ACK messages from Central
// Assumption: Time to receive an ACK is generally much shorter than ROX_NODE_SCANNING_INTERVAL.
type NodeScanResend struct {
	// baseInterval defines the delay after which we resend a message
	baseInterval time.Duration
	// resendInterval is a multiply of the baseInterval and defines how much time should pass before the next retry
	resendInterval time.Duration
	// ticker controls the delay after which we resend a message
	ticker *time.Ticker
	// msg holds message in memory until a tick happens
	msg *sensor.MsgFromCompliance
	// ch will contain a message if msg is not nil and tick happens
	ch chan *sensor.MsgFromCompliance
	// retry counts the number of retries for a given message
	retry int
}

// NewNodeScanResend returns new NodeScanResend
func NewNodeScanResend(resendInterval time.Duration) *NodeScanResend {
	nsr := &NodeScanResend{
		baseInterval:   resendInterval,
		resendInterval: resendInterval,
		ticker:         time.NewTicker(resendInterval),
		msg:            nil,
		ch:             make(chan *sensor.MsgFromCompliance),
		retry:          0,
	}
	nsr.ticker.Stop()
	return nsr
}

// ResendChannel returns a channel with messages that should be resent
func (s *NodeScanResend) ResendChannel() <-chan *sensor.MsgFromCompliance {
	return s.ch
}

// Run starts the ticker, so that resending can happen after defined time without ACK passes
func (s *NodeScanResend) Run(ctx context.Context) {
	go func() {
		defer close(s.ch)
		for {
			select {
			case <-s.ticker.C:
				s.retry++
				s.incrementTicker()
				log.Infof("Resending node scan, retry %d (next retry in %s)", s.retry, s.resendInterval)
				if s.msg != nil {
					s.ch <- s.msg
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *NodeScanResend) incrementTicker() {
	nextIn := s.retry * int(s.baseInterval.Seconds())
	next, err := time.ParseDuration(fmt.Sprintf("%ds", nextIn))
	if err != nil {
		next = 5 * time.Second
	}
	s.resendInterval = next
	s.ticker.Stop()
	s.ticker.Reset(s.resendInterval)
}

// RegisterScan should be called when a new node-msg is sent
func (s *NodeScanResend) RegisterScan(msg *sensor.MsgFromCompliance) {
	log.Infof("Registering node scan. Waiting for an ACK for %s", s.baseInterval.String())
	s.retry = 0
	s.msg = msg
	s.ticker.Stop()
	s.ticker.Reset(s.baseInterval)
}

// RegisterACK should be called when an ACK for node-msg is received
func (s *NodeScanResend) RegisterACK() {
	log.Info("Node Scan has been acknowledged")
	s.ticker.Stop()
	s.msg = nil
	s.retry = 0
}
