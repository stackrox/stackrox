// Frozen pre-migration GORM schema for report_configurations.
// Reproduces old index tags so AutoMigrate creates the _idx indexes that the migration drops.

package schema

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableReportConfigurationsStmt holds the create statement for table `report_configurations`.
	CreateTableReportConfigurationsStmt = &postgres.CreateStmts{
		GormModel: (*ReportConfigurations)(nil),
		Children: []*postgres.CreateStmts{
			{GormModel: (*ReportConfigurationsNotifiers)(nil), Children: []*postgres.CreateStmts{}},
		},
	}
)

// ReportConfigurations holds the Gorm model for Postgres table `report_configurations`.
type ReportConfigurations struct {
	ID                        string                                 `gorm:"column:id;type:varchar;primaryKey"`
	Name                      string                                 `gorm:"column:name;type:varchar"`
	Type                      storage.ReportConfiguration_ReportType `gorm:"column:type;type:integer"`
	ScopeID                   string                                 `gorm:"column:scopeid;type:varchar"`
	ResourceScopeCollectionID string                                 `gorm:"column:resourcescope_collectionid;type:varchar"`
	CreatorName               string                                 `gorm:"column:creator_name;type:varchar"`
	Serialized                []byte                                 `gorm:"column:serialized;type:bytea"`
}

// TableName returns the table name for GORM.
func (ReportConfigurations) TableName() string { return "report_configurations" }

// ReportConfigurationsNotifiers holds the Gorm model for Postgres table `report_configurations_notifiers`.
type ReportConfigurationsNotifiers struct {
	ReportConfigurationsID  string               `gorm:"column:report_configurations_id;type:varchar;primaryKey"`
	Idx                     int                  `gorm:"column:idx;type:integer;primaryKey;index:reportconfigurationsnotifiers_idx,type:btree"`
	ID                      string               `gorm:"column:id;type:varchar"`
	ReportConfigurationsRef ReportConfigurations `gorm:"foreignKey:report_configurations_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

// TableName returns the table name for GORM.
func (ReportConfigurationsNotifiers) TableName() string {
	return "report_configurations_notifiers"
}
