//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sac/testutils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestAdministrationEventsDatastoreSAC(t *testing.T) {
	suite.Run(t, new(administrationEventsDatastoreSACTestSuite))
}

type administrationEventsDatastoreSACTestSuite struct {
	suite.Suite

	datastore DataStore

	pgTestBase *pgtest.TestPostgres

	testContexts map[string]context.Context
}

func (s *administrationEventsDatastoreSACTestSuite) SetupSuite() {
	s.pgTestBase = pgtest.ForT(s.T())
	s.Require().NotNil(s.pgTestBase)
	s.datastore = GetTestPostgresDataStore(s.T(), s.pgTestBase.DB)
	s.testContexts = testutils.GetGloballyScopedTestContexts(context.Background(), s.T(), resources.Integration, resources.Administration)
}

func (s *administrationEventsDatastoreSACTestSuite) TearDownSuite() {
	s.pgTestBase.DB.Close()
}

func (s *administrationEventsDatastoreSACTestSuite) TearDownTest() {
	// Clean up all events after each test
	unrestrictedCtx := sac.WithAllAccess(context.Background())
	allEvents, err := s.datastore.ListEvents(unrestrictedCtx, nil)
	s.Require().NoError(err)
	ids := make([]string, 0, len(allEvents))
	for _, event := range allEvents {
		ids = append(ids, event.GetId())
	}
	if len(ids) > 0 {
		err = RemoveTestEvents(unrestrictedCtx, s.T(), s.datastore, ids...)
		s.Require().NoError(err)
	}
}

// Helper functions to create test objects
func (s *administrationEventsDatastoreSACTestSuite) createTestEvent() *events.AdministrationEvent {
	return &events.AdministrationEvent{
		Domain:       "test-domain-" + uuid.NewV4().String(),
		Level:        storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_INFO,
		Message:      "Test event message",
		Hint:         "Test hint",
		Type:         storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
		ResourceID:   uuid.NewV4().String(),
		ResourceType: "TestResource",
		ResourceName: "test-resource",
	}
}

func (s *administrationEventsDatastoreSACTestSuite) createTestStorageEvent() *storage.AdministrationEvent {
	event := s.createTestEvent()
	return event.ToStorageEvent()
}

func (s *administrationEventsDatastoreSACTestSuite) objectIDExtractor(obj *storage.AdministrationEvent) string {
	return obj.GetId()
}

func (s *administrationEventsDatastoreSACTestSuite) objectCreator() *storage.AdministrationEvent {
	return s.createTestStorageEvent()
}

func (s *administrationEventsDatastoreSACTestSuite) objectInjector(ctx context.Context, obj *storage.AdministrationEvent) error {
	return UpsertTestEvents(ctx, s.T(), s.datastore, obj)
}

func (s *administrationEventsDatastoreSACTestSuite) objectGetter(ctx context.Context, id string) (*storage.AdministrationEvent, bool, error) {
	obj, err := s.datastore.GetEvent(ctx, id)
	if err != nil {
		return nil, false, err
	}
	if obj == nil {
		return nil, false, nil
	}
	return obj, true, nil
}

func (s *administrationEventsDatastoreSACTestSuite) objectRemover(ctx context.Context, id string) error {
	return RemoveTestEvents(ctx, s.T(), s.datastore, id)
}

func (s *administrationEventsDatastoreSACTestSuite) TestAddEvent() {
	cases := testutils.GenericGlobalSACWriteTestCases("add")

	// Use a custom test implementation since we need to convert between event types
	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]

			// Create test event (events.AdministrationEvent)
			event := s.createTestEvent()

			// Try to add with the test context
			err := s.datastore.AddEvent(ctx, event)

			// AddEvent itself may fail with access denied
			if c.ExpectError {
				s.Error(err)
				if c.ExpectedError != nil {
					s.ErrorIs(err, c.ExpectedError)
				}
				return
			}

			s.NoError(err)

			// Flush may also fail with access denied
			err = s.datastore.Flush(ctx)
			if c.ExpectError {
				s.Error(err)
				if c.ExpectedError != nil {
					s.ErrorIs(err, c.ExpectedError)
				}

				// Cleanup: flush with unrestricted access to clear buffer
				unrestrictedCtx := sac.WithAllAccess(context.Background())
				_ = s.datastore.Flush(unrestrictedCtx)
				return
			}

			s.NoError(err)

			// Verify the event was added
			unrestrictedCtx := sac.WithAllAccess(context.Background())
			eventID := events.GenerateEventID(event)
			storedEvent, found, err := s.objectGetter(unrestrictedCtx, eventID)
			s.NoError(err)
			s.True(found)
			s.NotNil(storedEvent)

			// Cleanup
			_ = s.objectRemover(unrestrictedCtx, eventID)
		})
	}
}

func (s *administrationEventsDatastoreSACTestSuite) TestFlush() {
	cases := testutils.GenericGlobalSACWriteTestCases("flush")

	for name, c := range cases {
		s.Run(name, func() {
			ctx := s.testContexts[c.ScopeKey]

			// Create a test event with unrestricted context
			unrestrictedCtx := sac.WithAllAccess(context.Background())
			event := s.createTestEvent()
			eventID := events.GenerateEventID(event)

			err := s.datastore.AddEvent(unrestrictedCtx, event)
			s.Require().NoError(err)

			// Try to flush with the test context
			err = s.datastore.Flush(ctx)

			if c.ExpectError {
				s.Error(err)
				if c.ExpectedError != nil {
					s.ErrorIs(err, c.ExpectedError)
				}
				// Cleanup: flush with unrestricted context to clear buffer
				_ = s.datastore.Flush(unrestrictedCtx)
			} else {
				s.NoError(err)

				// Verify event was flushed
				storedEvent, found, err := s.objectGetter(unrestrictedCtx, eventID)
				s.NoError(err)
				s.True(found)
				s.NotNil(storedEvent)
			}

			// Cleanup
			_ = s.objectRemover(unrestrictedCtx, eventID)
		})
	}
}

func (s *administrationEventsDatastoreSACTestSuite) TestGetEvent() {
	cases := testutils.GenericGlobalSACReadTestCasesNoAccessNoError("get")

	testutils.RunGetTests(
		s.T(),
		cases,
		s.testContexts,
		s.objectIDExtractor,
		s.objectCreator,
		s.objectInjector,
		s.objectGetter,
		s.objectRemover,
	)
}

func (s *administrationEventsDatastoreSACTestSuite) TestCountEvents() {
	cases := testutils.GenericGlobalSACReadTestCasesNoAccessNoError("count")

	testutils.RunCountTests(
		s.T(),
		cases,
		s.testContexts,
		s.objectIDExtractor,
		s.objectCreator,
		s.objectInjector,
		func(ctx context.Context, query *v1.Query) (int, error) {
			return s.datastore.CountEvents(ctx, query)
		},
		s.objectRemover,
	)
}

func (s *administrationEventsDatastoreSACTestSuite) TestListEvents() {
	// Create multiple test events and verify they can be listed based on SAC permissions
	cases := testutils.GenericGlobalSACReadTestCasesNoAccessNoError("list")

	const numEvents = 3
	unrestrictedCtx := sac.WithAllAccess(context.Background())

	// Setup: Create multiple test events once with unrestricted access
	var createdEvents []*storage.AdministrationEvent
	var eventIDs []string

	for i := 0; i < numEvents; i++ {
		event := s.objectCreator()
		err := s.objectInjector(unrestrictedCtx, event)
		s.Require().NoError(err)
		createdEvents = append(createdEvents, event)
		eventIDs = append(eventIDs, s.objectIDExtractor(event))
	}

	// Cleanup after all test cases run
	defer func() {
		for _, id := range eventIDs {
			_ = s.objectRemover(unrestrictedCtx, id)
		}
	}()

	// Run all subtests against the same data set
	for testName, testCase := range cases {
		s.Run(testName, func() {
			ctx := s.testContexts[testCase.ScopeKey]
			results, err := s.datastore.ListEvents(ctx, nil)

			// Read operations with globally scoped store don't return errors
			s.NoError(err)
			if testCase.ExpectedFound {
				s.GreaterOrEqual(len(results), numEvents, "Expected to find at least %d events", numEvents)
				// Verify our test events are in the results
				foundIDs := make(map[string]bool)
				for _, r := range results {
					foundIDs[r.GetId()] = true
				}
				for _, expectedID := range eventIDs {
					s.True(foundIDs[expectedID], "Expected to find event %s in results", expectedID)
				}
			} else {
				// When access is denied, should not find our test events
				for _, r := range results {
					for _, expectedID := range eventIDs {
						s.NotEqual(expectedID, r.GetId(), "Should not find event %s when access is denied", expectedID)
					}
				}
			}
		})
	}
}
