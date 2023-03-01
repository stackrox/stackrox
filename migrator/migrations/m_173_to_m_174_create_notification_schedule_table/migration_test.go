//go:build sql_integration

package m173tom174

import (
	"context"
	"testing"

	notificationschedulestore "github.com/stackrox/rox/migrator/migrations/m_173_to_m_174_create_notification_schedule_table/notificationschedulestore"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/assert"
)

func TestMigration(t *testing.T) {
	ctx := sac.WithAllAccess(context.Background())
	pgTest := pgtest.ForT(t)
	assert.NotNil(t, pgTest)

	storage := notificationschedulestore.New(pgTest.DB)
	// Test get from the not-created-table returns an error
	_, _, errPre := storage.Get(ctx)
	assert.ErrorIs(t, errPre, "table notification_schedules does not exist")

	// Create the table
	createNotificationScheduleTable(pgTest.DB, pgTest.GetGormDB(t))

	// Test get from the created table returns nil
	objPost, foundPost, errPost := storage.Get(ctx)
	assert.NoError(t, errPost)
	assert.False(foundPost)
	assert.Nil(t, objPost)
}
