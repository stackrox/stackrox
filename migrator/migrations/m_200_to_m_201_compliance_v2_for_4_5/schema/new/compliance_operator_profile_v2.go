package new

import (
	"reflect"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	// CreateTableComplianceOperatorProfileV2Stmt holds the create statement for table `compliance_operator_profile_v2`.
	CreateTableComplianceOperatorProfileV2Stmt = &postgres.CreateStmts{
		GormModel: (*ComplianceOperatorProfileV2)(nil),
		Children: []*postgres.CreateStmts{
			{
				GormModel: (*ComplianceOperatorProfileV2Rules)(nil),
				Children:  []*postgres.CreateStmts{},
			},
		},
	}

	// ComplianceOperatorProfileV2Schema is the go schema for table `compliance_operator_profile_v2`.
	ComplianceOperatorProfileV2Schema = func() *walker.Schema {
		schema := walker.Walk(reflect.TypeOf((*storage.ComplianceOperatorProfileV2)(nil)), "compliance_operator_profile_v2")
		schema.ScopingResource = resources.Compliance
		return schema
	}()
)

const (
	// ComplianceOperatorProfileV2TableName specifies the name of the table in postgres.
	ComplianceOperatorProfileV2TableName = "compliance_operator_profile_v2"
	// ComplianceOperatorProfileV2RulesTableName specifies the name of the table in postgres.
	ComplianceOperatorProfileV2RulesTableName = "compliance_operator_profile_v2_rules"
)

// ComplianceOperatorProfileV2 holds the Gorm model for Postgres table `compliance_operator_profile_v2`.
type ComplianceOperatorProfileV2 struct {
	ID             string `gorm:"column:id;type:varchar;primaryKey"`
	ProfileID      string `gorm:"column:profileid;type:varchar"`
	Name           string `gorm:"column:name;type:varchar"`
	ProfileVersion string `gorm:"column:profileversion;type:varchar"`
	ProductType    string `gorm:"column:producttype;type:varchar"`
	Standard       string `gorm:"column:standard;type:varchar"`
	ClusterID      string `gorm:"column:clusterid;type:uuid;index:complianceoperatorprofilev2_sac_filter,type:hash"`
	ProfileRefID   string `gorm:"column:profilerefid;type:uuid"`
	Serialized     []byte `gorm:"column:serialized;type:bytea"`
}

// ComplianceOperatorProfileV2Rules holds the Gorm model for Postgres table `compliance_operator_profile_v2_rules`.
type ComplianceOperatorProfileV2Rules struct {
	ComplianceOperatorProfileV2ID  string                      `gorm:"column:compliance_operator_profile_v2_id;type:varchar;primaryKey"`
	Idx                            int                         `gorm:"column:idx;type:integer;primaryKey;index:complianceoperatorprofilev2rules_idx,type:btree"`
	RuleName                       string                      `gorm:"column:rulename;type:varchar"`
	ComplianceOperatorProfileV2Ref ComplianceOperatorProfileV2 `gorm:"foreignKey:compliance_operator_profile_v2_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}
