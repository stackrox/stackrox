package schema

import (
	"reflect"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

var (
	// CreateTableIntegrationHealthsStmt holds the create statement for table `integration_healths`.
	CreateTableIntegrationHealthsStmt = &postgres.CreateStmts{
		GormModel: (*IntegrationHealths)(nil),
		Children:  []*postgres.CreateStmts{},
	}

	// IntegrationHealthsSchema is the go schema for table `integration_healths`.
	IntegrationHealthsSchema = func() *walker.Schema {
		schema := walker.Walk(reflect.TypeOf((*storage.IntegrationHealth)(nil)), "integration_healths")
		return schema
	}()
)

// IntegrationHealths holds the Gorm model for Postgres table `integration_healths`.
type IntegrationHealths struct {
	ID         string `gorm:"column:id;type:varchar;primaryKey"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}
