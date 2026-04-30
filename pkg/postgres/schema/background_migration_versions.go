package schema

import (
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

var (
	// CreateTableBackgroundMigrationVersionsStmt holds the create statement for table `background_migration_versions`.
	CreateTableBackgroundMigrationVersionsStmt = &postgres.CreateStmts{
		GormModel: (*BackgroundMigrationVersion)(nil),
		Children:  []*postgres.CreateStmts{},
	}

	// BackgroundMigrationVersionsSchema is the go schema for table `background_migration_versions`.
	BackgroundMigrationVersionsSchema = func() *walker.Schema {
		schema := GetSchemaForTable("background_migration_versions")
		if schema != nil {
			return schema
		}
		schema = &walker.Schema{
			Table: "background_migration_versions",
			Type:  "BackgroundMigrationVersion",
		}
		RegisterTable(schema, CreateTableBackgroundMigrationVersionsStmt)
		return schema
	}()
)

const (
	// BackgroundMigrationVersionsTableName specifies the name of the table in postgres.
	BackgroundMigrationVersionsTableName = "background_migration_versions"
)

// BackgroundMigrationVersion holds the Gorm model for Postgres table `background_migration_versions`.
type BackgroundMigrationVersion struct {
	SeqNum      int32  `gorm:"column:seqnum;type:integer;primaryKey"`
	OverrideTag string `gorm:"column:override_tag;type:text;default:''"`
}
