// Frozen pre-PR#21423 schema copied from release-4.11.

package schema

import (
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableCollectionsStmt holds the create statement for table `collections`.
	CreateTableCollectionsStmt = &postgres.CreateStmts{
		GormModel: (*Collections)(nil),
		Children: []*postgres.CreateStmts{
			&postgres.CreateStmts{
				GormModel: (*CollectionsEmbeddedCollections)(nil),
				Children:  []*postgres.CreateStmts{},
			},
		},
	}
)

const (
	// CollectionsTableName specifies the name of the table in postgres.
	CollectionsTableName = "collections"
	// CollectionsEmbeddedCollectionsTableName specifies the name of the table in postgres.
	CollectionsEmbeddedCollectionsTableName = "collections_embedded_collections"
)

// Collections holds the Gorm model for Postgres table `collections`.
type Collections struct {
	ID            string `gorm:"column:id;type:varchar;primaryKey"`
	Name          string `gorm:"column:name;type:varchar;unique"`
	CreatedByName string `gorm:"column:createdby_name;type:varchar"`
	UpdatedByName string `gorm:"column:updatedby_name;type:varchar"`
	Serialized    []byte `gorm:"column:serialized;type:bytea"`
}

// CollectionsEmbeddedCollections holds the Gorm model for Postgres table `collections_embedded_collections`.
type CollectionsEmbeddedCollections struct {
	CollectionsID       string      `gorm:"column:collections_id;type:varchar;primaryKey"`
	Idx                 int         `gorm:"column:idx;type:integer;primaryKey;index:collectionsembeddedcollections_idx,type:btree"`
	ID                  string      `gorm:"column:id;type:varchar"`
	CollectionsRef      Collections `gorm:"foreignKey:collections_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
	CollectionsCycleRef Collections `gorm:"foreignKey:id;references:id;belongsTo;constraint:OnDelete:RESTRICT"`
}
