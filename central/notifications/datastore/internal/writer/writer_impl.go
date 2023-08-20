package writer

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/notifications/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	_ Writer = (*writerImpl)(nil)

	rootNamespaceUUID = uuid.Nil
)

type writerImpl struct {
	mutex sync.Mutex

	buffer map[string]*storage.Notification
	store  store.Store
}

func (c *writerImpl) read(id string) (*storage.Notification, bool) {
	notification, found := c.buffer[id]
	return notification, found
}

func (c *writerImpl) write(notification *storage.Notification) {
	c.buffer[notification.GetId()] = notification
}

func (c *writerImpl) Upsert(ctx context.Context, notification *storage.Notification) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	enrichedNotification := enrichNotificationWithDefaults(notification)
	id := enrichedNotification.GetId()

	var baseNotification *storage.Notification

	notificationInBuffer, found := c.read(id)
	// If a notification already exists in the buffer it is the most recent.
	// We use it as a base to merge with the new notification.
	if found {
		baseNotification = notificationInBuffer
	} else {
		// If no notification is in the buffer, we try to fetch a notification
		// from the database. If found, we use it as the base for the merge.
		notificationInDB, found, err := c.store.Get(ctx, id)
		if err != nil {
			return errors.Wrap(err, "failed to query for existing record")
		}
		if found {
			baseNotification = notificationInDB
		}
	}

	// Merge notifications to up the occurrence and update the time stamps.
	if baseNotification != nil {
		c.write(mergeNotifications(baseNotification, enrichedNotification))
		return nil
	}

	// No notification with the dedup id exists in the buffer or the database.
	// We simply write the new enriched notification to the buffer.
	c.write(enrichedNotification)
	return nil
}

func (c *writerImpl) Flush(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	notificationChunk := make([]*storage.Notification, 0, len(c.buffer))
	for _, notification := range c.buffer {
		notificationChunk = append(notificationChunk, notification)
	}
	c.buffer = make(map[string]*storage.Notification)
	err := c.store.UpsertMany(ctx, notificationChunk)
	if err != nil {
		return errors.Wrap(err, "failed to upsert notification chunk")
	}
	return nil
}

func getNotificationID(notification *storage.Notification) string {
	dedupKey := strings.Join([]string{
		notification.GetDomain(),
		notification.GetMessage(),
		notification.GetResourceId(),
		notification.GetResourceType(),
		notification.GetType().String(),
	}, ",")
	return uuid.NewV5(rootNamespaceUUID, dedupKey).String()
}

func enrichNotificationWithDefaults(notification *storage.Notification) *storage.Notification {
	if notification == nil {
		return nil
	}

	enrichedNotification := notification.Clone()
	id := getNotificationID(notification)
	enrichedNotification.Id = id
	if notification.Occurrences == 0 {
		enrichedNotification.Occurrences = 1
	}
	if notification.CreatedAt == nil {
		enrichedNotification.CreatedAt = protoconv.ConvertTimeToTimestamp(time.Now())
	}
	if notification.CreatedAt == nil {
		enrichedNotification.LastOccurred = protoconv.ConvertTimeToTimestamp(time.Now())
	}
	return enrichedNotification
}

func mergeNotifications(base *storage.Notification, new *storage.Notification) *storage.Notification {
	if base == nil {
		return nil
	}
	if new == nil {
		return base
	}

	mergedNotification := base.Clone()

	// Set CreatedAt timestamp to the earliest timestamp.
	if new.CreatedAt.GetSeconds() < base.CreatedAt.GetSeconds() {
		mergedNotification.CreatedAt = new.CreatedAt
	}
	// Set LastOccured timestamp to the latest timestamp.
	if new.LastOccurred.GetSeconds() > base.LastOccurred.GetSeconds() {
		mergedNotification.LastOccurred = new.LastOccurred
	}
	mergedNotification.Occurrences++
	return mergedNotification
}
