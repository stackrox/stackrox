// Frozen pre-PR#21423 schema copied from release-4.11.

package schema

import (
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableComplianceOperatorScanConfigurationV2Stmt holds the create statement for table `compliance_operator_scan_configuration_v2`.
	CreateTableComplianceOperatorScanConfigurationV2Stmt = &postgres.CreateStmts{
		GormModel: (*ComplianceOperatorScanConfigurationV2)(nil),
		Children: []*postgres.CreateStmts{
			&postgres.CreateStmts{
				GormModel: (*ComplianceOperatorScanConfigurationV2Profiles)(nil),
				Children:  []*postgres.CreateStmts{},
			},
			&postgres.CreateStmts{
				GormModel: (*ComplianceOperatorScanConfigurationV2Clusters)(nil),
				Children:  []*postgres.CreateStmts{},
			},
			&postgres.CreateStmts{
				GormModel: (*ComplianceOperatorScanConfigurationV2Notifiers)(nil),
				Children:  []*postgres.CreateStmts{},
			},
		},
	}
)

const (
	// ComplianceOperatorScanConfigurationV2TableName specifies the name of the table in postgres.
	ComplianceOperatorScanConfigurationV2TableName = "compliance_operator_scan_configuration_v2"
	// ComplianceOperatorScanConfigurationV2ProfilesTableName specifies the name of the table in postgres.
	ComplianceOperatorScanConfigurationV2ProfilesTableName = "compliance_operator_scan_configuration_v2_profiles"
	// ComplianceOperatorScanConfigurationV2ClustersTableName specifies the name of the table in postgres.
	ComplianceOperatorScanConfigurationV2ClustersTableName = "compliance_operator_scan_configuration_v2_clusters"
	// ComplianceOperatorScanConfigurationV2NotifiersTableName specifies the name of the table in postgres.
	ComplianceOperatorScanConfigurationV2NotifiersTableName = "compliance_operator_scan_configuration_v2_notifiers"
)

// ComplianceOperatorScanConfigurationV2 holds the Gorm model for Postgres table `compliance_operator_scan_configuration_v2`.
type ComplianceOperatorScanConfigurationV2 struct {
	ID             string `gorm:"column:id;type:uuid;primaryKey"`
	ScanConfigName string `gorm:"column:scanconfigname;type:varchar;unique"`
	ModifiedByName string `gorm:"column:modifiedby_name;type:varchar"`
	Serialized     []byte `gorm:"column:serialized;type:bytea"`
}

// ComplianceOperatorScanConfigurationV2Profiles holds the Gorm model for Postgres table `compliance_operator_scan_configuration_v2_profiles`.
type ComplianceOperatorScanConfigurationV2Profiles struct {
	ComplianceOperatorScanConfigurationV2ID  string                                `gorm:"column:compliance_operator_scan_configuration_v2_id;type:uuid;primaryKey"`
	Idx                                      int                                   `gorm:"column:idx;type:integer;primaryKey;index:complianceoperatorscanconfigurationv2profiles_idx,type:btree"`
	ProfileName                              string                                `gorm:"column:profilename;type:varchar"`
	ComplianceOperatorScanConfigurationV2Ref ComplianceOperatorScanConfigurationV2 `gorm:"foreignKey:compliance_operator_scan_configuration_v2_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

// ComplianceOperatorScanConfigurationV2Clusters holds the Gorm model for Postgres table `compliance_operator_scan_configuration_v2_clusters`.
type ComplianceOperatorScanConfigurationV2Clusters struct {
	ComplianceOperatorScanConfigurationV2ID  string                                `gorm:"column:compliance_operator_scan_configuration_v2_id;type:uuid;primaryKey"`
	Idx                                      int                                   `gorm:"column:idx;type:integer;primaryKey;index:complianceoperatorscanconfigurationv2clusters_idx,type:btree"`
	ClusterID                                string                                `gorm:"column:clusterid;type:uuid;index:complianceoperatorscanconfigurationv2clusters_sac_filter,type:hash"`
	ComplianceOperatorScanConfigurationV2Ref ComplianceOperatorScanConfigurationV2 `gorm:"foreignKey:compliance_operator_scan_configuration_v2_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

// ComplianceOperatorScanConfigurationV2Notifiers holds the Gorm model for Postgres table `compliance_operator_scan_configuration_v2_notifiers`.
type ComplianceOperatorScanConfigurationV2Notifiers struct {
	ComplianceOperatorScanConfigurationV2ID  string                                `gorm:"column:compliance_operator_scan_configuration_v2_id;type:uuid;primaryKey"`
	Idx                                      int                                   `gorm:"column:idx;type:integer;primaryKey;index:complianceoperatorscanconfigurationv2notifiers_idx,type:btree"`
	ID                                       string                                `gorm:"column:id;type:varchar"`
	ComplianceOperatorScanConfigurationV2Ref ComplianceOperatorScanConfigurationV2 `gorm:"foreignKey:compliance_operator_scan_configuration_v2_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}
