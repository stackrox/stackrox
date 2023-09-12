package writer

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/administration/events/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retry"
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
	if c == nil {
		return nil
	}
	if event == nil {
		return errox.InvalidArgs.CausedBy("empty event")
	}

	enrichedEvent := event.ToStorageEvent()
	id := enrichedEvent.GetId()

	if ok, err := eventSAC.WriteAllowed(ctx); err != nil || !ok {
		if err != nil {
			log.Errorf("failed to verify scope access control: ", err)
		}
		return errors.Wrapf(sac.ErrResourceAccessDenied, "administration event %q", enrichedEvent.GetId())
	}

	var baseEvent *storage.AdministrationEvent

	c.mutex.Lock()
	defer c.mutex.Unlock()

	eventInBuffer, found := c.readNoLock(id)
	// If an event already exists in the buffer it is the most recent.
	// We use it as a base to merge with the new event.
	if found {
		baseEvent = eventInBuffer
	} else {
		// Short circuit if buffer reached capacity and we're about to add another event.
		if len(c.buffer) >= maxWriterSize {
			return retry.MakeRetryable(errWriteBufferExhausted)
		}

		// If no event is in the buffer, we try to fetch an event
		// from the database. If found, we use it as the base for the merge.
		eventInDB, found, err := c.store.Get(ctx, id)
		if err != nil {
			return errors.Wrap(err, "failed to query for existing record")
		}
		if found {
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
	if c == nil {
		return nil
	}

	if ok, err := eventSAC.WriteAllowed(ctx); err != nil || !ok {
		if err != nil {
			log.Errorf("failed to verify scope access control: ", err)
		}
		return errors.Wrap(sac.ErrResourceAccessDenied, "administration events flush")
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

// Modifies `new` in place with the values of the merged event.
func mergeEvents(base *storage.AdministrationEvent, new *storage.AdministrationEvent) {
	if base == nil {
		return
	}
	if new == nil {
		new = base
		return
	}

	// Set CreatedAt timestamp to the earliest timestamp.
	if base.GetCreatedAt().GetSeconds() < new.GetCreatedAt().GetSeconds() {
		new.CreatedAt = base.GetCreatedAt()
	}
	// Set LastOccured timestamp to the latest timestamp.
	if base.GetLastOccurredAt().GetSeconds() > new.GetLastOccurredAt().GetSeconds() {
		new.LastOccurredAt = base.GetLastOccurredAt()
	}
	new.NumOccurrences = base.GetNumOccurrences() + 1
}
