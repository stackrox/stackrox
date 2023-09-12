//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/administration/events/datastore/internal/search"
	pgStore "github.com/stackrox/rox/central/administration/events/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/administration/events/datastore/internal/writer"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
)

func TestEventsDatastorePostgres(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(datastorePostgresTestSuite))
}

type datastorePostgresTestSuite struct {
	suite.Suite

	readCtx      context.Context
	writeCtx     context.Context
	postgresTest *pgtest.TestPostgres
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
			sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration),
		),
	)

	s.postgresTest = pgtest.ForT(s.T())
	s.Require().NotNil(s.postgresTest)
	searcher := search.New(pgStore.NewIndexer(s.postgresTest.DB))
	store := pgStore.New(s.postgresTest.DB)
	s.writer = writer.New(store)
	s.datastore = NewDataStore(searcher, store, s.writer)
}

func (s *datastorePostgresTestSuite) TearDownTest() {
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
}

func (s *datastorePostgresTestSuite) TestUpsertEvent_Success() {
	event := &events.AdministrationEvent{
		Level:   storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message: "message",
		Type:    storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		Hint:    "hint",
		Domain:  "domain",
	}

	err := s.datastore.AddEvent(s.writeCtx, event)
	s.Require().NoError(err)

	err = s.datastore.Flush(s.writeCtx)
	s.Require().NoError(err)

	id := "73072ecb-2222-5922-8948-5944338861c8"
	dbEvent, err := s.datastore.GetEventByID(s.readCtx, id)
	s.Require().NoError(err)
	s.assertEventsEqual(event, dbEvent)
	s.EqualValues(dbEvent.GetNumOccurrences(), 1)
}

func (s *datastorePostgresTestSuite) TestUpsertEvent_MultipleOccurrencesFlushOnce() {
	event := &events.AdministrationEvent{
		Level:   storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message: "message",
		Type:    storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		Hint:    "hint",
		Domain:  "domain",
	}

	err := s.datastore.AddEvent(s.writeCtx, event)
	s.Require().NoError(err)

	err = s.datastore.AddEvent(s.writeCtx, event)
	s.Require().NoError(err)

	err = s.datastore.Flush(s.writeCtx)
	s.Require().NoError(err)

	id := "73072ecb-2222-5922-8948-5944338861c8"
	dbEvent, err := s.datastore.GetEventByID(s.readCtx, id)
	s.Require().NoError(err)
	s.assertEventsEqual(event, dbEvent)
	s.EqualValues(dbEvent.GetNumOccurrences(), 2)
}

func (s *datastorePostgresTestSuite) TestUpsertEvent_MultipleOccurrencesFlushEach() {
	event := &events.AdministrationEvent{
		Level:   storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message: "message",
		Type:    storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		Hint:    "hint",
		Domain:  "domain",
	}

	err := s.datastore.AddEvent(s.writeCtx, event)
	s.Require().NoError(err)

	err = s.datastore.Flush(s.writeCtx)
	s.Require().NoError(err)

	err = s.datastore.AddEvent(s.writeCtx, event)
	s.Require().NoError(err)

	err = s.datastore.Flush(s.writeCtx)
	s.Require().NoError(err)

	id := "73072ecb-2222-5922-8948-5944338861c8"
	dbEvent, err := s.datastore.GetEventByID(s.readCtx, id)
	s.Require().NoError(err)
	s.assertEventsEqual(event, dbEvent)
	s.EqualValues(dbEvent.GetNumOccurrences(), 2)
}
