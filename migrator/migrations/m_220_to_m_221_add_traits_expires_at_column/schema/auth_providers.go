package schema

import (
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableAuthProvidersStmt holds the create statement for table `auth_providers`.
	CreateTableAuthProvidersStmt = &postgres.CreateStmts{
		GormModel: (*AuthProviders)(nil),
		Children:  []*postgres.CreateStmts{},
	}
)

// AuthProviders holds the Gorm model for Postgres table `auth_providers`.
type AuthProviders struct {
	ID         string `gorm:"column:id;type:varchar;primaryKey"`
	Name       string `gorm:"column:name;type:varchar;unique"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}
