// Code generated by pg-bindings generator. DO NOT EDIT.

package schema

import (
	"reflect"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
)

var (
	// CreateTableAuthProvidersStmt holds the create statement for table `auth_providers`.
	CreateTableAuthProvidersStmt = &postgres.CreateStmts{
		GormModel: (*AuthProviders)(nil),
		Children:  []*postgres.CreateStmts{},
	}

	// AuthProvidersSchema is the go schema for table `auth_providers`.
	AuthProvidersSchema = func() *walker.Schema {
		schema := GetSchemaForTable("auth_providers")
		if schema != nil {
			return schema
		}
		schema = walker.Walk(reflect.TypeOf((*storage.AuthProvider)(nil)), "auth_providers")
		schema.SetOptionsMap(search.Walk(v1.SearchCategory_AUTH_PROVIDERS, "authprovider", (*storage.AuthProvider)(nil)))
		schema.ScopingResource = resources.Access
		RegisterTable(schema, CreateTableAuthProvidersStmt)
		mapping.RegisterCategoryToTable(v1.SearchCategory_AUTH_PROVIDERS, schema)
		return schema
	}()
)

const (
	// AuthProvidersTableName specifies the name of the table in postgres.
	AuthProvidersTableName = "auth_providers"
)

// AuthProviders holds the Gorm model for Postgres table `auth_providers`.
type AuthProviders struct {
	ID         string `gorm:"column:id;type:varchar;primaryKey"`
	Name       string `gorm:"column:name;type:varchar;unique"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}
