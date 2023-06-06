package retry

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// UnconfirmedMessageHandlerImpl informs the caller whether a resending should happen based on receiving ACK messsages.
// Assumption: Time to receive an ACK is generally much shorter than the interval between sending consequtive messages.
type UnconfirmedMessageHandlerImpl struct {
	// resendInterval is a multiply of the baseInterval and defines how much time should pass before the next retry
	resendInterval time.Duration
	// ctx is a context that can be used to stop this object
	ctx context.Context

	b         *backoff.ExponentialBackOff
	resendFun func() error
	gotAck    concurrency.Signal
}

// NewUnconfirmedMessageHandler returns a new running UnconfirmedMessageHandlerImpl.
// It can be stopped by canceling the context.
func NewUnconfirmedMessageHandler(ctx context.Context, b *backoff.ExponentialBackOff) *UnconfirmedMessageHandlerImpl {
	nsr := &UnconfirmedMessageHandlerImpl{
		ctx: ctx,
		b:   b,
		resendFun: func() error {
			return nil
		},
		gotAck: concurrency.NewSignal(),
	}
	nsr.b.Reset()
	return nsr
}

// SetOperation sets function that should produce a message
func (s *UnconfirmedMessageHandlerImpl) SetOperation(send func() error) {
	s.resendFun = func() error {
		go func() {
			_ = send()
		}()
		select {
		case <-s.gotAck.Done():
			log.Infof("Got ack!")
			s.gotAck.Reset()
			return nil
		case <-time.After(s.b.NextBackOff()):
			log.Infof("Seen no Ack in the last %s", s.b.NextBackOff().String())
			return errors.New("time is up")
		case <-s.ctx.Done():
			return s.ctx.Err()
		}
	}
}

// ExecOperation executes an operation that produces a new message expected to be [N]ACKed
func (s *UnconfirmedMessageHandlerImpl) ExecOperation() {
	if err := backoff.Retry(s.resendFun, s.b); err != nil {
		log.Errorf("Failed obtainig ACK for message %v", err)
	}
}

// HandleACK is called when ACK is received
func (s *UnconfirmedMessageHandlerImpl) HandleACK() {
	log.Infof("Received ACK")
	s.gotAck.Signal()
	s.b.Reset()
}

// HandleNACK is called when NACK is received
func (s *UnconfirmedMessageHandlerImpl) HandleNACK() {
	log.Debug("Received NACK. Message will be resent")
}
