package writer

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/administration/events/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	_ Writer = (*writerImpl)(nil)

	rootNamespaceUUID = uuid.FromStringOrNil("d4dcc3d8-fcdf-4621-8386-0be1372ecbba")
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

func (c *writerImpl) Upsert(ctx context.Context, event *storage.AdministrationEvent) error {
	enrichedEvent := enrichEventWithDefaults(event)
	id := enrichedEvent.GetId()

	var baseEvent *storage.AdministrationEvent

	c.mutex.Lock()
	defer c.mutex.Unlock()

	eventInBuffer, found := c.readNoLock(id)
	// If an event already exists in the buffer it is the most recent.
	// We use it as a base to merge with the new event.
	if found {
		baseEvent = eventInBuffer
	} else {
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
		enrichedEvent = mergeEvents(baseEvent, enrichedEvent)
	}

	c.writeNoLock(enrichedEvent)
	return nil
}

func (c *writerImpl) Flush(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	eventChunk := make([]*storage.AdministrationEvent, 0, len(c.buffer))
	for _, event := range c.buffer {
		eventChunk = append(eventChunk, event)
	}
	c.buffer = make(map[string]*storage.AdministrationEvent)
	err := c.store.UpsertMany(ctx, eventChunk)
	if err != nil {
		return errors.Wrap(err, "failed to upsert event chunk")
	}
	return nil
}

func getEventID(event *storage.AdministrationEvent) string {
	dedupKey := strings.Join([]string{
		event.GetDomain(),
		event.GetMessage(),
		event.GetResourceId(),
		event.GetResourceType(),
		event.GetType().String(),
	}, ",")
	return uuid.NewV5(rootNamespaceUUID, dedupKey).String()
}

func enrichEventWithDefaults(event *storage.AdministrationEvent) *storage.AdministrationEvent {
	if event == nil {
		return nil
	}

	enrichedEvent := event.Clone()
	enrichedEvent.Id = getEventID(event)
	if event.GetNumOccurrences() == 0 {
		enrichedEvent.NumOccurrences = 1
	}
	if event.GetCreatedAt() == nil {
		enrichedEvent.CreatedAt = protoconv.ConvertTimeToTimestamp(time.Now())
	}
	if event.GetLastOccurredAt() == nil {
		enrichedEvent.LastOccurredAt = protoconv.ConvertTimeToTimestamp(time.Now())
	}
	return enrichedEvent
}

func mergeEvents(base *storage.AdministrationEvent, new *storage.AdministrationEvent) *storage.AdministrationEvent {
	if base == nil {
		return nil
	}
	if new == nil {
		return base
	}

	mergedEvent := base.Clone()

	// Set CreatedAt timestamp to the earliest timestamp.
	if new.GetCreatedAt().GetSeconds() < base.GetCreatedAt().GetSeconds() {
		mergedEvent.CreatedAt = new.GetCreatedAt()
	}
	// Set LastOccured timestamp to the latest timestamp.
	if new.GetLastOccurredAt().GetSeconds() > base.GetLastOccurredAt().GetSeconds() {
		mergedEvent.LastOccurredAt = new.GetLastOccurredAt()
	}
	mergedEvent.NumOccurrences++
	return mergedEvent
}
