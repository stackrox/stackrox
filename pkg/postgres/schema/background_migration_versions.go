package schema

import (
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableBackgroundMigrationVersionsStmt holds the create statement for table `background_migration_versions`.
	CreateTableBackgroundMigrationVersionsStmt = &postgres.CreateStmts{
		GormModel: (*BackgroundMigrationVersion)(nil),
		Children:  []*postgres.CreateStmts{},
	}
)

const (
	// BackgroundMigrationVersionsTableName specifies the name of the table in postgres.
	BackgroundMigrationVersionsTableName = "background_migration_version"
)

// BackgroundMigrationVersion holds the Gorm model for Postgres table `background_migration_version`.
type BackgroundMigrationVersion struct {
	SeqNum      int32  `gorm:"column:seqnum;type:integer;primaryKey;not null"`
	OverrideTag string `gorm:"column:override_tag;type:text;default:''"`
}
