// Originally copied from pkg/postgres/schema/roles.go

package schema

import (
	"reflect"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

var (
	// CreateTableRolesStmt holds the create statement for table `roles`.
	CreateTableRolesStmt = &postgres.CreateStmts{
		GormModel: (*Roles)(nil),
		Children:  []*postgres.CreateStmts{},
	}

	// RolesSchema is the go schema for table `roles`.
	RolesSchema = func() *walker.Schema {
		schema := walker.Walk(reflect.TypeOf((*storage.Role)(nil)), "roles")
		return schema
	}()
)

// Roles holds the Gorm model for Postgres table `roles`.
type Roles struct {
	Name       string `gorm:"column:name;type:varchar;primaryKey"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}
