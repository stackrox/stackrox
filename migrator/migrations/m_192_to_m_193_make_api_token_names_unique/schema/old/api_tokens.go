package schema

import (
	"reflect"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
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
		schema.SetOptionsMap(search.Walk(v1.SearchCategory_API_TOKEN, "tokenmetadata", (*storage.TokenMetadata)(nil)))
		schema.ScopingResource = resources.Integration
		return schema
	}()
)

// APITokens holds the Gorm model for Postgres table `api_tokens`.
type APITokens struct {
	ID         string     `gorm:"column:id;type:varchar;primaryKey"`
	Expiration *time.Time `gorm:"column:expiration;type:timestamp"`
	Revoked    bool       `gorm:"column:revoked;type:bool"`
	Serialized []byte     `gorm:"column:serialized;type:bytea"`
}
