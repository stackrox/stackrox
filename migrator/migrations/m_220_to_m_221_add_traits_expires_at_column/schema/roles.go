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

// Roles holds the Gorm model for Postgres table `roles`.
type Roles struct {
	Name       string `gorm:"column:name;type:varchar;primaryKey"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}
