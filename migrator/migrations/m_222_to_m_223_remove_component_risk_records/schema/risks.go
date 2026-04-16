package schema

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableRisksStmt holds the create statement for table `risks`.
	CreateTableRisksStmt = &postgres.CreateStmts{
		GormModel: (*Risks)(nil),
		Children:  []*postgres.CreateStmts{},
	}
)

const (
	// RisksTableName specifies the name of the table in postgres.
	RisksTableName = "risks"
)

// Risks holds the Gorm model for Postgres table `risks`.
type Risks struct {
	ID               string                  `gorm:"column:id;type:varchar;primaryKey"`
	SubjectNamespace string                  `gorm:"column:subject_namespace;type:varchar;index:risks_sac_filter,type:btree"`
	SubjectClusterID string                  `gorm:"column:subject_clusterid;type:uuid;index:risks_sac_filter,type:btree"`
	SubjectType      storage.RiskSubjectType `gorm:"column:subject_type;type:integer"`
	Score            float32                 `gorm:"column:score;type:numeric"`
	Serialized       []byte                  `gorm:"column:serialized;type:bytea"`
}
