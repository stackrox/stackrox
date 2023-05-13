package schema

import (
	"reflect"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

var (
	// CreateTableContinuousIntegrationConfigsStmt holds the create statement for table `continuous_integration_configs`.
	CreateTableContinuousIntegrationConfigsStmt = &postgres.CreateStmts{
		GormModel: (*ContinuousIntegrationConfigs)(nil),
		Children:  []*postgres.CreateStmts{},
	}

	// ContinuousIntegrationConfigsSchema is the go schema for table `continuous_integration_configs`.
	ContinuousIntegrationConfigsSchema = func() *walker.Schema {
		schema := walker.Walk(reflect.TypeOf((*storage.ContinuousIntegrationConfig)(nil)), "continuous_integration_configs")
		return schema
	}()
)

// ContinuousIntegrationConfigs holds the Gorm model for Postgres table `continuous_integration_configs`.
type ContinuousIntegrationConfigs struct {
	Id         string `gorm:"column:id;type:uuid;primaryKey"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}
