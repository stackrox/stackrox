package m223tom224

import (
	"context"

	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())
	db := database.GormDB.WithContext(ctx)

	if err := db.AutoMigrate(&schema.BackgroundMigrationVersion{}); err != nil {
		return err
	}

	// Seed with initial row if empty.
	var count int64
	if err := db.Table(schema.BackgroundMigrationVersionsTableName).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return db.Table(schema.BackgroundMigrationVersionsTableName).Create(&schema.BackgroundMigrationVersion{
			SeqNum: 0,
		}).Error
	}
	return nil
}
