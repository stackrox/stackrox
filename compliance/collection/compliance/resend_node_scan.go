package compliance

import (
	"context"
	"fmt"
	"time"
)

var _ unconfirmedMessageHandler = (*NodeScanResend)(nil)

var defaultBaseInterval = 1 * time.Minute

// NodeScanResend handles resending node scans based on ACK messages from Central
// Assumption: Time to receive an ACK is generally much shorter than ROX_NODE_SCANNING_INTERVAL.
type NodeScanResend struct {
	// baseInterval defines the delay after which we resend a message
	baseInterval time.Duration
	// resendInterval is a multiply of the baseInterval and defines how much time should pass before the next retry
	resendInterval time.Duration
	// ticker controls the delay after which we resend a message
	ticker *time.Ticker
	// ch will produce a message when the message should be resent
	ch chan struct{}
	// retry counts the number of retries for a given message
	retry int
	// ctx is a context that can be used to stop this object
	ctx context.Context
}

// NewNodeScanResend returns a new running NodeScanResend.
// It can be stopped by canceling the context.
func NewNodeScanResend(ctx context.Context, resendInterval time.Duration) *NodeScanResend {
	nsr := &NodeScanResend{
		baseInterval:   resendInterval,
		resendInterval: resendInterval,
		ticker:         time.NewTicker(resendInterval),
		ch:             make(chan struct{}),
		retry:          0,
		ctx:            ctx,
	}
	nsr.ticker.Stop()
	nsr.run()
	return nsr
}

// RetryCommand returns a channel that will produce a message when sending should be retried
func (s *NodeScanResend) RetryCommand() <-chan struct{} {
	return s.ch
}

// run starts the ticker, so that resending can happen after defined time without ACK passes
func (s *NodeScanResend) run() {
	go func() {
		defer close(s.ch)
		for {
			select {
			case <-s.ticker.C:
				s.retryLater()
				log.Infof("Resending node scan, retry %d (next retry in %s)", s.retry, s.resendInterval)
				s.ch <- struct{}{}
			case <-s.ctx.Done():
				return
			}
		}
	}()
}

func (s *NodeScanResend) retryLater() {
	s.retry++
	nextIn := s.retry * int(s.baseInterval.Seconds())
	next, err := time.ParseDuration(fmt.Sprintf("%ds", nextIn))
	if err != nil {
		next = defaultBaseInterval
	}
	s.resendInterval = next
	s.ticker.Stop()
	s.ticker.Reset(s.resendInterval)
}

// ObserveSending should be called when a new node-inventory is sent
func (s *NodeScanResend) ObserveSending() {
	log.Infof("Observing node scan being sent. Waiting for an ACK for %s", s.baseInterval.String())
	s.ticker.Stop()
	s.retry = 0
	s.ticker.Reset(s.baseInterval)
}

// ObserveConfirmation should be called when an ACK for node-inventory is received
func (s *NodeScanResend) ObserveConfirmation() {
	log.Info("Node Scan has been acknowledged")
	s.ticker.Stop()
	s.retry = 0
}
