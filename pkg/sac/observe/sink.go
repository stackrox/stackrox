package observe

import (
	"context"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/sync"
)

// AuthzTraceSink is a sink for authz traces which uses channels as the way of communication.
type AuthzTraceSink interface {
	IsActive() bool
	PublishAuthzTrace(trace *v1.AuthorizationTraceResponse)
	Subscribe(ctx context.Context) <-chan *v1.AuthorizationTraceResponse
}

type authzTraceSinkImpl struct {
	publishChannels []channelWithContext
	lock            sync.RWMutex
}

type channelWithContext struct {
	channel chan<- *v1.AuthorizationTraceResponse
	ctx     context.Context
}

// IsActive returns whether sink is active or not.
func (s *authzTraceSinkImpl) IsActive() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return len(s.publishChannels) != 0
}

// PublishAuthzTrace publishes trace to all channels present in sink.
// It blocks until every channel reads trace or signals it is not interested any longer by
// cancelling the associated context.
// In case there are no consumers, it's safe to call as in this case trace is just ignored.
func (s *authzTraceSinkImpl) PublishAuthzTrace(trace *v1.AuthorizationTraceResponse) {
	// Copy publishChannels to avoid sending message inside critical section.
	channels := s.fetchPublishChannels()
	for _, c := range channels {
		select {
		case <-c.ctx.Done():
		case c.channel <- trace:
		}
	}
}

// Subscribe creates new channel in the sink and associates it with passed context.
// We will send traces to this channel until we will receive <-ctx.Done().
func (s *authzTraceSinkImpl) Subscribe(ctx context.Context) <-chan *v1.AuthorizationTraceResponse {
	// Buffer size 1 improves probability that single PublishAuthzTrace() will be executed without waiting for receiver.
	// If you ever decide to change this capacity, especially by reducing it to 0 (unbuffered), you MUST make sure that
	// the assumptions laid out in `sink_test.go:TestPublishAuthzTrace` still hold,
	// otherwise that test might deadlock.
	publishC := make(chan *v1.AuthorizationTraceResponse, 1)

	s.lock.Lock()
	defer s.lock.Unlock()

	s.publishChannels = append(s.publishChannels, channelWithContext{channel: publishC, ctx: ctx})
	return publishC
}

// fetchPublishChannels purges done channels from the stored publish channels
// and returns a copy which can be manipulated without synchronization. The
// caller must handle done channels in the returned slice.
func (s *authzTraceSinkImpl) fetchPublishChannels() []channelWithContext {
	s.lock.Lock()
	defer s.lock.Unlock()

	purged := make([]channelWithContext, 0, len(s.publishChannels))
	for _, c := range s.publishChannels {
		// Don't close done channels to avoid writing to a closed channel
		// and thus panic().
		if c.ctx.Err() == nil {
			purged = append(purged, c)
		}
	}

	old := s.publishChannels
	s.publishChannels = purged
	return old
}

// NewAuthzTraceSink returns new AuthzTraceSink.
func NewAuthzTraceSink() AuthzTraceSink {
	return &authzTraceSinkImpl{}
}
