package writer

import (
	"context"

	"github.com/stackrox/rox/central/administration/events/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
)

// Writer implements a buffered write for the administration events datastore.
//
// Since we generate events from logs, there could be many log events created
// in a small timeframe. The buffer is essentially a write optimization such
// that we don't have to perform an upsert query for each event individually.
// While event ingestion would probably be fine without the buffer from Central's
// perspective, it would increase the load on the database due to many concurrent
// transactions and table locks.
//
// When events are upserted to the writer, they first end up in a buffer.
// The buffered event has the most recent event state. The buffer is written to the
// data store once the writer is flushed. If an entry for the event is already
// present in the data store, this record is merged with the buffered record.
//
//go:generate mockgen-wrapper
type Writer interface {
	Upsert(ctx context.Context, obj *events.AdministrationEvent) error
	Flush(ctx context.Context) error
}

// New returns a new writer instance.
func New(store store.Store) Writer {
	return &writerImpl{
		buffer: make(map[string]*storage.AdministrationEvent),
		store:  store,
	}
}
