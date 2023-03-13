//go:build sql_integration

package m175tom176

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	notificationschedulestore "github.com/stackrox/rox/migrator/migrations/m_175_to_m_176_create_notification_schedule_table/notificationschedulestore"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigration(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	pgTest := pghelper.ForT(t, false)
	require.NotNil(t, pgTest)

	scheduleStore := notificationschedulestore.New(pgTest.DB)
	// Test get from the not-created-table returns an error
	_, _, errPre := scheduleStore.Get(ctx)
	assert.ErrorContains(t, errPre, "relation \"notification_schedules\" does not exist")

	// Create the table
	assert.NoError(t, createNotificationScheduleTable(pgTest.DB, pgTest.GetGormDB()))

	// Test get from the created table returns nil
	objPost, foundPost, errPost := scheduleStore.Get(ctx)
	assert.NoError(t, errPost)
	assert.False(t, foundPost)
	assert.Nil(t, objPost)

	// Test upsert is possible
	scheduleToInsert := &storage.NotificationSchedule{
		LastRun: &types.Timestamp{
			Seconds: 1234567890,
			Nanos:   0,
		},
	}
	upsertErr := scheduleStore.Upsert(ctx, scheduleToInsert)
	assert.NoError(t, upsertErr)

	// Test get now retrieves the upserted item
	fetchedSchedule, found, fetchErr := scheduleStore.Get(ctx)
	assert.NoError(t, fetchErr)
	assert.True(t, found)
	assert.Equal(t, scheduleToInsert, fetchedSchedule)

	// Test Delete
	deleteErr := scheduleStore.Delete(ctx)
	assert.NoError(t, deleteErr)

	deletedSchedule, deletedFound, deletedRetrieveErr := scheduleStore.Get(ctx)
	assert.NoError(t, deletedRetrieveErr)
	assert.False(t, deletedFound)
	assert.Nil(t, deletedSchedule)
}
