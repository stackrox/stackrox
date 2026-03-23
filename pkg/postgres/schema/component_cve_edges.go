package schema

import (
	"time"

	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableComponentCveEdgesStmt holds the create statement for table `component_cve_edges`.
	CreateTableComponentCveEdgesStmt = &postgres.CreateStmts{
		GormModel: (*ComponentCveEdges)(nil),
		Children:  []*postgres.CreateStmts{},
	}
)

const (
	// ComponentCveEdgesTableName specifies the name of the table in postgres.
	ComponentCveEdgesTableName = "component_cve_edges"
)

// ComponentCveEdges holds the Gorm model for Postgres table `component_cve_edges`.
type ComponentCveEdges struct {
	ID                    string           `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
	ComponentID           string           `gorm:"column:component_id;type:text;not null"`
	CveID                 string           `gorm:"column:cve_id;type:uuid;not null"`
	IsFixable             bool             `gorm:"column:is_fixable;type:bool;not null;default:false"`
	FixedBy               *string          `gorm:"column:fixed_by;type:text"`
	State                 string           `gorm:"column:state;type:text;not null;default:'OBSERVED'"`
	FirstSystemOccurrence time.Time        `gorm:"column:first_system_occurrence;type:timestamptz;not null;default:now()"`
	FixAvailableAt        *time.Time       `gorm:"column:fix_available_at;type:timestamptz"`
	ImageComponentV2Ref   ImageComponentV2 `gorm:"foreignKey:component_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
	CvesRef               Cves             `gorm:"foreignKey:cve_id;references:id;belongsTo"`
}
