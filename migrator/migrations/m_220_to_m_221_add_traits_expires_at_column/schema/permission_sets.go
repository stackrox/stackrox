package schema

import (
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTablePermissionSetsStmt holds the create statement for table `permission_sets`.
	CreateTablePermissionSetsStmt = &postgres.CreateStmts{
		GormModel: (*PermissionSets)(nil),
		Children:  []*postgres.CreateStmts{},
	}
)

// PermissionSets holds the Gorm model for Postgres table `permission_sets`.
type PermissionSets struct {
	ID         string `gorm:"column:id;type:uuid;primaryKey"`
	Name       string `gorm:"column:name;type:varchar;unique"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}
