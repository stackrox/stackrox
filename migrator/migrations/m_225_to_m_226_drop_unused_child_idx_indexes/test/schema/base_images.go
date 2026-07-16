// Frozen pre-PR#21423 schema copied from release-4.11.

package schema

import (
	"time"

	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableBaseImagesStmt holds the create statement for table `base_images`.
	CreateTableBaseImagesStmt = &postgres.CreateStmts{
		GormModel: (*BaseImages)(nil),
		Children: []*postgres.CreateStmts{
			{
				GormModel: (*BaseImagesLayers)(nil),
				Children:  []*postgres.CreateStmts{},
			},
		},
	}
)

const (
	// BaseImagesTableName specifies the name of the table in postgres.
	BaseImagesTableName = "base_images"
	// BaseImagesLayersTableName specifies the name of the table in postgres.
	BaseImagesLayersTableName = "base_images_layers"
)

// BaseImages holds the Gorm model for Postgres table `base_images`.
type BaseImages struct {
	ID                       string                `gorm:"column:id;type:uuid;primaryKey"`
	BaseImageRepositoryID    string                `gorm:"column:baseimagerepositoryid;type:varchar"`
	Repository               string                `gorm:"column:repository;type:varchar"`
	Tag                      string                `gorm:"column:tag;type:varchar"`
	ManifestDigest           string                `gorm:"column:manifestdigest;type:varchar"`
	DiscoveredAt             *time.Time            `gorm:"column:discoveredat;type:timestamp"`
	Active                   bool                  `gorm:"column:active;type:bool"`
	FirstLayerDigest         string                `gorm:"column:firstlayerdigest;type:varchar;index:baseimages_firstlayerdigest,type:btree"`
	Serialized               []byte                `gorm:"column:serialized;type:bytea"`
	BaseImageRepositoriesRef BaseImageRepositories `gorm:"foreignKey:baseimagerepositoryid;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

// BaseImagesLayers holds the Gorm model for Postgres table `base_images_layers`.
type BaseImagesLayers struct {
	BaseImagesID  string     `gorm:"column:base_images_id;type:uuid;primaryKey"`
	Idx           int        `gorm:"column:idx;type:integer;primaryKey;index:baseimageslayers_idx,type:btree"`
	LayerDigest   string     `gorm:"column:layerdigest;type:varchar;uniqueIndex:base_image_id_layer"`
	Index         int32      `gorm:"column:index;type:integer"`
	BaseImagesRef BaseImages `gorm:"foreignKey:base_images_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}
