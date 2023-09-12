package writer

import (
	"context"

	"github.com/stackrox/rox/central/administration/events/datastore/internal/store"
	"github.com/stackrox/rox/pkg/administration/events"
)

// Writer implements a buffered write for the administration events datastore.
//
// When events are upserted to the writer, they first end up in a buffer.
// The buffered event has the most recent event state. If an entry for
// the event is already present in the data store, this record is merged
// with the buffered record. The buffer is written to the data store once
// the writer is flushed.
//
//go:generate mockgen-wrapper
type Writer interface {
	Upsert(ctx context.Context, obj *events.AdministrationEvent) error
	Flush(ctx context.Context) error
}

// New returns a new writer instance.
func New(_ store.Store) Writer {
	return nil
}
