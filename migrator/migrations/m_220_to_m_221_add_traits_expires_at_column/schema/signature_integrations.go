package schema

import (
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableSignatureIntegrationsStmt holds the create statement for table `signature_integrations`.
	CreateTableSignatureIntegrationsStmt = &postgres.CreateStmts{
		GormModel: (*SignatureIntegrations)(nil),
		Children:  []*postgres.CreateStmts{},
	}
)

// SignatureIntegrations holds the Gorm model for Postgres table `signature_integrations`.
type SignatureIntegrations struct {
	ID         string `gorm:"column:id;type:varchar;primaryKey"`
	Name       string `gorm:"column:name;type:varchar;unique"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}
