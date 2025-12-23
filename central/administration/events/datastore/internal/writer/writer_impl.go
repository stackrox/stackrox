package writer

import (
	"context"
	"maps"
	"slices"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/administration/events/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	_ Writer = (*writerImpl)(nil)

	maxWriterSize = 1000
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

// flushEventsToDatabase performs database operations for a snapshot of events.
// This function does NOT hold the mutex and is safe to call with slow DB operations.
func (c *writerImpl) flushEventsToDatabase(ctx context.Context, eventsToFlush []*storage.AdministrationEvent) error {
	// Short-circuit if no events to flush
	if len(eventsToFlush) == 0 {
		return nil
	}

	ids := protoutils.GetIDs(eventsToFlush)

	// The events we currently hold in the buffer are de-duplicated within the context of the buffer. However, they are
	// not de-duplicated against stored events from the database. The reason why we do the de-duplication during Flush
	// instead of upon adding is to lower the amount of database operations, since adding events may be done quite
	// frequently (i.e. for each emitted log statement that is subject to administration events creation).
	//
	// Database query (no mutex held)
	storedEvents, missingIndicies, err := c.store.GetMany(ctx, ids)
	if err != nil {
		return errors.Wrap(err, "failed to query for existing records")
	}

	// Event merging logic (in-memory, no mutex needed)
	// Remove the events that didn't exist within the database from the events to add and keep them separate.
	// This way, the indices match between stored events and events to add.
	notStoredEvents := getNotStoredEvents(eventsToFlush, missingIndicies)
	mergedEvents := sliceutils.Without(eventsToFlush, notStoredEvents)

	// Merge the events in the buffer with the stored event's information (i.e. timestamps and occurrences).
	for i, storedEvent := range storedEvents {
		mergedEvents[i] = mergeEvents(mergedEvents[i], storedEvent)
	}

	// Database upsert (no mutex held)
	// After merging events, upsert both the newly added events, and the merged events.
	if err := c.store.UpsertMany(ctx, append(mergedEvents, notStoredEvents...)); err != nil {
		return errors.Wrap(err, "failed to upsert events chunk")
	}

	return nil
}

func (c *writerImpl) flushNoLock(ctx context.Context) error {
	// Caller (Upsert) already holds mutex
	// Extract events and clear buffer
	eventsToFlush := slices.Collect(maps.Values(c.buffer))
	c.resetNoLock()

	// IMPORTANT: Release mutex BEFORE database operations to prevent timeouts.
	// With a single DB connection, holding mutex while waiting for the connection causes timeouts:
	// - Thread A: Holds mutex → waits for DB connection (blocked for >10s) → PANIC
	// - Thread B: Holds DB connection → waits for mutex (blocked)
	// Solution: Acquire mutex ONLY for updating the in-memory buffer, not during DB operations.
	c.mutex.Unlock()

	// Perform database operations without holding mutex
	err := c.flushEventsToDatabase(ctx, eventsToFlush)

	// Re-acquire mutex before returning to caller (Upsert expects lock held)
	c.mutex.Lock()

	return err
}

func (c *writerImpl) Upsert(ctx context.Context, event *events.AdministrationEvent) error {
	if err := event.Validate(); err != nil {
		return err
	}

	enrichedEvent := event.ToStorageEvent()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if len(c.buffer) >= maxWriterSize {
		if err := c.flushNoLock(ctx); err != nil {
			return errors.Wrap(err, "failed to flush events when buffer was full")
		}
	}

	var baseEvent *storage.AdministrationEvent
	// We operate the de-duplication under the context of the buffer and are not taking into account events within
	// the database. This will be done once Flush is called. The reason for this is that upserting events is done in a
	// high frequency (i.e. each time a log statement is issued that is subject to administration event creation), and
	// we want to avoid high bursts of read operations for the database and instead do those _only_ during the flush
	// operation.
	eventInBuffer, foundInBuffer := c.readNoLock(enrichedEvent.GetId())
	if foundInBuffer {
		baseEvent = eventInBuffer
	}
	// Merge events to up the occurrence and update the time stamps.
	if baseEvent != nil {
		enrichedEvent = mergeEvents(enrichedEvent, baseEvent)
	}

	c.writeNoLock(enrichedEvent)
	return nil
}

func (c *writerImpl) Flush(ctx context.Context) error {
	// Acquire mutex ONLY to snapshot and clear the buffer.
	// This prevents mutex timeout when DB operations are slow.
	c.mutex.Lock()
	eventsToFlush := slices.Collect(maps.Values(c.buffer))
	c.resetNoLock()
	c.mutex.Unlock()

	// Perform database operations WITHOUT holding mutex.
	// This prevents mutex timeout when DB operations take >10 seconds.
	return c.flushEventsToDatabase(ctx, eventsToFlush)
}

// Modifies `updated` with the values of the base event and returns the merged event.
func mergeEvents(updated *storage.AdministrationEvent, base *storage.AdministrationEvent) *storage.AdministrationEvent {
	if base == nil {
		return updated
	}
	if updated == nil {
		updated = base
		return updated
	}

	// Set CreatedAt timestamp to the earliest timestamp.
	if base.GetCreatedAt().GetSeconds() < updated.GetCreatedAt().GetSeconds() {
		updated.CreatedAt = base.GetCreatedAt()
	}
	// Set LastOccurred timestamp to the latest timestamp.
	if base.GetLastOccurredAt().GetSeconds() > updated.GetLastOccurredAt().GetSeconds() {
		updated.LastOccurredAt = base.GetLastOccurredAt()
	}
	updated.NumOccurrences = base.GetNumOccurrences() + 1
	return updated
}

func getNotStoredEvents(events []*storage.AdministrationEvent, eventsNotStoredIndices []int) []*storage.AdministrationEvent {
	eventsNotStored := make([]*storage.AdministrationEvent, 0, len(eventsNotStoredIndices))
	for _, index := range eventsNotStoredIndices {
		eventsNotStored = append(eventsNotStored, events[index])
	}
	return eventsNotStored
}
