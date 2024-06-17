package new

import (
	"reflect"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	// CreateTableComplianceOperatorRuleV2Stmt holds the create statement for table `compliance_operator_rule_v2`.
	CreateTableComplianceOperatorRuleV2Stmt = &postgres.CreateStmts{
		GormModel: (*ComplianceOperatorRuleV2)(nil),
		Children: []*postgres.CreateStmts{
			{
				GormModel: (*ComplianceOperatorRuleV2Controls)(nil),
				Children:  []*postgres.CreateStmts{},
			},
		},
	}

	// ComplianceOperatorRuleV2Schema is the go schema for table `compliance_operator_rule_v2`.
	ComplianceOperatorRuleV2Schema = func() *walker.Schema {
		schema := walker.Walk(reflect.TypeOf((*storage.ComplianceOperatorRuleV2)(nil)), "compliance_operator_rule_v2")
		schema.ScopingResource = resources.Compliance
		return schema
	}()
)

const (
	// ComplianceOperatorRuleV2TableName specifies the name of the table in postgres.
	ComplianceOperatorRuleV2TableName = "compliance_operator_rule_v2"
	// ComplianceOperatorRuleV2ControlsTableName specifies the name of the table in postgres.
	ComplianceOperatorRuleV2ControlsTableName = "compliance_operator_rule_v2_controls"
)

// ComplianceOperatorRuleV2 holds the Gorm model for Postgres table `compliance_operator_rule_v2`.
type ComplianceOperatorRuleV2 struct {
	ID         string               `gorm:"column:id;type:varchar;primaryKey"`
	Name       string               `gorm:"column:name;type:varchar"`
	RuleType   string               `gorm:"column:ruletype;type:varchar"`
	Severity   storage.RuleSeverity `gorm:"column:severity;type:integer"`
	ClusterID  string               `gorm:"column:clusterid;type:uuid;index:complianceoperatorrulev2_sac_filter,type:hash"`
	RuleRefID  string               `gorm:"column:rulerefid;type:uuid"`
	Serialized []byte               `gorm:"column:serialized;type:bytea"`
}

// ComplianceOperatorRuleV2Controls holds the Gorm model for Postgres table `compliance_operator_rule_v2_controls`.
type ComplianceOperatorRuleV2Controls struct {
	ComplianceOperatorRuleV2ID  string                   `gorm:"column:compliance_operator_rule_v2_id;type:varchar;primaryKey"`
	Idx                         int                      `gorm:"column:idx;type:integer;primaryKey;index:complianceoperatorrulev2controls_idx,type:btree"`
	Standard                    string                   `gorm:"column:standard;type:varchar"`
	Control                     string                   `gorm:"column:control;type:varchar"`
	ComplianceOperatorRuleV2Ref ComplianceOperatorRuleV2 `gorm:"foreignKey:compliance_operator_rule_v2_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}
