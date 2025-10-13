package schema

import (
	"reflect"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	// CreateTableAuthProvidersStmt holds the create statement for table `auth_providers`.
	CreateTableAuthProvidersStmt = &postgres.CreateStmts{
		GormModel: (*AuthProviders)(nil),
		Children:  []*postgres.CreateStmts{},
	}

	// AuthProvidersSchema is the go schema for table `auth_providers`.
	AuthProvidersSchema = func() *walker.Schema {
		schema := walker.Walk(reflect.TypeOf((*storage.AuthProvider)(nil)), "auth_providers")
		schema.ScopingResource = resources.Access
		return schema
	}()
)

// AuthProviders holds the Gorm model for Postgres table `auth_providers`.
type AuthProviders struct {
	ID         string `gorm:"column:id;type:varchar;primaryKey"`
	Name       string `gorm:"column:name;type:varchar;unique"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}
