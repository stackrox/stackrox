//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/administration/events/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/administration/events/testutils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGather(t *testing.T) {
	pool := pgtest.ForT(t)
	t.Cleanup(func() {
		pool.Teardown(t)
	})
	store := pgStore.New(pool.DB)
	ds := &datastoreImpl{store: store}
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Integration),
		),
	)
	infoEvent := storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_INFO
	warningEvent := storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_WARNING
	errorEvent := storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR

	testCases := map[string]struct {
		administrativeEvents  []*events.AdministrationEvent
		expectedTotalEvents   int
		expectedInfoEvents    int
		expectedWarningEvents int
		expectedErrorEvents   int
	}{
		"one info event": {
			administrativeEvents: []*events.AdministrationEvent{
				testutils.GenerateAdministrativeEvent(infoEvent),
			},
			expectedTotalEvents: 1,
			expectedInfoEvents:  1,
		},
		"one warning event": {
			administrativeEvents: []*events.AdministrationEvent{
				testutils.GenerateAdministrativeEvent(warningEvent),
			},
			expectedTotalEvents:   1,
			expectedWarningEvents: 1,
		},
		"one error event": {
			administrativeEvents: []*events.AdministrationEvent{
				testutils.GenerateAdministrativeEvent(errorEvent),
			},
			expectedTotalEvents: 1,
			expectedErrorEvents: 1,
		},
		"one of each event": {
			administrativeEvents: []*events.AdministrationEvent{
				testutils.GenerateAdministrativeEvent(infoEvent),
				testutils.GenerateAdministrativeEvent(warningEvent),
				testutils.GenerateAdministrativeEvent(errorEvent),
			},
			expectedTotalEvents:   3,
			expectedInfoEvents:    1,
			expectedWarningEvents: 1,
			expectedErrorEvents:   1,
		},
		"3 errors 2 warning events": {
			administrativeEvents: []*events.AdministrationEvent{
				testutils.GenerateAdministrativeEvent(warningEvent),
				testutils.GenerateAdministrativeEvent(warningEvent),
				testutils.GenerateAdministrativeEvent(errorEvent),
				testutils.GenerateAdministrativeEvent(errorEvent),
				testutils.GenerateAdministrativeEvent(errorEvent),
			},
			expectedTotalEvents:   5,
			expectedWarningEvents: 2,
			expectedErrorEvents:   3,
		},
		"no event": {},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			for _, event := range tc.administrativeEvents {
				err := ds.AddEvent(ctx, event)
				require.NoError(t, err)
			}

			props, err := Gather(ds)(ctx)
			require.NoError(t, err)

			expectedProps := map[string]any{
				"Total administrative events":        tc.expectedTotalEvents,
				"Info type administrative events":    tc.expectedInfoEvents,
				"Warning type administrative events": tc.expectedWarningEvents,
				"Error type administrative events":   tc.expectedErrorEvents,
			}
			assert.Equal(t, expectedProps, props)

			for _, event := range tc.administrativeEvents {
				err := store.Delete(ctx, event.GetResourceID())
				require.NoError(t, err)
			}
		})
	}
}
