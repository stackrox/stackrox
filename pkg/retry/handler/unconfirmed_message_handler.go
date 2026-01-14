package handler

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log                 = logging.LoggerForModule()
	defaultBaseInterval = 1 * time.Minute
)

// resourceState tracks the retry state for a single resource.
type resourceState struct {
	// retry counts the number of retries for this resource
	retry int32
	// numUnackedSendings counts how many send attempts occurred since the last ack
	numUnackedSendings int32
	// timer fires when a retry should be attempted
	timer *time.Timer
}

// UnconfirmedMessageHandlerImpl handles ACK/NACK messages for multiple resources.
// Each resource has independent retry state with exponential backoff.
type UnconfirmedMessageHandlerImpl struct {
	handlerName  string
	baseInterval time.Duration

	resources map[string]*resourceState
	mu        sync.Mutex

	// retryCommandCh emits resourceID when a retry should be attempted
	retryCommandCh chan string
	// onACK is called when an ACK is received for a resource (optional)
	onACK func(resourceID string)
	ctx   context.Context

	// cleanupDone signals when cleanup is complete
	cleanupDone concurrency.Stopper
}

// NewUnconfirmedMessageHandler creates a new handler for per-resource ACK/NACK tracking.
// It can be stopped by canceling the context.
func NewUnconfirmedMessageHandler(ctx context.Context, handlerName string, baseInterval time.Duration) *UnconfirmedMessageHandlerImpl {
	h := &UnconfirmedMessageHandlerImpl{
		handlerName:    handlerName,
		baseInterval:   baseInterval,
		resources:      make(map[string]*resourceState),
		retryCommandCh: make(chan string),
		ctx:            ctx,
		cleanupDone:    concurrency.NewStopper(),
	}

	// Cleanup goroutine
	go func() {
		defer h.cleanupDone.Flow().ReportStopped()
		<-ctx.Done()
		h.mu.Lock()
		defer h.mu.Unlock()
		// Stop all timers to prevent more sends to channels
		for _, state := range h.resources {
			if state.timer != nil {
				state.timer.Stop()
			}
		}
		// Close channel after timers are stopped
		close(h.retryCommandCh)
	}()

	return h
}

// Stopped returns a signal that is triggered when cleanup is complete.
// Callers can wait on this to ensure the handler has fully shut down.
func (h *UnconfirmedMessageHandlerImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return h.cleanupDone.Client().Stopped()
}

// RetryCommand returns a channel that emits resourceIDs when they should be retried.
func (h *UnconfirmedMessageHandlerImpl) RetryCommand() <-chan string {
	return h.retryCommandCh
}

// OnACK registers a callback to be invoked when an ACK is received for a resource.
// The callback is invoked outside the lock, so it is safe to perform blocking operations.
// Only one callback can be registered; subsequent calls replace the previous callback.
func (h *UnconfirmedMessageHandlerImpl) OnACK(callback func(resourceID string)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onACK = callback
}

// ObserveSending should be called when a message is sent for a resource.
func (h *UnconfirmedMessageHandlerImpl) ObserveSending(resourceID string) {
	// Check if handler is stopped before any operations
	if h.ctx.Err() != nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	state := h.getOrCreateStateNoLock(resourceID)
	state.numUnackedSendings++

	log.Debugf("[%s] Observing send for resource %s (unacked: %d)",
		h.handlerName, resourceID, state.numUnackedSendings)

	if state.numUnackedSendings > 1 {
		// Previous message not acked - don't reset timer
		return
	}

	// First unacked message - start/reset timer
	state.retry = 0
	h.resetTimer(resourceID, state, h.baseInterval)
}

// HandleACK is called when an ACK is received for a resource.
func (h *UnconfirmedMessageHandlerImpl) HandleACK(resourceID string) {
	concurrency.WithLock(&h.mu, func() {
		// Check if handler is stopped before any operations
		if h.ctx.Err() != nil {
			return
		}

		state, exists := h.resources[resourceID]
		if exists {
			if state.timer != nil {
				state.timer.Stop()
			}
			state.retry = 0
			state.numUnackedSendings = 0
			log.Debugf("[%s] Received ACK for resource %s", h.handlerName, resourceID)
		} else {
			log.Debugf("[%s] Received ACK for unknown resource %s", h.handlerName, resourceID)
		}
	})

	// Invoke callback outside lock
	if h.onACK != nil {
		h.onACK(resourceID)
	}
}

// HandleNACK is called when a NACK is received for a resource.
// It just logs - the existing timer will handle retry based on normal backoff.
func (h *UnconfirmedMessageHandlerImpl) HandleNACK(resourceID string) {
	log.Debugf("[%s] Received NACK for resource %s. Message will be resent.", h.handlerName, resourceID)
}

// getOrCreateStateNoLock returns the state for a resource, creating it if needed.
func (h *UnconfirmedMessageHandlerImpl) getOrCreateStateNoLock(resourceID string) *resourceState {
	state, exists := h.resources[resourceID]
	if !exists {
		state = &resourceState{}
		h.resources[resourceID] = state
	}
	return state
}

// resetTimer sets up the retry timer for a resource.
func (h *UnconfirmedMessageHandlerImpl) resetTimer(resourceID string, state *resourceState, interval time.Duration) {
	if state.timer != nil {
		state.timer.Stop()
	}

	state.timer = time.AfterFunc(interval, func() {
		h.onTimerFired(resourceID)
	})
}

// onTimerFired is called when a resource's retry timer fires.
func (h *UnconfirmedMessageHandlerImpl) onTimerFired(resourceID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Check context
	if h.ctx.Err() != nil {
		return
	}
	state, exists := h.resources[resourceID]
	if !exists || state.numUnackedSendings == 0 {
		return
	}

	state.retry++
	nextInterval := h.calculateNextInterval(state.retry)

	log.Infof("[%s] Resource %s has %d unacked messages, suggesting retry %d (next in %s)",
		h.handlerName, resourceID, state.numUnackedSendings, state.retry, nextInterval)

	// Schedule next retry
	h.resetTimer(resourceID, state, nextInterval)

	// Signal retry (non-blocking); if the channel is full we log and drop the signal.
	select {
	case <-h.ctx.Done():
		return
	case h.retryCommandCh <- resourceID:
	default:
		log.Warnf("[%s] Retry channel full, dropping retry signal for %s", h.handlerName, resourceID)
	}
}

// calculateNextInterval returns the next retry interval with exponential backoff.
func (h *UnconfirmedMessageHandlerImpl) calculateNextInterval(retry int32) time.Duration {
	if h.baseInterval <= 0 {
		return defaultBaseInterval
	}

	next := time.Duration(retry+1) * h.baseInterval
	if next <= 0 {
		return defaultBaseInterval
	}
	return next
}
