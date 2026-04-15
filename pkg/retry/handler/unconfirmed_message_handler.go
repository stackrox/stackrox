package handler

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log                 = logging.LoggerForModule()
	defaultBaseInterval = 1 * time.Minute
)

// resourceState tracks the retry state for a single resource.
type resourceState struct {
	// retry counts the number of retries for this resource
	retry int
	// numUnackedSendings counts how many send attempts occurred since the last ack
	numUnackedSendings int
	// timer fires when a retry should be attempted
	timer *time.Timer
}

// UnconfirmedMessageHandlerImpl handles ACK/NACK messages for multiple resources.
// Each resource has independent retry state with linear backoff.
type UnconfirmedMessageHandlerImpl struct {
	handlerName  string
	baseInterval time.Duration

	resources map[string]*resourceState
	mu        sync.Mutex

	// retryNotifyCh wakes the retry dispatcher when at least one resource timer fires.
	// Timer callbacks do a non-blocking send after marking pendingRetries; with buffer size 1,
	// multiple firings coalesce into a single wake-up until the dispatcher drains pendingRetries.
	retryNotifyCh chan struct{}
	// pendingRetries is the per-resource retry set consumed by the dispatcher.
	// Access is guarded by mu: timer callbacks add entries, dispatcher snapshots and clears.
	pendingRetries set.StringSet
	// retryCommandCh carries concrete resourceIDs that callers should resend now.
	// It is intentionally unbuffered so retry production naturally back-pressures to consumption.
	retryCommandCh chan string
	// retryWorkerDone is a one-shot signal set when the dispatcher exits, so cleanup can
	// wait for the worker before closing retryCommandCh.
	retryWorkerDone concurrency.Signal
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
		handlerName:     handlerName,
		baseInterval:    baseInterval,
		resources:       make(map[string]*resourceState),
		retryNotifyCh:   make(chan struct{}, 1),
		pendingRetries:  set.NewStringSet(),
		retryCommandCh:  make(chan string),
		retryWorkerDone: concurrency.NewSignal(),
		ctx:             ctx,
		cleanupDone:     concurrency.NewStopper(),
	}

	go h.runRetryDispatcher()

	// Cleanup goroutine
	go func() {
		defer h.cleanupDone.Flow().ReportStopped()
		<-ctx.Done()
		concurrency.WithLock(&h.mu, func() {
			// Stop all timers to prevent more sends to channels.
			for _, state := range h.resources {
				if state.timer != nil {
					state.timer.Stop()
				}
			}
			// Close notification channel after timers are stopped.
			// Timers and cleanup both use h.mu to avoid close/send races.
			close(h.retryNotifyCh)
		})

		h.retryWorkerDone.Wait()
		// Close command channel after the worker exits.
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
	h.resetTimerNoLock(resourceID, state, h.calculateNextInterval(0))
}

// HandleACK is called when an ACK is received for a resource.
func (h *UnconfirmedMessageHandlerImpl) HandleACK(resourceID string) {
	var onAckCallback func(string)
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
			delete(h.resources, resourceID)
			h.pendingRetries.Remove(resourceID)
			log.Debugf("[%s] Received ACK for resource %s", h.handlerName, resourceID)
		} else {
			log.Debugf("[%s] Received ACK for unknown resource %s", h.handlerName, resourceID)
		}
		// Check callback inside the lock.
		if h.onACK != nil {
			onAckCallback = h.onACK
		}
	})

	// Invoke callback outside the lock to avoid potentially long-running operations inside the lock.
	if onAckCallback != nil {
		onAckCallback(resourceID)
	}
}

// HandleNACK is called when a NACK is received for a resource.
// It just logs - the existing timer will handle retry based on normal backoff.
func (h *UnconfirmedMessageHandlerImpl) HandleNACK(resourceID string) {
	// HandleNACK is currently a no-op and has the same behavior as not receiving any [N]ACK message.
	// This is intentional as we want to keep retrying until Central is able to process the message.
	// This will change in the future where NACK can be treated as a signal to slow down retries.
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

// resetTimerNoLock sets up the retry timer for a resource.
// Caller must hold h.mu.
func (h *UnconfirmedMessageHandlerImpl) resetTimerNoLock(resourceID string, state *resourceState, interval time.Duration) {
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
	h.resetTimerNoLock(resourceID, state, nextInterval)

	// Mark this resource as pending for retry and coalesce notifications.
	h.pendingRetries.Add(resourceID)
	select {
	case <-h.ctx.Done():
		return
	case h.retryNotifyCh <- struct{}{}:
	default:
		// A pending notification is already queued (or being processed); retries remain
		// tracked in pendingRetries and will be drained by the worker.
		log.Debugf("[%s] Retry notification queue full, coalescing notification for %s", h.handlerName, resourceID)
	}
}

func (h *UnconfirmedMessageHandlerImpl) runRetryDispatcher() {
	defer h.retryWorkerDone.Signal()
	for {
		select {
		case <-h.ctx.Done():
			return
		case _, ok := <-h.retryNotifyCh:
			if !ok {
				return
			}
		}

		// Drain all pending retries. Each takePendingRetries atomically swaps the set,
		// so new entries only appear from subsequent timer firings (which are spaced
		// by backoff intervals). The unbuffered retryCommandCh naturally back-pressures
		// here, ensuring the loop terminates once the current batch is delivered.
		for {
			pending := h.takePendingRetries()
			if len(pending) == 0 {
				break
			}

			for _, resourceID := range pending {
				if !h.isResourceActive(resourceID) {
					continue
				}
				select {
				case <-h.ctx.Done():
					return
				case h.retryCommandCh <- resourceID:
				}
			}
		}
	}
}

// isResourceActive returns true if the resource still has unacked sendings tracked.
// Used by the dispatcher to skip retries for resources that were ACKed between
// takePendingRetries and the actual send.
func (h *UnconfirmedMessageHandlerImpl) isResourceActive(resourceID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, exists := h.resources[resourceID]
	return exists
}

func (h *UnconfirmedMessageHandlerImpl) takePendingRetries() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.pendingRetries.Cardinality() == 0 {
		return nil
	}

	pending := h.pendingRetries.AsSlice()
	h.pendingRetries = set.NewStringSet()
	return pending
}

// calculateNextInterval returns the next retry interval with linear backoff.
func (h *UnconfirmedMessageHandlerImpl) calculateNextInterval(retry int) time.Duration {
	if h.baseInterval <= 0 {
		return defaultBaseInterval
	}

	next := time.Duration(retry+1) * h.baseInterval
	if next <= 0 {
		return defaultBaseInterval
	}
	return next
}
