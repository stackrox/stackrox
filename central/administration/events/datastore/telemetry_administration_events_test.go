//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/administration/events/datastore/internal/search"
	pgStore "github.com/stackrox/rox/central/administration/events/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/administration/events/datastore/internal/writer"
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
	require.NotNil(t, pool)
	store := pgStore.New(pool)
	datastore := newDataStore(search.New(store), store, writer.New(store))
	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration),
		),
	)
	infoEvent := storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_INFO
	warningEvent := storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_WARNING
	errorEvent := storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR
	authDomain := events.AuthenticationDomain
	defaultDomain := events.DefaultDomain
	imageScanningDomain := events.ImageScanningDomain
	integrationDomain := events.IntegrationDomain

	testCases := map[string]struct {
		administrativeEvents              []*events.AdministrationEvent
		expectedTotalEvents               int
		expectedInfoEvents                int
		expectedWarningEvents             int
		expectedErrorEvents               int
		expectedAuthDomainEvents          int
		expectedDefaultDomainEvents       int
		expectedImageScanningDomainEvents int
		expectedIntegrationDomainEvents   int
	}{
		"one info event in default domain": {
			administrativeEvents: []*events.AdministrationEvent{
				testutils.GenerateAdministrativeEvent(infoEvent, defaultDomain),
			},
			expectedTotalEvents:         1,
			expectedInfoEvents:          1,
			expectedDefaultDomainEvents: 1,
		},
		"one warning event in image scanning": {
			administrativeEvents: []*events.AdministrationEvent{
				testutils.GenerateAdministrativeEvent(warningEvent, imageScanningDomain),
			},
			expectedTotalEvents:               1,
			expectedWarningEvents:             1,
			expectedImageScanningDomainEvents: 1,
		},
		"one error event in integrations": {
			administrativeEvents: []*events.AdministrationEvent{
				testutils.GenerateAdministrativeEvent(errorEvent, integrationDomain),
			},
			expectedTotalEvents:             1,
			expectedErrorEvents:             1,
			expectedIntegrationDomainEvents: 1,
		},
		"one of each": {
			administrativeEvents: []*events.AdministrationEvent{
				testutils.GenerateAdministrativeEvent(infoEvent, authDomain),
				testutils.GenerateAdministrativeEvent(warningEvent, imageScanningDomain),
				testutils.GenerateAdministrativeEvent(errorEvent, integrationDomain),
				testutils.GenerateAdministrativeEvent(errorEvent, defaultDomain),
			},
			expectedTotalEvents:               4,
			expectedInfoEvents:                1,
			expectedWarningEvents:             1,
			expectedErrorEvents:               2,
			expectedAuthDomainEvents:          1,
			expectedImageScanningDomainEvents: 1,
			expectedIntegrationDomainEvents:   1,
			expectedDefaultDomainEvents:       1,
		},
		"3 errors 2 warning events, all in default domain": {
			administrativeEvents: []*events.AdministrationEvent{
				testutils.GenerateAdministrativeEvent(warningEvent, defaultDomain),
				testutils.GenerateAdministrativeEvent(warningEvent, defaultDomain),
				testutils.GenerateAdministrativeEvent(errorEvent, defaultDomain),
				testutils.GenerateAdministrativeEvent(errorEvent, defaultDomain),
				testutils.GenerateAdministrativeEvent(errorEvent, defaultDomain),
			},
			expectedTotalEvents:         5,
			expectedWarningEvents:       2,
			expectedErrorEvents:         3,
			expectedDefaultDomainEvents: 5,
		},
		"no event": {},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			for _, event := range tc.administrativeEvents {
				err := datastore.AddEvent(ctx, event)
				require.NoError(t, err)
			}
			err := datastore.Flush(ctx)
			require.NoError(t, err)

			props, err := Gather(datastore)(ctx)
			require.NoError(t, err)

			expectedProps := map[string]any{
				"Total Error type Administrative Events":            tc.expectedErrorEvents,
				"Total Info type Administrative Events":             tc.expectedInfoEvents,
				"Total Administrative Events":                       tc.expectedTotalEvents,
				"Total Warning type Administrative Events":          tc.expectedWarningEvents,
				"Total Authentication domain Administrative Events": tc.expectedAuthDomainEvents,
				"Total Default domain Administrative Events":        tc.expectedDefaultDomainEvents,
				"Total Image Scanning domain Administrative Events": tc.expectedImageScanningDomainEvents,
				"Total Integration domain Administrative Events":    tc.expectedIntegrationDomainEvents,
			}
			assert.Equal(t, expectedProps, props)

			for _, event := range tc.administrativeEvents {
				err = store.Delete(ctx, events.GenerateEventID(event))
				require.NoError(t, err)
			}
		})
	}
}
