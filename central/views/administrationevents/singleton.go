package administrationevents

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	view EventView
)

// NewEventView returns a view for searching administration events.
func NewEventView(db postgres.DB) EventView {
	return &eventViewImpl{db: db, schema: schema.AdministrationEventsSchema}
}

// Singleton returns the administration event view.
func Singleton() EventView {
	once.Do(func() {
		view = NewEventView(globaldb.GetPostgres())
	})
	return view
}
