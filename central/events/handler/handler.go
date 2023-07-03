package handler

import (
	"context"

	"github.com/stackrox/rox/central/events/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/events"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	_ events.Handler = (*handlerImpl)(nil)

	eventWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))

	log = logging.LoggerForModule()
)

type handlerImpl struct {
	ds datastore.DataStore

	eventUpdateChan chan *storage.Event

	stopSignal concurrency.Signal
}

func newHandler(ds datastore.DataStore) events.Handler {
	h := &handlerImpl{
		ds:              ds,
		eventUpdateChan: make(chan *storage.Event, 10), // Let's not do more than 10 for now.
		stopSignal:      concurrency.NewSignal(),
	}
	go h.watchForEvents()
	return h
}

func (h *handlerImpl) AddEventAsync(event *storage.Event) {
	select {
	case h.eventUpdateChan <- event:
		return
	case <-h.stopSignal.Done():
		return
	}
}

func (h *handlerImpl) watchForEvents() {
	for {
		select {
		case event := <-h.eventUpdateChan:
			if err := h.ds.AddEvent(eventWriteCtx, event); err != nil {
				log.Errorf("Failed to add event(message: %q): %v", event.GetMsg(), err)
			}
		case <-h.stopSignal.Done():
			return
		}
	}
}

func (h *handlerImpl) Stop() {
	h.stopSignal.Signal()
}
