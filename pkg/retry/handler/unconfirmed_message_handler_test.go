package handler

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/suite"
)

func TestUnconfirmedMessageHandler(t *testing.T) {
	suite.Run(t, new(UnconfirmedMessageHandlerTestSuite))
}

type UnconfirmedMessageHandlerTestSuite struct {
	suite.Suite
}

func (suite *UnconfirmedMessageHandlerTestSuite) TestWithRetryable() {
	cases := map[string]struct {
		baseDuration    time.Duration
		wait            time.Duration
		expectedRetries int
		sendAfter       []time.Duration
		ackAfter        []time.Duration
		nackAfter       []time.Duration
	}{
		"should retry once when 0 acks": {
			baseDuration:    time.Second,
			wait:            1100 * time.Millisecond,
			expectedRetries: 1,
			sendAfter:       []time.Duration{1 * time.Millisecond},
			ackAfter:        []time.Duration{},
			nackAfter:       []time.Duration{},
		},
		"should not retry when acks arrives immediately": {
			baseDuration:    time.Second,
			wait:            500 * time.Millisecond,
			expectedRetries: 0,
			sendAfter:       []time.Duration{1 * time.Millisecond},
			ackAfter:        []time.Duration{10 * time.Millisecond},
			nackAfter:       []time.Duration{},
		},
		"should retry 3 times within 6 seconds when base set to 1s": {
			baseDuration:    time.Second,
			wait:            6100 * time.Millisecond, // Retries after: 1s, 2s, 3s
			expectedRetries: 3,
			sendAfter:       []time.Duration{1 * time.Millisecond},
			ackAfter:        []time.Duration{},
			nackAfter:       []time.Duration{},
		},
		"should not reset retries if the first messsage is unacked and second message is sent": {
			baseDuration: time.Second,
			wait:         9100 * time.Millisecond,
			// Withouth reset, it retries 3 times in 9s: (send) 1s, 2s, (send), 3s, (test stop) 4s, 5s.
			// With reset, it would retry 4 times in 9s: (send) 1s, 2s, (send), 1s, 2s, 3s, (test stop).
			expectedRetries: 3,
			sendAfter:       []time.Duration{1 * time.Millisecond, 4100 * time.Millisecond},
			ackAfter:        []time.Duration{},
			nackAfter:       []time.Duration{},
		},
		"should retry normally when nack is received": {
			baseDuration:    time.Second,
			wait:            1100 * time.Millisecond,
			expectedRetries: 1,
			sendAfter:       []time.Duration{1 * time.Millisecond},
			ackAfter:        []time.Duration{},
			nackAfter:       []time.Duration{3 * time.Millisecond},
		},
	}

	for name, cc := range cases {
		suite.Run(name, func() {
			counterMux := &sync.Mutex{}
			counter := 0

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			umh := NewUnconfirmedMessageHandler(ctx, cc.baseDuration)
			// sending loop
			for _, tt := range cc.sendAfter {
				go func(tt time.Duration) {
					<-time.After(tt)
					suite.T().Logf("Sending test message")
					umh.ObserveSending()
				}(tt)
			}
			// acking loop
			for _, tt := range cc.ackAfter {
				go func(tt time.Duration) {
					<-time.After(tt)
					umh.HandleACK()
				}(tt)
			}
			// nacking loop
			for _, tt := range cc.ackAfter {
				go func(tt time.Duration) {
					<-time.After(tt)
					umh.HandleNACK()
				}(tt)
			}
			// retry-counting loop
			go func() {
				for range umh.RetryCommand() {
					counterMux.Lock()
					counter++
					counterMux.Unlock()
				}
			}()
			<-time.After(cc.wait)

			counterMux.Lock()
			defer counterMux.Unlock()
			suite.Equal(cc.expectedRetries, counter)
		})
	}
}
