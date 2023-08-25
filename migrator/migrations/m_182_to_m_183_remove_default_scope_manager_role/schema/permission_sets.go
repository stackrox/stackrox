// Originally copied from pkg/postgres/schema/permission_sets.go

package schema

import (
	"reflect"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

var (
	// CreateTablePermissionSetsStmt holds the create statement for table `permission_sets`.
	CreateTablePermissionSetsStmt = &postgres.CreateStmts{
		GormModel: (*PermissionSets)(nil),
		Children:  []*postgres.CreateStmts{},
	}

	// PermissionSetsSchema is the go schema for table `permission_sets`.
	PermissionSetsSchema = func() *walker.Schema {
		schema := walker.Walk(reflect.TypeOf((*storage.PermissionSet)(nil)), "permission_sets")
		return schema
	}()
)

// PermissionSets holds the Gorm model for Postgres table `permission_sets`.
type PermissionSets struct {
	ID         string `gorm:"column:id;type:uuid;primaryKey"`
	Name       string `gorm:"column:name;type:varchar;unique"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}
