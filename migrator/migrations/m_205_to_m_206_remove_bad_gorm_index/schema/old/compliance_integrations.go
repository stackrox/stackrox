package old

import (
	"reflect"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
)

var (
	// CreateTableComplianceIntegrationsStmt holds the create statement for table `compliance_integrations`.
	CreateTableComplianceIntegrationsStmt = &postgres.CreateStmts{
		GormModel: (*ComplianceIntegrations)(nil),
		Children:  []*postgres.CreateStmts{},
	}

	// ComplianceIntegrationsSchema is the go schema for table `compliance_integrations`.
	ComplianceIntegrationsSchema = func() *walker.Schema {
		schema := walker.Walk(reflect.TypeOf((*storage.ComplianceIntegration)(nil)), "compliance_integrations")
		schema.SetOptionsMap(search.Walk(v1.SearchCategory_COMPLIANCE_INTEGRATIONS, "complianceintegration", (*storage.ComplianceIntegration)(nil)))
		schema.ScopingResource = resources.Compliance
		return schema
	}()
)

const (
	// ComplianceIntegrationsTableName specifies the name of the table in postgres.
	ComplianceIntegrationsTableName = "compliance_integrations"
)

// ComplianceIntegrations holds the Gorm model for Postgres table `compliance_integrations`.
type ComplianceIntegrations struct {
	ID                string           `gorm:"column:id;type:uuid;primaryKey"`
	Version           string           `gorm:"column:version;type:varchar"`
	ClusterID         string           `gorm:"column:clusterid;type:uuid;uniqueIndex:compliance_unique_indicator;index:complianceintegrations_sac_filter,type:hash"`
	OperatorInstalled bool             `gorm:"column:operatorinstalled;type:bool"`
	OperatorStatus    storage.COStatus `gorm:"column:operatorstatus;type:integer"`
	Serialized        []byte           `gorm:"column:serialized;type:bytea"`
}
