// Frozen pre-PR#21423 schema copied from release-4.11.

package schema

import (
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableBaseImageRepositoriesStmt holds the create statement for table `base_image_repositories`.
	CreateTableBaseImageRepositoriesStmt = &postgres.CreateStmts{
		GormModel: (*BaseImageRepositories)(nil),
		Children:  []*postgres.CreateStmts{},
	}
)

const (
	// BaseImageRepositoriesTableName specifies the name of the table in postgres.
	BaseImageRepositoriesTableName = "base_image_repositories"
)

// BaseImageRepositories holds the Gorm model for Postgres table `base_image_repositories`.
type BaseImageRepositories struct {
	ID             string `gorm:"column:id;type:uuid;primaryKey"`
	RepositoryPath string `gorm:"column:repositorypath;type:varchar;unique"`
	CreatedByID    string `gorm:"column:createdby_id;type:varchar"`
	CreatedByName  string `gorm:"column:createdby_name;type:varchar"`
	Serialized     []byte `gorm:"column:serialized;type:bytea"`
}
