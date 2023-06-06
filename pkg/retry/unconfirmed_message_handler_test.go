package retry

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/stretchr/testify/suite"
)

func TestUnconfirmedMessageHandler(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(UnconfirmedMessageHandlerTestSuite))
}

type UnconfirmedMessageHandlerTestSuite struct {
	suite.Suite
}

func (suite *UnconfirmedMessageHandlerTestSuite) TestWithRetryable() {
	cases := map[string]struct {
		wait                    time.Duration
		expectedSendingAttempts int32
		sendAfter               []time.Duration
		ackAfter                []time.Duration
		nackAfter               []time.Duration
	}{
		"should attempt once within a second when 0 acks": {
			wait:                    1100 * time.Millisecond, // 100ms flake-buffer
			expectedSendingAttempts: 1,
			sendAfter:               []time.Duration{1 * time.Millisecond},
			ackAfter:                []time.Duration{},
			nackAfter:               []time.Duration{},
		},
		"should attempt twice within two seconds when 0 acks": {
			wait:                    2100 * time.Millisecond, // 100ms flake-buffer
			expectedSendingAttempts: 2,
			sendAfter:               []time.Duration{1 * time.Millisecond},
			ackAfter:                []time.Duration{},
			nackAfter:               []time.Duration{},
		},
		"should attempt only once when ack arrives immediately": {
			wait:                    500 * time.Millisecond,
			expectedSendingAttempts: 1,
			sendAfter:               []time.Duration{1 * time.Millisecond},
			ackAfter:                []time.Duration{10 * time.Millisecond},
			nackAfter:               []time.Duration{},
		},
		"should attempt 3 times within 5 seconds": {
			wait:                    3100 * time.Millisecond,
			expectedSendingAttempts: 3,
			sendAfter:               []time.Duration{1 * time.Millisecond},
			ackAfter:                []time.Duration{},
			nackAfter:               []time.Duration{},
		},
		"should retry normally when nack is received": {
			wait:                    1100 * time.Millisecond,
			expectedSendingAttempts: 1,
			sendAfter:               []time.Duration{1 * time.Millisecond},
			ackAfter:                []time.Duration{},
			nackAfter:               []time.Duration{3 * time.Millisecond},
		},
	}

	for name, cc := range cases {
		suite.Run(name, func() {
			var numSent int32

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			back := backoff.NewExponentialBackOff()
			back.InitialInterval = time.Second
			back.RandomizationFactor = 0.0
			back.Multiplier = 1.0
			back.MaxElapsedTime = 30 * time.Minute

			umh := NewUnconfirmedMessageHandler(ctx, back)
			umh.SetOperation(func() error {
				atomic.AddInt32(&numSent, 1)
				suite.T().Logf("Sent message. Counter: %d", numSent)
				return nil
			})
			// sending loop
			go umh.ExecOperation()

			// acking loop
			for _, tt := range cc.ackAfter {
				go func(tt time.Duration) {
					<-time.After(tt)
					suite.T().Logf("Acking-test")
					umh.HandleACK()
				}(tt)
			}
			// nacking loop
			for _, tt := range cc.ackAfter {
				go func(tt time.Duration) {
					<-time.After(tt)
					suite.T().Logf("Nacking-test")
					umh.HandleNACK()
				}(tt)
			}
			<-time.After(cc.wait)

			suite.Equal(cc.expectedSendingAttempts, numSent)
		})
	}
}
