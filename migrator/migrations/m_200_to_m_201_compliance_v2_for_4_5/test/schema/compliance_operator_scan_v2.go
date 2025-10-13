package schema

import (
	"reflect"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	// CreateTableComplianceOperatorScanV2Stmt holds the create statement for table `compliance_operator_scan_v2`.
	CreateTableComplianceOperatorScanV2Stmt = &postgres.CreateStmts{
		GormModel: (*ComplianceOperatorScanV2)(nil),
		Children:  []*postgres.CreateStmts{},
	}

	// ComplianceOperatorScanV2Schema is the go schema for table `compliance_operator_scan_v2`.
	ComplianceOperatorScanV2Schema = func() *walker.Schema {
		schema := walker.Walk(reflect.TypeOf((*storage.ComplianceOperatorScanV2)(nil)), "compliance_operator_scan_v2")

		schema.ScopingResource = resources.Compliance
		return schema
	}()
)

const (
	// ComplianceOperatorScanV2TableName specifies the name of the table in postgres.
	ComplianceOperatorScanV2TableName = "compliance_operator_scan_v2"
)

// ComplianceOperatorScanV2 holds the Gorm model for Postgres table `compliance_operator_scan_v2`.
type ComplianceOperatorScanV2 struct {
	ID               string           `gorm:"column:id;type:varchar;primaryKey"`
	ScanConfigName   string           `gorm:"column:scanconfigname;type:varchar"`
	ScanName         string           `gorm:"column:scanname;type:varchar"`
	ClusterID        string           `gorm:"column:clusterid;type:uuid;index:complianceoperatorscanv2_sac_filter,type:hash"`
	ProfileProfileID string           `gorm:"column:profile_profileid;type:varchar"`
	ScanType         storage.ScanType `gorm:"column:scantype;type:integer"`
	StatusResult     string           `gorm:"column:status_result;type:varchar"`
	LastExecutedTime *time.Time       `gorm:"column:lastexecutedtime;type:timestamp"`
	Serialized       []byte           `gorm:"column:serialized;type:bytea"`
}
