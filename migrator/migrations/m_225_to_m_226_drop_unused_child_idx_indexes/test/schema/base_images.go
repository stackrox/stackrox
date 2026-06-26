// Frozen pre-migration GORM schema for base_images.
// Reproduces old index tags so AutoMigrate creates the _idx indexes that the migration drops.

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
			{GormModel: (*BaseImagesLayers)(nil), Children: []*postgres.CreateStmts{}},
		},
	}
)

// BaseImages holds the Gorm model for Postgres table `base_images`.
type BaseImages struct {
	ID                    string     `gorm:"column:id;type:uuid;primaryKey"`
	BaseImageRepositoryID string     `gorm:"column:baseimagerepositoryid;type:varchar"`
	Repository            string     `gorm:"column:repository;type:varchar"`
	Tag                   string     `gorm:"column:tag;type:varchar"`
	ManifestDigest        string     `gorm:"column:manifestdigest;type:varchar"`
	DiscoveredAt          *time.Time `gorm:"column:discoveredat;type:timestamp"`
	Active                bool       `gorm:"column:active;type:bool"`
	FirstLayerDigest      string     `gorm:"column:firstlayerdigest;type:varchar;index:baseimages_firstlayerdigest,type:btree"`
	Serialized            []byte     `gorm:"column:serialized;type:bytea"`
}

// TableName returns the table name for GORM.
func (BaseImages) TableName() string { return "base_images" }

// BaseImagesLayers holds the Gorm model for Postgres table `base_images_layers`.
type BaseImagesLayers struct {
	BaseImagesID  string     `gorm:"column:base_images_id;type:uuid;primaryKey"`
	Idx           int        `gorm:"column:idx;type:integer;primaryKey;index:baseimageslayers_idx,type:btree"`
	LayerDigest   string     `gorm:"column:layerdigest;type:varchar;uniqueIndex:base_image_id_layer"`
	Index         int32      `gorm:"column:index;type:integer"`
	BaseImagesRef BaseImages `gorm:"foreignKey:base_images_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

// TableName returns the table name for GORM.
func (BaseImagesLayers) TableName() string { return "base_images_layers" }
