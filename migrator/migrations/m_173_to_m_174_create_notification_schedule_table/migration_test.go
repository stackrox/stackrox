//go:build sql_integration

package m173tom174

import (
	"context"
	"testing"

	notificationschedulestore "github.com/stackrox/rox/migrator/migrations/m_173_to_m_174_create_notification_schedule_table/notificationschedulestore"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
)

func TestMigration(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	pgTest := pghelper.ForT(t, false)
	assert.NotNil(t, pgTest)

	storage := notificationschedulestore.New(pgTest.DB)
	// Test get from the not-created-table returns an error
	_, _, errPre := storage.Get(ctx)
	assert.ErrorContains(t, errPre, "relation \"notification_schedules\" does not exist")

	// Create the table
	assert.NoError(t, createNotificationScheduleTable(pgTest.DB, pgTest.GetGormDB()))

	// Test get from the created table returns nil
	objPost, foundPost, errPost := storage.Get(ctx)
	assert.NoError(t, errPost)
	assert.False(t, foundPost)
	assert.Nil(t, objPost)
}
