package handler

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/notifications/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/notifications"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	_ Handler = (*handlerImpl)(nil)

	notificationWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration),
		),
	)

	flushInterval = time.Minute

	log = logging.LoggerForModule()
)

// Handler is an interface to handle the notification stream.
type Handler interface {
	Start()
	Stop()
}

type handlerImpl struct {
	ds         datastore.DataStore
	stream     notifications.Stream
	stopSignal concurrency.Signal
}

func newHandler(ds datastore.DataStore, stream notifications.Stream) Handler {
	h := &handlerImpl{
		ds:         ds,
		stream:     stream,
		stopSignal: concurrency.NewSignal(),
	}
	return h
}

func (h *handlerImpl) watchForNotifications() {
	for {
		select {
		case notification := <-h.stream.Consume():
			if err := h.ds.AddNotification(notificationWriteCtx, notification); err != nil {
				log.Errorf("failed to store notification(message: %q): %v", notification.GetMessage(), err)
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
			if err := h.ds.Flush(notificationWriteCtx); err != nil {
				log.Error("failed to flush notifications")
			}
		case <-h.stopSignal.Done():
			return
		}
	}
}

func (h *handlerImpl) Start() {
	if h != nil {
		go h.watchForNotifications()
		go h.runDatastoreFlush()
	}
}

func (h *handlerImpl) Stop() {
	if h != nil {
		h.stopSignal.Signal()
	}
}
