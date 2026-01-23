package schema

import (
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableSimpleAccessScopesStmt holds the create statement for table `simple_access_scopes`.
	CreateTableSimpleAccessScopesStmt = &postgres.CreateStmts{
		GormModel: (*SimpleAccessScopes)(nil),
		Children:  []*postgres.CreateStmts{},
	}
)

// SimpleAccessScopes holds the Gorm model for Postgres table `simple_access_scopes`.
type SimpleAccessScopes struct {
	ID         string `gorm:"column:id;type:uuid;primaryKey"`
	Name       string `gorm:"column:name;type:varchar;unique"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}
