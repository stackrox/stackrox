package new

import (
	"reflect"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	// CreateTableComplianceOperatorCheckResultV2Stmt holds the create statement for table `compliance_operator_check_result_v2`.
	CreateTableComplianceOperatorCheckResultV2Stmt = &postgres.CreateStmts{
		GormModel: (*ComplianceOperatorCheckResultV2)(nil),
		Children:  []*postgres.CreateStmts{},
	}

	// ComplianceOperatorCheckResultV2Schema is the go schema for table `compliance_operator_check_result_v2`.
	ComplianceOperatorCheckResultV2Schema = func() *walker.Schema {
		schema := walker.Walk(reflect.TypeOf((*storage.ComplianceOperatorCheckResultV2)(nil)), "compliance_operator_check_result_v2")
		schema.ScopingResource = resources.Compliance
		return schema
	}()
)

const (
	// ComplianceOperatorCheckResultV2TableName specifies the name of the table in postgres.
	ComplianceOperatorCheckResultV2TableName = "compliance_operator_check_result_v2"
)

// ComplianceOperatorCheckResultV2 holds the Gorm model for Postgres table `compliance_operator_check_result_v2`.
type ComplianceOperatorCheckResultV2 struct {
	ID             string                                              `gorm:"column:id;type:varchar;primaryKey"`
	CheckID        string                                              `gorm:"column:checkid;type:varchar"`
	CheckName      string                                              `gorm:"column:checkname;type:varchar"`
	ClusterID      string                                              `gorm:"column:clusterid;type:uuid;index:complianceoperatorcheckresultv2_sac_filter,type:hash"`
	Status         storage.ComplianceOperatorCheckResultV2_CheckStatus `gorm:"column:status;type:integer"`
	Severity       storage.RuleSeverity                                `gorm:"column:severity;type:integer"`
	CreatedTime    *time.Time                                          `gorm:"column:createdtime;type:timestamp"`
	ScanConfigName string                                              `gorm:"column:scanconfigname;type:varchar"`
	Rationale      string                                              `gorm:"column:rationale;type:varchar"`
	ScanRefID      string                                              `gorm:"column:scanrefid;type:uuid"`
	RuleRefID      string                                              `gorm:"column:rulerefid;type:uuid"`
	Serialized     []byte                                              `gorm:"column:serialized;type:bytea"`
}
