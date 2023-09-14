package writer

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/administration/events/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	_ Writer = (*writerImpl)(nil)

	log           = logging.LoggerForModule()
	maxWriterSize = 1000
	eventSAC      = sac.ForResource(resources.Administration)
)

type writerImpl struct {
	mutex sync.Mutex

	buffer map[string]*storage.AdministrationEvent
	store  store.Store
}

func (c *writerImpl) readNoLock(id string) (*storage.AdministrationEvent, bool) {
	event, found := c.buffer[id]
	return event, found
}

func (c *writerImpl) writeNoLock(event *storage.AdministrationEvent) {
	c.buffer[event.GetId()] = event
}

func (c *writerImpl) resetNoLock() {
	c.buffer = make(map[string]*storage.AdministrationEvent)
}

func (c *writerImpl) Upsert(ctx context.Context, event *events.AdministrationEvent) error {
	if event == nil {
		return errox.InvalidArgs.CausedBy("empty event")
	}

	if err := sac.VerifyAuthzOK(eventSAC.WriteAllowed(ctx)); err != nil {
		return err
	}

	var isBufferFull bool
	concurrency.WithLock(&c.mutex, func() {
		isBufferFull = len(c.buffer) >= maxWriterSize
	})

	// If the buffer is full, first flush and clear the buffer. This ensures concurrent callers won't receive
	// unexpected errors and additional logic on their side to flush and retry.
	if isBufferFull {
		if err := c.Flush(ctx); err != nil {
			return errors.Wrap(err, "failed to flush events when buffer was full")
		}
	}

	enrichedEvent := event.ToStorageEvent()
	id := enrichedEvent.GetId()

	var baseEvent *storage.AdministrationEvent

	c.mutex.Lock()
	defer c.mutex.Unlock()

	eventInBuffer, foundInBuffer := c.readNoLock(id)
	// If an event already exists in the buffer it is the most recent.
	// We use it as a base to merge with the new event.
	if foundInBuffer {
		baseEvent = eventInBuffer
	} else {
		// If no event is in the buffer, we try to fetch an event
		// from the database. If foundInDB, we use it as the base for the merge.
		eventInDB, foundInDB, err := c.store.Get(ctx, id)
		if err != nil {
			return errors.Wrap(err, "failed to query for existing record")
		}
		if foundInDB {
			baseEvent = eventInDB
		}
	}

	// Merge events to up the occurrence and update the time stamps.
	if baseEvent != nil {
		mergeEvents(baseEvent, enrichedEvent)
	}

	c.writeNoLock(enrichedEvent)
	return nil
}

func (c *writerImpl) Flush(ctx context.Context) error {
	if err := sac.VerifyAuthzOK(eventSAC.WriteAllowed(ctx)); err != nil {
		return err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	eventChunk := make([]*storage.AdministrationEvent, 0, len(c.buffer))
	for _, event := range c.buffer {
		eventChunk = append(eventChunk, event)
	}
	err := c.store.UpsertMany(ctx, eventChunk)
	if err != nil {
		return errors.Wrap(err, "failed to upsert event chunk")
	}
	// Reset buffer only if upsert was successful.
	c.resetNoLock()
	return nil
}

// Modifies `updated` in place with the values of the merged event.
func mergeEvents(base *storage.AdministrationEvent, updated *storage.AdministrationEvent) {
	if base == nil {
		return
	}
	if updated == nil {
		updated = base //nolint:staticcheck
		return
	}

	// Set CreatedAt timestamp to the earliest timestamp.
	if base.GetCreatedAt().GetSeconds() < updated.GetCreatedAt().GetSeconds() {
		updated.CreatedAt = base.GetCreatedAt()
	}
	// Set LastOccured timestamp to the latest timestamp.
	if base.GetLastOccurredAt().GetSeconds() > updated.GetLastOccurredAt().GetSeconds() {
		updated.LastOccurredAt = base.GetLastOccurredAt()
	}
	updated.NumOccurrences = base.GetNumOccurrences() + 1
}
