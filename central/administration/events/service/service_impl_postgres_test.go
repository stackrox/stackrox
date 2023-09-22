//go:build sql_integration

package service

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/administration/events/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
)

func TestServicePostgres(t *testing.T) {
	suite.Run(t, new(servicePostgresTestSuite))
}

type servicePostgresTestSuite struct {
	suite.Suite

	ctx       context.Context
	pool      *pgtest.TestPostgres
	datastore datastore.DataStore
	service   Service
}

func (s *servicePostgresTestSuite) SetupTest() {
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration),
		),
	)
	s.pool = pgtest.ForT(s.T())
	s.Require().NotNil(s.pool)
	s.datastore = datastore.GetTestPostgresDataStore(s.T(), s.pool)
	s.service = newService(s.datastore)
}

func (s *servicePostgresTestSuite) TearDownTest() {
	s.pool.Teardown(s.T())
	s.pool.Close()
}

func (s *servicePostgresTestSuite) TestCount() {
	fromBeforeNow, err := types.TimestampProto(time.Now().Add(-1 * time.Hour))
	s.Require().NoError(err)
	fromAfterNow, err := types.TimestampProto(time.Now().Add(1 * time.Hour))
	s.Require().NoError(err)
	untilBeforeNow, err := types.TimestampProto(time.Now().Add(-1 * time.Hour))
	s.Require().NoError(err)
	untilAfterNow, err := types.TimestampProto(time.Now().Add(1 * time.Hour))
	s.Require().NoError(err)

	s.addEvents(50)

	// 1. Count events without providing a query filter.
	resp, err := s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{})
	s.NoError(err)
	s.Equal(int64(50), resp.GetCount())

	// 2. Filter events based on the resource type.
	resp, err = s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{Filter: &v1.AdministrationEventsFilter{
		ResourceType: "Image",
	}})
	s.NoError(err)
	s.Equal(int64(25), resp.GetCount())

	// 3. Filter events based on the type.
	resp, err = s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{Filter: &v1.AdministrationEventsFilter{
		Type: toV1TypeEnum(storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE),
	}})
	s.NoError(err)
	s.Equal(int64(25), resp.GetCount())

	// 4. Filter events based on the level.
	resp, err = s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{Filter: &v1.AdministrationEventsFilter{
		Level: toV1LevelEnum(storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_WARNING),
	}})
	s.NoError(err)
	s.Equal(int64(25), resp.GetCount())

	// 5. Filter events based on the domain.
	resp, err = s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{Filter: &v1.AdministrationEventsFilter{
		Domain: "sample domain 2",
	}})
	s.NoError(err)
	s.Equal(int64(1), resp.GetCount())

	// 6. Filter events based on the time they were created.

	// 6.1. Filter all events created one hour ago.
	resp, err = s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{Filter: &v1.AdministrationEventsFilter{
		From: fromBeforeNow,
	}})
	s.NoError(err)
	s.Equal(int64(50), resp.GetCount())

	// 6.2. Filter all events created in one hour.
	resp, err = s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{Filter: &v1.AdministrationEventsFilter{
		From: fromAfterNow,
	}})
	s.NoError(err)
	s.Equal(int64(0), resp.GetCount())

	// 6.3. Filter all events created up until in one hour.
	resp, err = s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{Filter: &v1.AdministrationEventsFilter{
		Until: untilAfterNow,
	}})
	s.NoError(err)
	s.Equal(int64(50), resp.GetCount())

	// 6.4. Filter all events created up until one hour ago.
	resp, err = s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{Filter: &v1.AdministrationEventsFilter{
		Until: untilBeforeNow,
	}})
	s.NoError(err)
	s.Equal(int64(0), resp.GetCount())

	// 6.5. Filter all events from one hour ago until in one hour.
	resp, err = s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{Filter: &v1.AdministrationEventsFilter{
		Until: untilAfterNow,
		From:  fromBeforeNow,
	}})
	s.NoError(err)
	s.Equal(int64(50), resp.GetCount())

	// 6.6. Filter all events from in one hour until one hour ago.
	resp, err = s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{Filter: &v1.AdministrationEventsFilter{
		Until: untilBeforeNow,
		From:  fromAfterNow,
	}})
	s.NoError(err)
	s.Equal(int64(0), resp.GetCount())
}

func (s *servicePostgresTestSuite) TestListAdministrationEvents() {
	listEvents := s.addListEvents()

	resp, err := s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{})
	s.NoError(err)
	s.assertMatchEvents(listEvents, resp.GetEvents())

	resp, err = s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: &v1.AdministrationEventsFilter{
			Domain: "Image Scanning",
		},
	})
	s.NoError(err)
	s.assertMatchEvents([]*events.AdministrationEvent{listEvents[1], listEvents[3], listEvents[5]}, resp.GetEvents())

	resp, err = s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: &v1.AdministrationEventsFilter{
			Domain: "General",
		},
	})
	s.NoError(err)
	s.assertMatchEvents([]*events.AdministrationEvent{listEvents[0], listEvents[2], listEvents[4]}, resp.GetEvents())

	resp, err = s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: &v1.AdministrationEventsFilter{
			ResourceType: "Image",
		},
	})
	s.NoError(err)
	s.assertMatchEvents([]*events.AdministrationEvent{listEvents[0], listEvents[1], listEvents[3]}, resp.GetEvents())

	resp, err = s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: &v1.AdministrationEventsFilter{
			ResourceType: "Node",
		},
	})
	s.NoError(err)
	s.assertMatchEvents([]*events.AdministrationEvent{listEvents[2], listEvents[4], listEvents[5]}, resp.GetEvents())

	resp, err = s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: &v1.AdministrationEventsFilter{
			Type: toV1TypeEnum(storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE),
		},
	})
	s.NoError(err)
	s.assertMatchEvents([]*events.AdministrationEvent{listEvents[0], listEvents[1], listEvents[2]}, resp.GetEvents())

	resp, err = s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: &v1.AdministrationEventsFilter{
			Type: toV1TypeEnum(storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC),
		},
	})
	s.NoError(err)
	s.assertMatchEvents([]*events.AdministrationEvent{listEvents[3], listEvents[4], listEvents[5]}, resp.GetEvents())

	resp, err = s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: &v1.AdministrationEventsFilter{
			Level: toV1LevelEnum(storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_WARNING),
		},
	})
	s.NoError(err)
	s.assertMatchEvents([]*events.AdministrationEvent{listEvents[2], listEvents[3], listEvents[5]}, resp.GetEvents())

	resp, err = s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: &v1.AdministrationEventsFilter{
			Level: toV1LevelEnum(storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR),
		},
	})
	s.NoError(err)
	s.assertMatchEvents([]*events.AdministrationEvent{listEvents[0], listEvents[1], listEvents[4]}, resp.GetEvents())

}

func (s *servicePostgresTestSuite) addListEvents() []*events.AdministrationEvent {
	listEvents := fixtures.GetListAdministrationEvents()
	for _, event := range listEvents {
		s.Require().NoError(s.datastore.AddEvent(s.ctx, event))
	}
	s.Require().NoError(s.datastore.Flush(s.ctx))
	return listEvents
}

func (s *servicePostgresTestSuite) addEvents(numOfEvents int) {
	events := fixtures.GetMultipleAdministrationEvents(numOfEvents)
	for _, event := range events {
		s.Require().NoError(s.datastore.AddEvent(s.ctx, event))
	}
	s.Require().NoError(s.datastore.Flush(s.ctx))
}

func (s *servicePostgresTestSuite) assertMatchEvents(adminEvents []*events.AdministrationEvent, apiEvents []*v1.AdministrationEvent) {
	for _, adminEvent := range adminEvents {
		s.Truef(s.matchAdminEvent(adminEvent, apiEvents), "expected %v to be found within %+v",
			adminEvent, apiEvents)
	}
}

func (s *servicePostgresTestSuite) matchAdminEvent(adminEvent *events.AdministrationEvent, apiEvents []*v1.AdministrationEvent) bool {
	for _, apiEvent := range apiEvents {
		if s.eventsEqual(adminEvent, apiEvent) {
			return true
		}
	}
	return false
}

func (s *servicePostgresTestSuite) eventsEqual(event *events.AdministrationEvent, apiEvent *v1.AdministrationEvent) bool {
	return toV1LevelEnum(event.GetLevel()) == apiEvent.GetLevel() &&
		event.GetMessage() == apiEvent.GetMessage() &&
		toV1TypeEnum(event.GetType()) == apiEvent.GetType() &&
		event.GetHint() == apiEvent.GetHint() &&
		event.GetDomain() == apiEvent.GetDomain() &&
		event.GetResourceID() == apiEvent.GetResource().GetId() &&
		event.GetResourceType() == apiEvent.GetResource().GetType() &&
		event.GetResourceName() == apiEvent.GetResource().GetName()
}
