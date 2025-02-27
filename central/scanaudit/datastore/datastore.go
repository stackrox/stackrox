package datastore

import (
	"context"

	"github.com/stackrox/rox/central/scanaudit/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/caudit"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
)

type DataStore interface {
	GetAuditEvents(ctx context.Context, imageID string) ([]*storage.ScanAudit, bool, error)
	UpsertEvent(context.Context, *storage.ScanAudit) error
	UpsertCTXEvents(ctx context.Context, status caudit.Status, imageID string, message string) error
}

func New(store store.Store) DataStore {
	return &datastoreImpl{
		store: store,
	}
}

type datastoreImpl struct {
	store store.Store
}

func (d *datastoreImpl) GetAuditEvents(ctx context.Context, imageID string) ([]*storage.ScanAudit, bool, error) {
	results, err := d.store.GetAll(ctx)
	if err != nil {
		return nil, false, err
	}
	if len(results) == 0 {
		return nil, false, nil
	}

	filtered := make([]*storage.ScanAudit, 0, len(results))
	for _, r := range results {
		if r.ImageID == imageID {
			filtered = append(filtered, r)
		}
	}

	return filtered, true, nil
}

func (d *datastoreImpl) UpsertEvent(ctx context.Context, event *storage.ScanAudit) error {
	return d.store.Upsert(ctx, event)
}

func (d *datastoreImpl) UpsertCTXEvents(ctx context.Context, status caudit.Status, imageID string, message string) error {
	events := caudit.Events(ctx)
	sEvents := make([]*storage.ScanAudit_ScanAuditEvent, 0, len(events))
	for _, e := range events {
		s := storage.ScanAudit_SUCCESS
		if e.Status == caudit.StatusFailure {
			s = storage.ScanAudit_FAILURE
		}

		sEvents = append(sEvents, &storage.ScanAudit_ScanAuditEvent{
			Time:    protocompat.ConvertTimeToTimestampOrNil(&e.Time),
			Status:  s,
			Message: e.Message,
		})
	}

	topStatus := storage.ScanAudit_SUCCESS
	if status == caudit.StatusFailure {
		topStatus = storage.ScanAudit_FAILURE
	}

	return d.store.Upsert(ctx, &storage.ScanAudit{
		Id:        uuid.NewV4().String(),
		EventTime: protocompat.TimestampNow(),
		ImageID:   imageID,
		Message:   message,
		Events:    sEvents,
		Status:    topStatus,
	})
}
