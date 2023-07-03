package schema

import (
	"reflect"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

var (
	// CreateTableEventsStmt holds the create statement for table `events`.
	CreateTableEventsStmt = &postgres.CreateStmts{
		GormModel: (*Events)(nil),
		Children:  []*postgres.CreateStmts{},
	}

	// EventsSchema is the go schema for table `events`.
	EventsSchema = func() *walker.Schema {
		schema := walker.Walk(reflect.TypeOf((*storage.Event)(nil)), "events")
		return schema
	}()
)

// Events holds the Gorm model for Postgres table `events`.
type Events struct {
	ID         string `gorm:"column:id;type:uuid;primaryKey"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}
