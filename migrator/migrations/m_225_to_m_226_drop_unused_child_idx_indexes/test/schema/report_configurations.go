// Frozen pre-PR#21423 schema copied from release-4.11.

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
			&postgres.CreateStmts{
				GormModel: (*ReportConfigurationsNotifiers)(nil),
				Children:  []*postgres.CreateStmts{},
			},
		},
	}
)

const (
	// ReportConfigurationsTableName specifies the name of the table in postgres.
	ReportConfigurationsTableName = "report_configurations"
	// ReportConfigurationsNotifiersTableName specifies the name of the table in postgres.
	ReportConfigurationsNotifiersTableName = "report_configurations_notifiers"
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

// ReportConfigurationsNotifiers holds the Gorm model for Postgres table `report_configurations_notifiers`.
type ReportConfigurationsNotifiers struct {
	ReportConfigurationsID  string               `gorm:"column:report_configurations_id;type:varchar;primaryKey"`
	Idx                     int                  `gorm:"column:idx;type:integer;primaryKey;index:reportconfigurationsnotifiers_idx,type:btree"`
	ID                      string               `gorm:"column:id;type:varchar"`
	ReportConfigurationsRef ReportConfigurations `gorm:"foreignKey:report_configurations_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}
