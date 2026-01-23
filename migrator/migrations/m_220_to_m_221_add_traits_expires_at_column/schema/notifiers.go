package schema

import (
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableNotifiersStmt holds the create statement for table `notifiers`.
	CreateTableNotifiersStmt = &postgres.CreateStmts{
		GormModel: (*Notifiers)(nil),
		Children:  []*postgres.CreateStmts{},
	}
)

// Notifiers holds the Gorm model for Postgres table `notifiers`.
type Notifiers struct {
	ID         string `gorm:"column:id;type:varchar;primaryKey"`
	Name       string `gorm:"column:name;type:varchar;unique"`
	Serialized []byte `gorm:"column:serialized;type:bytea"`
}
