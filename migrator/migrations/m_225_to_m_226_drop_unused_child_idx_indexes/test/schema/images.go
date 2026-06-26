// Frozen pre-migration GORM schema for images.
// Reproduces old index tags so AutoMigrate creates the _idx indexes that the migration drops.

package schema

import (
	"time"

	"github.com/lib/pq"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	// CreateTableImagesStmt holds the create statement for table `images`.
	CreateTableImagesStmt = &postgres.CreateStmts{
		GormModel: (*Images)(nil),
		Children: []*postgres.CreateStmts{
			{GormModel: (*ImagesLayers)(nil), Children: []*postgres.CreateStmts{}},
		},
	}
)

// Images holds the Gorm model for Postgres table `images`.
type Images struct {
	ID                   string            `gorm:"column:id;type:varchar;primaryKey"`
	NameRegistry         string            `gorm:"column:name_registry;type:varchar"`
	NameRemote           string            `gorm:"column:name_remote;type:varchar"`
	NameTag              string            `gorm:"column:name_tag;type:varchar"`
	NameFullName         string            `gorm:"column:name_fullname;type:varchar"`
	MetadataV1Created    *time.Time        `gorm:"column:metadata_v1_created;type:timestamp"`
	MetadataV1User       string            `gorm:"column:metadata_v1_user;type:varchar"`
	MetadataV1Command    *pq.StringArray   `gorm:"column:metadata_v1_command;type:text[]"`
	MetadataV1Entrypoint *pq.StringArray   `gorm:"column:metadata_v1_entrypoint;type:text[]"`
	MetadataV1Volumes    *pq.StringArray   `gorm:"column:metadata_v1_volumes;type:text[]"`
	MetadataV1Labels     map[string]string `gorm:"column:metadata_v1_labels;type:jsonb"`
	ScanScanTime         *time.Time        `gorm:"column:scan_scantime;type:timestamp"`
	ScanOperatingSystem  string            `gorm:"column:scan_operatingsystem;type:varchar"`
	SignatureFetched     *time.Time        `gorm:"column:signature_fetched;type:timestamp"`
	Components           int32             `gorm:"column:components;type:integer"`
	Cves                 int32             `gorm:"column:cves;type:integer"`
	FixableCves          int32             `gorm:"column:fixablecves;type:integer"`
	LastUpdated          *time.Time        `gorm:"column:lastupdated;type:timestamp"`
	Priority             int64             `gorm:"column:priority;type:bigint"`
	RiskScore            float32           `gorm:"column:riskscore;type:numeric"`
	TopCvss              float32           `gorm:"column:topcvss;type:numeric"`
	Serialized           []byte            `gorm:"column:serialized;type:bytea"`
}

// TableName returns the table name for GORM.
func (Images) TableName() string { return "images" }

// ImagesLayers holds the Gorm model for Postgres table `images_layers`.
type ImagesLayers struct {
	ImagesID    string `gorm:"column:images_id;type:varchar;primaryKey"`
	Idx         int    `gorm:"column:idx;type:integer;primaryKey;index:imageslayers_idx,type:btree"`
	Instruction string `gorm:"column:instruction;type:varchar"`
	Value       string `gorm:"column:value;type:varchar"`
	ImagesRef   Images `gorm:"foreignKey:images_id;references:id;belongsTo;constraint:OnDelete:CASCADE"`
}

// TableName returns the table name for GORM.
func (ImagesLayers) TableName() string { return "images_layers" }
