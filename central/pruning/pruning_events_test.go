//go:build sql_integration

package pruning

import (
	"context"
	"testing"
	"time"

	administrationEventDS "github.com/stackrox/rox/central/administration/events/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdministrationEventsPruning(t *testing.T) {
	pool := pgtest.ForT(t)
	defer pool.Teardown(t)

	datastore := administrationEventDS.GetTestPostgresDataStore(t, pool)
	gc := garbageCollectorImpl{
		administrationEvents: datastore,
	}
	privateConfig := &storage.PrivateConfig{
		AdministrationEventsConfig: &storage.AdministrationEventsConfig{
			RetentionDurationDays: 4,
		},
	}
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration),
		),
	)

	events := []*storage.AdministrationEvent{
		{
			Id:             "cd118b6d-0b2e-5ab1-b1fc-c992d58eda9f",
			LastOccurredAt: timeBeforeDays(2),
		},
		{
			Id:             "460c8808-9f70-51e7-9f3a-973f44ab8595",
			LastOccurredAt: protoconv.ConvertTimeToTimestamp(time.Now()),
		},
		// 4 days ago should be subject to pruning (incl. a bit of a leeway).
		{
			Id:             "a10c6cae-c72f-58a3-bd86-dc0363990fe6",
			LastOccurredAt: protoconv.ConvertTimeToTimestamp(time.Now().Add(-(96*24*time.Hour + 30*time.Minute))),
		},
		{
			Id:             "5e2ab54d-0a19-5f31-9093-136d49b6bd94",
			LastOccurredAt: timeBeforeDays(3),
		},
		{
			Id:             "13d24bd2-1373-57b3-af07-066cdd65d226",
			LastOccurredAt: protoconv.ConvertTimeToTimestamp(time.Now().Add(4 * 24 * time.Hour)),
		},
		{
			Id:             "8e1876a3-a0c0-56c3-bccc-961d89f80220",
			LastOccurredAt: timeBeforeDays(12),
		},
		{
			Id:             "396ad8a4-1cd5-5c2d-9176-bd831c7cc0d7",
			LastOccurredAt: timeBeforeDays(365),
		},
	}
	require.NoError(t, administrationEventDS.UpsertTestEvents(ctx, t,
		datastore, events...))

	gc.removeExpiredAdministrationEvents(privateConfig)

	storedEvents, err := datastore.ListEvents(ctx, search.EmptyQuery())
	assert.NoError(t, err)

	assert.ElementsMatch(t, []*storage.AdministrationEvent{events[0], events[1], events[3], events[4]}, storedEvents)
}
