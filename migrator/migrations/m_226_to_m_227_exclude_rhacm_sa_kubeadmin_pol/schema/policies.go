package schema

import (
	"time"

	"github.com/lib/pq"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

// CreateTablePoliciesStmt holds the create statement for table `policies`.
var CreateTablePoliciesStmt = &postgres.CreateStmts{
	GormModel: (*Policies)(nil),
	Children:  []*postgres.CreateStmts{},
}

// Policies holds the Gorm model for Postgres table `policies`.
type Policies struct {
	ID                 string           `gorm:"column:id;type:varchar;primaryKey"`
	Name               string           `gorm:"column:name;type:varchar;unique"`
	Description        string           `gorm:"column:description;type:varchar"`
	Disabled           bool             `gorm:"column:disabled;type:bool"`
	Categories         *pq.StringArray  `gorm:"column:categories;type:text[]"`
	LifecycleStages    *pq.Int32Array   `gorm:"column:lifecyclestages;type:int[]"`
	Severity           storage.Severity `gorm:"column:severity;type:integer"`
	EnforcementActions *pq.Int32Array   `gorm:"column:enforcementactions;type:int[]"`
	LastUpdated        *time.Time       `gorm:"column:lastupdated;type:timestamp"`
	SORTName           string           `gorm:"column:sortname;type:varchar"`
	SORTLifecycleStage string           `gorm:"column:sortlifecyclestage;type:varchar"`
	SORTEnforcement    bool             `gorm:"column:sortenforcement;type:bool"`
	Serialized         []byte           `gorm:"column:serialized;type:bytea"`
}
