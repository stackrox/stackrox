// Frozen pre-migration GORM schema for compliance_operator_scan_configuration_v2.
// Reproduces old index tags so AutoMigrate creates the _idx indexes that the migration drops.

package schema

import (
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableComplianceOperatorScanConfigurationV2Stmt holds the create statement for table `compliance_operator_scan_configuration_v2`.
	CreateTableComplianceOperatorScanConfigurationV2Stmt = &postgres.CreateStmts{
		GormModel: (*ComplianceOperatorScanConfigurationV2)(nil),
		Children: []*postgres.CreateStmts{
			{GormModel: (*ComplianceOperatorScanConfigurationV2Profiles)(nil), Children: []*postgres.CreateStmts{}},
			{GormModel: (*ComplianceOperatorScanConfigurationV2Clusters)(nil), Children: []*postgres.CreateStmts{}},
			{GormModel: (*ComplianceOperatorScanConfigurationV2Notifiers)(nil), Children: []*postgres.CreateStmts{}},
		},
	}
)

// ComplianceOperatorScanConfigurationV2 holds the Gorm model for Postgres table `compliance_operator_scan_configuration_v2`.
type ComplianceOperatorScanConfigurationV2 struct {
	ID             string `gorm:"column:id;type:uuid;primaryKey"`
	ScanConfigName string `gorm:"column:scanconfigname;type:varchar;unique"`
	ModifiedByName string `gorm:"column:modifiedby_name;type:varchar"`
	Serialized     []byte `gorm:"column:serialized;type:bytea"`
}

// TableName returns the table name for GORM.
func (ComplianceOperatorScanConfigurationV2) TableName() string {
	return "compliance_operator_scan_configuration_v2"
}

// ComplianceOperatorScanConfigurationV2Profiles holds the Gorm model for Postgres table `compliance_operator_scan_configuration_v2_profiles`.
type ComplianceOperatorScanConfigurationV2Profiles struct {
	ComplianceOperatorScanConfigurationV2ID  string                                `gorm:"column:compliance_operator_scan_configuration_v2_id;type:uuid;primaryKey"`
	Idx                                      int                                   `gorm:"column:idx;type:integer;primaryKey;index:complianceoperatorscanconfigurationv2profiles_idx,type:btree"`
	ProfileName                              string                                `gorm:"column:profilename;type:varchar"`
	ComplianceOperatorScanConfigurationV2Ref ComplianceOperatorScanConfigurationV2 `gorm:"foreignKey:compliance_operator_scan_configuration_v2_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

// TableName returns the table name for GORM.
func (ComplianceOperatorScanConfigurationV2Profiles) TableName() string {
	return "compliance_operator_scan_configuration_v2_profiles"
}

// ComplianceOperatorScanConfigurationV2Clusters holds the Gorm model for Postgres table `compliance_operator_scan_configuration_v2_clusters`.
type ComplianceOperatorScanConfigurationV2Clusters struct {
	ComplianceOperatorScanConfigurationV2ID  string                                `gorm:"column:compliance_operator_scan_configuration_v2_id;type:uuid;primaryKey"`
	Idx                                      int                                   `gorm:"column:idx;type:integer;primaryKey;index:complianceoperatorscanconfigurationv2clusters_idx,type:btree"`
	ClusterID                                string                                `gorm:"column:clusterid;type:uuid;index:complianceoperatorscanconfigurationv2clusters_sac_filter,type:hash"`
	ComplianceOperatorScanConfigurationV2Ref ComplianceOperatorScanConfigurationV2 `gorm:"foreignKey:compliance_operator_scan_configuration_v2_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

// TableName returns the table name for GORM.
func (ComplianceOperatorScanConfigurationV2Clusters) TableName() string {
	return "compliance_operator_scan_configuration_v2_clusters"
}

// ComplianceOperatorScanConfigurationV2Notifiers holds the Gorm model for Postgres table `compliance_operator_scan_configuration_v2_notifiers`.
type ComplianceOperatorScanConfigurationV2Notifiers struct {
	ComplianceOperatorScanConfigurationV2ID  string                                `gorm:"column:compliance_operator_scan_configuration_v2_id;type:uuid;primaryKey"`
	Idx                                      int                                   `gorm:"column:idx;type:integer;primaryKey;index:complianceoperatorscanconfigurationv2notifiers_idx,type:btree"`
	ID                                       string                                `gorm:"column:id;type:varchar"`
	ComplianceOperatorScanConfigurationV2Ref ComplianceOperatorScanConfigurationV2 `gorm:"foreignKey:compliance_operator_scan_configuration_v2_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

// TableName returns the table name for GORM.
func (ComplianceOperatorScanConfigurationV2Notifiers) TableName() string {
	return "compliance_operator_scan_configuration_v2_notifiers"
}
