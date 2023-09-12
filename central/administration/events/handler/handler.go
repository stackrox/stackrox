package handler

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/administration/events/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	_ Handler = (*handlerImpl)(nil)

	flushInterval = time.Minute

	log = logging.LoggerForModule()
)

// Handler is an interface to handle the administration event stream.
type Handler interface {
	Start()
	Stop()
}

type handlerImpl struct {
	ds            datastore.DataStore
	eventWriteCtx context.Context
	stream        events.Stream
	stopSignal    concurrency.Signal
}

func newHandler(ds datastore.DataStore, stream events.Stream) Handler {
	h := &handlerImpl{
		ds: ds,
		eventWriteCtx: sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.Administration),
			),
		),
		stream:     stream,
		stopSignal: concurrency.NewSignal(),
	}
	return h
}

func (h *handlerImpl) watchForEvents() {
	for {
		select {
		case event := <-h.stream.Consume():
			if err := retry.WithRetry(
				func() error {
					return h.ds.AddEvent(h.eventWriteCtx, event)
				},
				retry.BetweenAttempts(
					func(_ int) {
						concurrency.WaitWithTimeout(h.eventWriteCtx, 10*time.Second)
					},
				),
				retry.OnlyRetryableErrors(),
				retry.Tries(10),
			); err != nil {
				log.Errorf("failed to store administration event(message: %q): %v", event.GetMessage(), err)
			}
		case <-h.stopSignal.Done():
			return
		}
	}
}

func (h *handlerImpl) runDatastoreFlush() {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := h.ds.Flush(h.eventWriteCtx); err != nil {
				log.Error(err)
			}
		case <-h.stopSignal.Done():
			return
		}
	}
}

func (h *handlerImpl) Start() {
	if h != nil {
		go h.watchForEvents()
		go h.runDatastoreFlush()
	}
}

func (h *handlerImpl) Stop() {
	if h != nil {
		h.stopSignal.Signal()
	}
}
