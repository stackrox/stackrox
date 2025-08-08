package schema

import (
	"reflect"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/search/postgres/mapping"
)

const (
	// BaseImagesTableName specifies the name of the table in postgres.
	BaseImagesTableName = "base_images"
	// BaseImageLayerTableName specifies the name of the table in postgres.
	BaseImageLayerTableName = "base_image_layer"
)

// ====================================================================
// Table Creation and Schema Definitions
// ====================================================================

var (
	// CreateTableBaseImagesStmt holds the create statement for table `base_images`.
	// It also includes the nested child table `base_image_layer` to handle the
	// foreign key relationship during table creation.
	CreateTableBaseImagesStmt = &postgres.CreateStmts{
		GormModel: (*BaseImages)(nil),
		Children: []*postgres.CreateStmts{
			{
				GormModel: (*BaseImageLayer)(nil),
				Children:  []*postgres.CreateStmts{},
			},
		},
	}

	// BaseImagesSchema is the go schema for table `base_images`.
	// This function demonstrates how the schema would be registered, similar to your example.
	// The `walker` and `search` packages are assumed to be part of your codebase.
	BaseImagesSchema = func() *walker.Schema {
		schema := GetSchemaForTable(BaseImagesTableName)
		if schema != nil {
			return schema
		}

		// This is a placeholder for your schema walking logic.
		schema = walker.Walk(reflect.TypeOf((*BaseImages)(nil)), BaseImagesTableName)

		RegisterTable(schema, CreateTableBaseImagesStmt)
		mapping.RegisterCategoryToTable(v1.SearchCategory_BASE_IMAGES, schema)
		return schema
	}()
)

// Dummy functions to satisfy the dependencies for the example.
// In a real codebase, these would be implemented in your project's packages.
// func GetSchemaForTable(tableName string) *walker.Schema                         { return nil }
// func RegisterTable(schema *walker.Schema, createStmts *postgres.CreateStmts)    {}
// func RegisterCategoryToTable(category v1.SearchCategory, schema *walker.Schema) {}
// ====================================================================
// GORM Model Structs
// ====================================================================

// BaseImages holds the Gorm model for Postgres table `base_images`.
// This struct maps directly to your DDL for the `base_images` table.
type BaseImages struct {
	// GORM's `gorm.Model` provides ID, CreatedAt, UpdatedAt, and DeletedAt fields.
	// Since your DDL uses a custom BIGSERIAL 'id' and 'created_at', we will
	// define them explicitly instead of embedding `gorm.Model`.
	ID           int64      `gorm:"column:id;type:bigserial;primaryKey"`
	Registry     string     `gorm:"column:registry;type:text;not null;uniqueIndex:idx_base_images_uniq"`
	Repository   string     `gorm:"column:repository;type:text;not null;uniqueIndex:idx_base_images_uniq"`
	Tag          *string    `gorm:"column:tag;type:text;uniqueIndex:idx_base_images_uniq"`
	Digest       *string    `gorm:"column:digest;type:text;uniqueIndex:idx_base_images_uniq"`
	ConfigDigest *string    `gorm:"column:config_digest;type:text"`
	CreatedAt    *time.Time `gorm:"column:created_at;type:timestamp;default:now()"`
	Active       bool       `gorm:"column:active;type:boolean"`

	// Define the one-to-many relationship to the layers table.
	// GORM will use the `ID` from this struct as the foreign key in the `BaseImageLayer` struct.
	Layers []*BaseImageLayer `gorm:"foreignKey:Iid;references:id"`
}

// BaseImageLayer holds the Gorm model for Postgres table `base_image_layer`.
// This struct maps directly to your DDL for the `base_image_layer` table.
type BaseImageLayer struct {
	ID        int64  `gorm:"column:id;type:bigserial;primaryKey"`
	Iid       int64  `gorm:"column:iid;type:bigint;not null;uniqueIndex:idx_base_image_layer_uniq"`
	LayerHash string `gorm:"column:layer_hash;type:text;not null"`
	Level     int32  `gorm:"column:level;type:integer;not null;uniqueIndex:idx_base_image_layer_uniq"`

	// Define the foreign key relationship back to the `BaseImages` parent.
	// This helps GORM manage the relationship, including the `ON DELETE CASCADE` constraint.
	BaseImage *BaseImages `gorm:"foreignKey:Iid;references:ID;constraint:OnDelete:CASCADE"`
}
