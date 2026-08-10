// Frozen pre-PR#21423 schema copied from release-4.11.

package schema

import (
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableRolesStmt holds the create statement for table `roles`.
	CreateTableRolesStmt = &postgres.CreateStmts{
		GormModel: (*Roles)(nil),
		Children:  []*postgres.CreateStmts{},
	}
)

const (
	// RolesTableName specifies the name of the table in postgres.
	RolesTableName = "roles"
)

// Roles holds the Gorm model for Postgres table `roles`.
type Roles struct {
	Name       string `gorm:"column:name;type:varchar;primaryKey"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}
