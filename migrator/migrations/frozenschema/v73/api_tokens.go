package schema

import (
	"reflect"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

var (
	// CreateTableAPITokensStmt holds the create statement for table `api_tokens`.
	CreateTableAPITokensStmt = &postgres.CreateStmts{
		GormModel: (*APITokens)(nil),
		Children:  []*postgres.CreateStmts{},
	}

	// APITokensSchema is the go schema for table `api_tokens`.
	APITokensSchema = func() *walker.Schema {
		schema := walker.Walk(reflect.TypeOf((*storage.TokenMetadata)(nil)), "api_tokens")
		return schema
	}()
)

const (
	// APITokensTableName is the name of the table used for storage.
	APITokensTableName = "api_tokens"
)

// APITokens holds the Gorm model for Postgres table `api_tokens`.
type APITokens struct {
	ID         string `gorm:"column:id;type:varchar;primaryKey"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}
