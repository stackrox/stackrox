package handler

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/stackrox/rox/pkg/logging"
)

var (
	log                 = logging.LoggerForModule()
	defaultBaseInterval = 1 * time.Minute
)

// UnconfirmedMessageHandlerImpl informs the caller whether a resending should happen based on receiving ACK messsages.
// Assumption: Time to receive an ACK is generally much shorter than the interval between sending consequtive messages.
type UnconfirmedMessageHandlerImpl struct {
	// baseInterval defines the delay after which we resend a message
	baseInterval time.Duration
	// resendInterval is a multiply of the baseInterval and defines how much time should pass before the next retry
	resendInterval time.Duration
	// ticker controls the delay after which we resend a message
	ticker *time.Ticker
	// ch will produce a message when the message should be resent
	ch chan struct{}
	// retry counts the number of retries for a given message
	retry atomic.Int32
	// numUnackedSendings counts how many sendings occured (not retries) since the last ack
	numUnackedSendings atomic.Int32
	// ctx is a context that can be used to stop this object
	ctx context.Context
}

// NewUnconfirmedMessageHandler returns a new running UnconfirmedMessageHandlerImpl.
// It can be stopped by canceling the context.
func NewUnconfirmedMessageHandler(ctx context.Context, resendInterval time.Duration) *UnconfirmedMessageHandlerImpl {
	nsr := &UnconfirmedMessageHandlerImpl{
		baseInterval:   resendInterval,
		resendInterval: resendInterval,
		ticker:         time.NewTicker(resendInterval),
		ch:             make(chan struct{}),
		ctx:            ctx,
	}
	nsr.ticker.Stop()
	nsr.run()
	return nsr
}

// RetryCommand returns a channel that will produce a message when sending should be retried
func (s *UnconfirmedMessageHandlerImpl) RetryCommand() <-chan struct{} {
	return s.ch
}

// run starts the ticker, so that resending can happen after defined time without ACK passes
func (s *UnconfirmedMessageHandlerImpl) run() {
	go func() {
		defer close(s.ch)
		for {
			select {
			case <-s.ticker.C:
				s.retryLater()
				log.Infof("Suggesting to resend, retry %d (next retry in %s)", s.retry.Load(), s.resendInterval)
				s.ch <- struct{}{}
			case <-s.ctx.Done():
				return
			}
		}
	}()
}

func (s *UnconfirmedMessageHandlerImpl) retryLater() {
	s.retry.Add(1)
	nextIn := (s.retry.Load() + 1) * int32(s.baseInterval.Seconds())
	next, err := time.ParseDuration(fmt.Sprintf("%ds", nextIn))
	if err != nil {
		next = defaultBaseInterval
	}
	s.resendInterval = next
	s.ticker.Stop()
	s.ticker.Reset(s.resendInterval)
}

// ObserveSending should be called when a new message is sent and it is expected to be [N]ACKed
func (s *UnconfirmedMessageHandlerImpl) ObserveSending() {
	s.numUnackedSendings.Add(1)
	log.Debugf("Observing message being sent. Waiting for an ACK for %s", s.baseInterval.String())
	if s.numUnackedSendings.Load() > 1 {
		// Not resetting the ticker to the the baseInterval, because previous message was not acked at all
		return
	}
	s.ticker.Stop()
	s.retry.Store(0)
	s.ticker.Reset(s.baseInterval)
}

func (s *UnconfirmedMessageHandlerImpl) observeConfirmation() {
	log.Debug("Message has been acknowledged")
	s.ticker.Stop()
	s.retry.Store(0)
	s.numUnackedSendings.Store(0)
}

// HandleACK is called when ACK is received
func (s *UnconfirmedMessageHandlerImpl) HandleACK() {
	log.Debug("Received ACK")
	s.observeConfirmation()
}

// HandleNACK is called when NACK is received
func (s *UnconfirmedMessageHandlerImpl) HandleNACK() {
	log.Debug("Received NACK. Message will be resent")
}
