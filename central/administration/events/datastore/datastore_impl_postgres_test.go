//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/administration/events/datastore/internal/search"
	pgStore "github.com/stackrox/rox/central/administration/events/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/administration/events/datastore/internal/writer"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
)

func TestEventsDatastorePostgres(t *testing.T) {
	suite.Run(t, new(datastorePostgresTestSuite))
}

type datastorePostgresTestSuite struct {
	suite.Suite

	readCtx      context.Context
	writeCtx     context.Context
	postgresTest *pgtest.TestPostgres
	store        pgStore.Store
	datastore    DataStore
	writer       writer.Writer
}

func (s *datastorePostgresTestSuite) SetupTest() {
	s.readCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Administration),
		),
	)
	s.writeCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration),
		),
	)

	s.postgresTest = pgtest.ForT(s.T())
	s.Require().NotNil(s.postgresTest)
	searcher := search.New(pgStore.NewIndexer(s.postgresTest.DB))
	s.store = pgStore.New(s.postgresTest.DB)
	s.writer = writer.New(s.store)
	s.datastore = newDataStore(searcher, s.store, s.writer)
}

func (s *datastorePostgresTestSuite) TearDownTest() {
	s.postgresTest.Teardown(s.T())
	s.postgresTest.Close()
}

func (s *datastorePostgresTestSuite) assertEventsEqual(
	event *events.AdministrationEvent,
	storageEvent *storage.AdministrationEvent,
) {
	s.Equal(event.GetLevel(), storageEvent.GetLevel())
	s.Equal(event.GetMessage(), storageEvent.GetMessage())
	s.Equal(event.GetType(), storageEvent.GetType())
	s.Equal(event.GetHint(), storageEvent.GetHint())
	s.Equal(event.GetDomain(), storageEvent.GetDomain())
	s.Equal(event.GetResourceID(), storageEvent.GetResourceId())
	s.Equal(event.GetResourceType(), storageEvent.GetResourceType())
}

func (s *datastorePostgresTestSuite) TestUpsertEvent_Success() {
	event := fixtures.GetAdministrationEvent()

	err := s.datastore.AddEvent(s.writeCtx, event)
	s.Require().NoError(err)

	err = s.datastore.Flush(s.writeCtx)
	s.Require().NoError(err)

	id := events.GenerateEventID(event)
	dbEvent, err := s.datastore.GetEvent(s.readCtx, id)
	s.Require().NoError(err)
	s.assertEventsEqual(event, dbEvent)
	s.EqualValues(dbEvent.GetNumOccurrences(), 1)
}

func (s *datastorePostgresTestSuite) TestUpsertEvent_MultipleOccurrencesFlushOnce() {
	event := fixtures.GetAdministrationEvent()

	err := s.datastore.AddEvent(s.writeCtx, event)
	s.Require().NoError(err)

	err = s.datastore.AddEvent(s.writeCtx, event)
	s.Require().NoError(err)

	id := events.GenerateEventID(event)
	dbEvent, err := s.datastore.GetEvent(s.readCtx, id)
	s.Require().ErrorIs(err, errox.NotFound)
	s.Empty(dbEvent)

	err = s.datastore.Flush(s.writeCtx)
	s.Require().NoError(err)

	dbEvent, err = s.datastore.GetEvent(s.readCtx, id)
	s.Require().NoError(err)
	s.assertEventsEqual(event, dbEvent)
	s.EqualValues(dbEvent.GetNumOccurrences(), 2)
}

func (s *datastorePostgresTestSuite) TestUpsertEvent_MultipleOccurrencesFlushEach() {
	event := fixtures.GetAdministrationEvent()

	err := s.datastore.AddEvent(s.writeCtx, event)
	s.Require().NoError(err)

	err = s.datastore.Flush(s.writeCtx)
	s.Require().NoError(err)

	id := events.GenerateEventID(event)
	dbEvent, err := s.datastore.GetEvent(s.writeCtx, id)
	s.Require().NoError(err)
	s.assertEventsEqual(event, dbEvent)
	s.EqualValues(dbEvent.GetNumOccurrences(), 1)

	err = s.datastore.AddEvent(s.writeCtx, event)
	s.Require().NoError(err)

	err = s.datastore.Flush(s.writeCtx)
	s.Require().NoError(err)

	dbEvent, err = s.datastore.GetEvent(s.readCtx, id)
	s.Require().NoError(err)
	s.assertEventsEqual(event, dbEvent)
	s.EqualValues(dbEvent.GetNumOccurrences(), 2)
}

func (s *datastorePostgresTestSuite) TestUpsertEvent_NilEvent() {
	err := s.datastore.AddEvent(s.writeCtx, nil)
	s.ErrorIs(err, errox.InvalidArgs)
}

func (s *datastorePostgresTestSuite) TestFlushWithEmptyBuffer() {
	err := s.datastore.Flush(s.writeCtx)
	s.NoError(err)

	err = s.datastore.Flush(s.writeCtx)
	s.NoError(err)
}

func (s *datastorePostgresTestSuite) TestGetEvent() {
	nonExistingID := "0925514f-3a33-5931-b431-756406e1a008"

	administrationEvent := fixtures.GetAdministrationEvent()
	err := s.datastore.AddEvent(s.writeCtx, administrationEvent)
	s.Require().NoError(err)

	event, err := s.datastore.GetEvent(s.readCtx, nonExistingID)
	s.ErrorIs(err, errox.NotFound)
	s.Empty(event)

	s.Require().NoError(s.datastore.Flush(s.writeCtx))

	id := events.GenerateEventID(administrationEvent)
	event, err = s.datastore.GetEvent(s.readCtx, id)
	s.NoError(err)
	s.assertEventsEqual(administrationEvent, event)
}

func (s *datastorePostgresTestSuite) TestCountEvents() {
	count, err := s.datastore.CountEvents(s.readCtx, &v1.Query{})
	s.NoError(err)
	s.Zero(count)

	s.addEvents(100)

	count, err = s.datastore.CountEvents(s.readCtx, &v1.Query{})
	s.NoError(err)
	s.Equal(100, count)
}

func (s *datastorePostgresTestSuite) TestAddEvent_WriterBufferFull() {
	// Ensure that when adding more events than the writer's internal buffer, `AddEvent()` will not
	// return an error and the events are successfully added.
	s.addEvents(1001)

	count, err := s.datastore.CountEvents(s.readCtx, &v1.Query{})
	s.NoError(err)
	s.Equal(1001, count)
}

func (s *datastorePostgresTestSuite) addEvents(numOfEvents int) {
	administrationEvents := fixtures.GetMultipleAdministrationEvents(numOfEvents)
	for _, administrationEvent := range administrationEvents {
		s.Require().NoError(s.datastore.AddEvent(s.writeCtx, administrationEvent))
	}
	s.Require().NoError(s.datastore.Flush(s.writeCtx))
}
