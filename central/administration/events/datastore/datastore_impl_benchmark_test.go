//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/require"
)

var testCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
	sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
		sac.ResourceScopeKeys(resources.Administration),
	),
)

func BenchmarkDatastore_Add_and_Flush(b *testing.B) {
	pool := pgtest.ForT(b)
	b.Cleanup(func() {
		pool.Close()
	})
	datastore := GetTestPostgresDataStore(b, pool)

	benchmarkEvents, preExistingEvents := getEvents(1000)
	b.Run("add 1000 events to the writer and flush it", benchmarkDatastoreWithFlush(datastore, benchmarkEvents,
		preExistingEvents))

	benchmarkEvents, preExistingEvents = getEvents(5000)
	b.Run("add 5000 events to the writer and flush it", benchmarkDatastoreWithFlush(datastore, benchmarkEvents,
		preExistingEvents))

	benchmarkEvents, preExistingEvents = getEvents(10000)
	b.Run("add 10000 events to the writer and flush it", benchmarkDatastoreWithFlush(datastore, benchmarkEvents,
		preExistingEvents))

	benchmarkEvents, preExistingEvents = getEvents(20000)
	b.Run("add 20000 events to the writer and flush it", benchmarkDatastoreWithFlush(datastore, benchmarkEvents,
		preExistingEvents))
}

// benchmarkDatastoreWithFlush does an explicit Flush call after adding all events, in addition to the implicit flush
// calls done by the writer once the buffer is full.
func benchmarkDatastoreWithFlush(datastore DataStore, benchmarkEvents []*events.AdministrationEvent,
	preExistingEvents []*events.AdministrationEvent) func(b *testing.B) {
	return func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Need to ensure the database is in its initial state:
			// - no pre-existing events are within the database.
			// - add the custom events that should exist beforehand.
			// Stopping / Starting the timer in-between, as this would otherwise contribute to false benchmark timings.
			// Also need to do this for each benchmark run, to ensure every run has the same initial conditions.
			b.StopTimer()
			resetDatabase(b, datastore, preExistingEvents)
			b.StartTimer()

			for _, evt := range benchmarkEvents {
				if err := datastore.AddEvent(testCtx, evt); err != nil {
					b.Error(err)
				}
			}
			if err := datastore.Flush(testCtx); err != nil {
				b.Error(err)
			}
		}
	}
}

func resetDatabase(b *testing.B, datastore DataStore, preExistingEvents []*events.AdministrationEvent) {
	// Flush once just to be sure.
	require.NoError(b, datastore.Flush(testCtx))

	existingEvents, err := datastore.ListEvents(testCtx, search.EmptyQuery())
	require.NoError(b, err)

	require.NoError(b, RemoveTestEvents(testCtx, b, datastore, protoutils.GetIDs(existingEvents)...))

	for _, event := range preExistingEvents {
		require.NoError(b, datastore.AddEvent(testCtx, event))
	}
	require.NoError(b, datastore.Flush(testCtx))
}

func getEvents(numOfEvents int) ([]*events.AdministrationEvent, []*events.AdministrationEvent) {
	fixtureEvents := fixtures.GetMultipleAdministrationEvents(numOfEvents)

	preExistingEventsLength := numOfEvents / 3 / 2
	uniqueEventsLength := numOfEvents / 3
	preExistingEvents := make([]*events.AdministrationEvent, 0, preExistingEventsLength)
	benchmarkEvents := make([]*events.AdministrationEvent, 0, numOfEvents)

	// A third of the events will be duplicated within the input, which should lead to deduplication within the writer's
	// buffer.
	benchmarkEvents = append(benchmarkEvents, fixtureEvents[0:uniqueEventsLength]...)
	benchmarkEvents = append(benchmarkEvents, fixtureEvents[0:uniqueEventsLength]...)
	// A third of the events will be unique within the input.
	benchmarkEvents = append(benchmarkEvents, fixtureEvents[uniqueEventsLength*2:numOfEvents]...)
	// A couple of the unique events should be pre-populated within the database, which should lead to deduplication
	// within the writer's buffer.
	preExistingEvents = append(preExistingEvents, fixtureEvents[uniqueEventsLength*2:numOfEvents-preExistingEventsLength]...)

	return benchmarkEvents, preExistingEvents
}
