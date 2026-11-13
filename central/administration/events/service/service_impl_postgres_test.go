//go:build sql_integration

package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/central/administration/events/datastore"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	namespaceDatastore "github.com/stackrox/rox/central/namespace/datastore"
	"github.com/stackrox/rox/central/views/administrationevents"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/administration/events"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

func TestServicePostgres(t *testing.T) {
	suite.Run(t, new(servicePostgresTestSuite))
}

type servicePostgresTestSuite struct {
	suite.Suite

	ctx          context.Context
	pool         *pgtest.TestPostgres
	datastore    datastore.DataStore
	deploymentDS deploymentDatastore.DataStore
	service      Service
}

func (s *servicePostgresTestSuite) SetupTest() {
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Administration, resources.Deployment, resources.Image),
		),
	)
	s.pool = pgtest.ForT(s.T())
	s.Require().NotNil(s.pool)
	s.datastore = datastore.GetTestPostgresDataStore(s.T(), s.pool)
	var err error
	s.deploymentDS, err = deploymentDatastore.GetTestPostgresDataStore(s.T(), s.pool)
	s.Require().NoError(err)
	s.service = newService(s.datastore, administrationevents.NewEventView(s.pool))
}

func (s *servicePostgresTestSuite) TestCount() {
	fromBeforeNow, err := protocompat.ConvertTimeToTimestampOrError(time.Now().Add(-1 * time.Hour))
	s.Require().NoError(err)
	fromAfterNow, err := protocompat.ConvertTimeToTimestampOrError(time.Now().Add(1 * time.Hour))
	s.Require().NoError(err)
	untilBeforeNow, err := protocompat.ConvertTimeToTimestampOrError(time.Now().Add(-1 * time.Hour))
	s.Require().NoError(err)
	untilAfterNow, err := protocompat.ConvertTimeToTimestampOrError(time.Now().Add(1 * time.Hour))
	s.Require().NoError(err)

	s.addEvents(50)

	// 1. Count events without providing a query filter.
	resp, err := s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{})
	s.NoError(err)
	s.Equal(int32(50), resp.GetCount())

	// 2. Filter events based on the resource type.
	resp, err = s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{Filter: &v1.AdministrationEventsFilter{
		ResourceType: []string{"Image"},
	}})
	s.NoError(err)
	s.Equal(int32(25), resp.GetCount())

	// 3. Filter events based on the type.
	resp, err = s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{Filter: &v1.AdministrationEventsFilter{
		Type: []v1.AdministrationEventType{
			toV1TypeEnum(storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE)},
	}})
	s.NoError(err)
	s.Equal(int32(25), resp.GetCount())

	// 4. Filter events based on the level.
	resp, err = s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{Filter: &v1.AdministrationEventsFilter{
		Level: []v1.AdministrationEventLevel{
			toV1LevelEnum(storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_WARNING)},
	},
	})
	s.NoError(err)
	s.Equal(int32(25), resp.GetCount())

	// 5. Filter events based on the domain.
	resp, err = s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{Filter: &v1.AdministrationEventsFilter{
		Domain: []string{"sample domain 2"},
	}})
	s.NoError(err)
	s.Equal(int32(1), resp.GetCount())

	// 6. Filter events based on the time they were created.

	// 6.1. Filter all events created one hour ago.
	resp, err = s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{Filter: &v1.AdministrationEventsFilter{
		From: fromBeforeNow,
	}})
	s.NoError(err)
	s.Equal(int32(50), resp.GetCount())

	// 6.2. Filter all events created in one hour.
	resp, err = s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{Filter: &v1.AdministrationEventsFilter{
		From: fromAfterNow,
	}})
	s.NoError(err)
	s.Equal(int32(0), resp.GetCount())

	// 6.3. Filter all events created up until in one hour.
	resp, err = s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{Filter: &v1.AdministrationEventsFilter{
		Until: untilAfterNow,
	}})
	s.NoError(err)
	s.Equal(int32(50), resp.GetCount())

	// 6.4. Filter all events created up until one hour ago.
	resp, err = s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{Filter: &v1.AdministrationEventsFilter{
		Until: untilBeforeNow,
	}})
	s.NoError(err)
	s.Equal(int32(0), resp.GetCount())

	// 6.5. Filter all events from one hour ago until in one hour.
	resp, err = s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{Filter: &v1.AdministrationEventsFilter{
		Until: untilAfterNow,
		From:  fromBeforeNow,
	}})
	s.NoError(err)
	s.Equal(int32(50), resp.GetCount())

	// 6.6. Filter all events from in one hour until one hour ago.
	resp, err = s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{Filter: &v1.AdministrationEventsFilter{
		Until: untilBeforeNow,
		From:  fromAfterNow,
	}})
	s.NoError(err)
	s.Equal(int32(0), resp.GetCount())
}

func (s *servicePostgresTestSuite) TestListAdministrationEvents() {
	listEvents := s.addListEvents()

	resp, err := s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{})
	s.NoError(err)
	s.assertMatchEvents(listEvents, resp.GetEvents())

	resp, err = s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: &v1.AdministrationEventsFilter{
			Domain: []string{"Image Scanning"},
		},
	})
	s.NoError(err)
	s.assertMatchEvents([]*events.AdministrationEvent{listEvents[1], listEvents[3], listEvents[5]}, resp.GetEvents())

	resp, err = s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: &v1.AdministrationEventsFilter{
			Domain: []string{"General"},
		},
	})
	s.NoError(err)
	s.assertMatchEvents([]*events.AdministrationEvent{listEvents[0], listEvents[2], listEvents[4]}, resp.GetEvents())

	resp, err = s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: &v1.AdministrationEventsFilter{
			ResourceType: []string{"Image"},
		},
	})
	s.NoError(err)
	s.assertMatchEvents([]*events.AdministrationEvent{listEvents[0], listEvents[1], listEvents[3]}, resp.GetEvents())

	resp, err = s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: &v1.AdministrationEventsFilter{
			ResourceType: []string{"Node"},
		},
	})
	s.NoError(err)
	s.assertMatchEvents([]*events.AdministrationEvent{listEvents[2], listEvents[4], listEvents[5]}, resp.GetEvents())

	resp, err = s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: &v1.AdministrationEventsFilter{
			Type: []v1.AdministrationEventType{
				toV1TypeEnum(storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE)},
		},
	})
	s.NoError(err)
	s.assertMatchEvents([]*events.AdministrationEvent{listEvents[0], listEvents[1], listEvents[2]}, resp.GetEvents())

	resp, err = s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: &v1.AdministrationEventsFilter{
			Type: []v1.AdministrationEventType{
				toV1TypeEnum(storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC)},
		},
	})
	s.NoError(err)
	s.assertMatchEvents([]*events.AdministrationEvent{listEvents[3], listEvents[4], listEvents[5]}, resp.GetEvents())

	resp, err = s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: &v1.AdministrationEventsFilter{
			Level: []v1.AdministrationEventLevel{
				toV1LevelEnum(storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_WARNING),
			},
		},
	})
	s.NoError(err)
	s.assertMatchEvents([]*events.AdministrationEvent{listEvents[2], listEvents[3], listEvents[5]}, resp.GetEvents())

	resp, err = s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: &v1.AdministrationEventsFilter{
			Level: []v1.AdministrationEventLevel{
				toV1LevelEnum(storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR),
			},
		},
	})
	s.NoError(err)
	s.assertMatchEvents([]*events.AdministrationEvent{listEvents[0], listEvents[1], listEvents[4]}, resp.GetEvents())

	resp, err = s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: &v1.AdministrationEventsFilter{
			Domain: []string{"Image Scanning", "General"},
		},
	})
	s.NoError(err)
	s.assertMatchEvents(listEvents, resp.GetEvents())

	resp, err = s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: &v1.AdministrationEventsFilter{
			ResourceType: []string{"Image", "Node"},
		},
	})
	s.NoError(err)
	s.assertMatchEvents(listEvents, resp.GetEvents())

	resp, err = s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: &v1.AdministrationEventsFilter{
			Level: []v1.AdministrationEventLevel{
				toV1LevelEnum(storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR),
				toV1LevelEnum(storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_WARNING),
			},
		},
	})
	s.NoError(err)
	s.assertMatchEvents(listEvents, resp.GetEvents())
}

func (s *servicePostgresTestSuite) TestFilterByCluster() {
	dep := &storage.Deployment{
		Id:          uuid.NewV4().String(),
		Name:        "my-app",
		Namespace:   "default",
		ClusterId:   uuid.NewV4().String(),
		ClusterName: "test-cluster",
		Containers: []*storage.Container{
			{
				Name: "main",
				Image: &storage.ContainerImage{
					Id:   "sha256:abc",
					IdV2: "image-abc",
				},
			},
		},
	}
	s.Require().NoError(s.deploymentDS.UpsertDeployment(s.ctx, dep))

	imgEvent := &events.AdministrationEvent{
		Domain:       "Image Scanning",
		Hint:         "hint",
		Level:        storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message:      "scan failed for sha256:abc",
		ResourceID:   "image-abc",
		ResourceType: "Image",
		Type:         storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE,
	}
	otherEvent := &events.AdministrationEvent{
		Domain:       "Image Scanning",
		Hint:         "hint",
		Level:        storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Message:      "scan failed for sha256:xyz",
		ResourceID:   "sha256:xyz",
		ResourceType: "Image",
		Type:         storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE,
	}
	s.Require().NoError(s.datastore.AddEvent(s.ctx, imgEvent))
	s.Require().NoError(s.datastore.AddEvent(s.ctx, otherEvent))
	newEvent := *imgEvent
	newEvent.Message = "another scan failure for image-abc"
	s.Require().NoError(s.datastore.AddEvent(s.ctx, &newEvent))
	nameOnlyEvent := *otherEvent
	nameOnlyEvent.ResourceID = ""
	nameOnlyEvent.ResourceName = "registry.example.com/image:latest"
	s.Require().NoError(s.datastore.AddEvent(s.ctx, &nameOnlyEvent))
	s.Require().NoError(s.datastore.Flush(s.ctx))

	// Simulate events persisted before the searchable resource_id column existed.
	_, err := s.pool.Exec(s.ctx, "UPDATE administration_events SET resource_id = NULL WHERE id = $1 OR id = $2",
		events.GenerateEventID(imgEvent), events.GenerateEventID(otherEvent))
	s.Require().NoError(err)

	filter := &v1.AdministrationEventsFilter{WorkloadQuery: `Cluster:"test-cluster"`}
	resp, err := s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: filter,
	})
	s.NoError(err)
	s.Require().Len(resp.GetEvents(), 1)
	s.Equal(events.GenerateEventID(&newEvent), resp.GetEvents()[0].GetId())
	count, err := s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{Filter: filter})
	s.Require().NoError(err)
	s.Equal(int32(1), count.GetCount())
	resp, err = s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: filter, Pagination: &v1.Pagination{Limit: 1, Offset: 1},
	})
	s.Require().NoError(err)
	s.Empty(resp.GetEvents())

	resp, err = s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: &v1.AdministrationEventsFilter{
			WorkloadQuery: `Cluster:"nonexistent"`,
		},
	})
	s.NoError(err)
	s.Empty(resp.GetEvents())
	count, err = s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{
		Filter: &v1.AdministrationEventsFilter{WorkloadQuery: `Cluster:"nonexistent"`},
	})
	s.Require().NoError(err)
	s.Zero(count.GetCount())

	// Events without a searchable resource ID are still visible without workload filters.
	resp, err = s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{})
	s.Require().NoError(err)
	s.Require().Len(resp.GetEvents(), 4)
	s.assertMatchEvents([]*events.AdministrationEvent{imgEvent, otherEvent, &newEvent, &nameOnlyEvent}, resp.GetEvents())
	count, err = s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{})
	s.Require().NoError(err)
	s.Equal(int32(4), count.GetCount())
}

func (s *servicePostgresTestSuite) TestSQLWorkloadFilter() {
	clusterID := uuid.NewV4().String()
	namespaceDS, err := namespaceDatastore.GetTestPostgresDataStore(s.T(), s.pool)
	s.Require().NoError(err)
	for _, name := range []string{"team-a", "team-b"} {
		namespace := &storage.NamespaceMetadata{
			Id: uuid.NewV4().String(), Name: name, ClusterId: clusterID,
			Labels: map[string]string{"team": name},
		}
		s.Require().NoError(namespaceDS.AddNamespace(sac.WithAllAccess(s.ctx), namespace))
		dep := &storage.Deployment{
			Id: uuid.NewV4().String(), Name: "app", ClusterId: clusterID, ClusterName: "test-cluster",
			Namespace: name, NamespaceId: namespace.GetId(),
			Containers: []*storage.Container{{Name: "main", Image: &storage.ContainerImage{IdV2: name}}},
		}
		s.Require().NoError(s.deploymentDS.UpsertDeployment(s.ctx, dep))
		// Several matching deployments must not duplicate events or consume page slots.
		dep.Id = uuid.NewV4().String()
		s.Require().NoError(s.deploymentDS.UpsertDeployment(s.ctx, dep))
	}
	imageA := &events.AdministrationEvent{Domain: "Image Scanning", ResourceType: "Image", ResourceID: "team-a"}
	imageB := &events.AdministrationEvent{Domain: "Image Scanning", ResourceType: "Image", ResourceID: "team-b"}
	node := &events.AdministrationEvent{Domain: "General", ResourceType: "Node", ResourceID: "node"}
	for _, event := range []*events.AdministrationEvent{imageA, imageB, node} {
		event.Message = "operation failed"
		event.Level = storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR
		event.Type = storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE
		s.Require().NoError(s.datastore.AddEvent(s.ctx, event))
	}
	s.Require().NoError(s.datastore.Flush(s.ctx))

	adminOnly := sac.WithGlobalAccessScopeChecker(s.ctx, sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS), sac.ResourceScopeKeys(resources.Administration)))
	deploymentOnly := sac.WithGlobalAccessScopeChecker(s.ctx, sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS), sac.ResourceScopeKeys(resources.Deployment)))
	namespaceA := sac.WithGlobalAccessScopeChecker(s.ctx, sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		sac.ResourceScopeKeys(resources.Administration, resources.Deployment),
		sac.ClusterScopeKeys(clusterID), sac.NamespaceScopeKeys("team-a")))

	for name, test := range map[string]struct {
		ctx           context.Context
		workloadQuery string
		resourceTypes []string
		domains       []string
		expected      []*events.AdministrationEvent
	}{
		"all resource types":      {expected: []*events.AdministrationEvent{imageA, imageB, node}},
		"images only":             {resourceTypes: []string{"Image"}, expected: []*events.AdministrationEvent{imageA, imageB}},
		"non-image only":          {resourceTypes: []string{"Node"}, expected: []*events.AdministrationEvent{node}},
		"event predicate":         {domains: []string{"Image Scanning"}, expected: []*events.AdministrationEvent{imageA, imageB}},
		"related table predicate": {workloadQuery: `Namespace Label:team=team-a`, expected: []*events.AdministrationEvent{imageA, node}},
		"no matching deployment":  {workloadQuery: `Cluster:nonexistent`, expected: []*events.AdministrationEvent{node}},
		"bound query value":       {workloadQuery: `Cluster:"test-cluster' OR true -- $$"`, expected: []*events.AdministrationEvent{node}},
		"namespace access":        {ctx: namespaceA, expected: []*events.AdministrationEvent{imageA, node}},
		"out of scope predicate":  {ctx: namespaceA, workloadQuery: `Namespace:team-b`, expected: []*events.AdministrationEvent{node}},
		"no deployment access":    {ctx: adminOnly, expected: []*events.AdministrationEvent{node}},
		"no image matches":        {ctx: adminOnly, resourceTypes: []string{"Image"}},
		"no event access":         {ctx: deploymentOnly},
	} {
		s.Run(name, func() {
			ctx := test.ctx
			if ctx == nil {
				ctx = s.ctx
			}
			workloadQuery := test.workloadQuery
			if workloadQuery == "" {
				workloadQuery = `Cluster:test-cluster`
			}
			filter := &v1.AdministrationEventsFilter{WorkloadQuery: workloadQuery, ResourceType: test.resourceTypes, Domain: test.domains}
			count, err := s.service.CountAdministrationEvents(ctx, &v1.CountAdministrationEventsRequest{Filter: filter})
			s.Require().NoError(err)
			s.Equal(int32(len(test.expected)), count.GetCount())
			var ids []string
			for offset := 0; offset <= len(test.expected); offset++ {
				resp, err := s.service.ListAdministrationEvents(ctx, &v1.ListAdministrationEventsRequest{
					Filter: filter, Pagination: &v1.Pagination{
						Limit: 1, Offset: int32(offset), SortOption: &v1.SortOption{Field: search.ResourceID.String()},
					},
				})
				s.Require().NoError(err)
				s.LessOrEqual(len(resp.GetEvents()), 1)
				for _, event := range resp.GetEvents() {
					ids = append(ids, event.GetId())
				}
			}
			var expectedIDs []string
			for _, event := range test.expected {
				expectedIDs = append(expectedIDs, events.GenerateEventID(event))
			}
			s.ElementsMatch(expectedIDs, ids)
		})
	}
}

func (s *servicePostgresTestSuite) TestSQLWorkloadFilterBeyondOldLimit() {
	dep := &storage.Deployment{Id: uuid.NewV4().String(), ClusterId: uuid.NewV4().String(), ClusterName: "large-cluster"}
	for i := range 10_001 {
		dep.Containers = append(dep.Containers, &storage.Container{
			Name: fmt.Sprintf("container-%d", i), Image: &storage.ContainerImage{IdV2: fmt.Sprintf("image-%d", i)},
		})
	}
	s.Require().NoError(s.deploymentDS.UpsertDeployment(s.ctx, dep))
	event := &events.AdministrationEvent{
		Domain: "Image Scanning", ResourceType: "Image", ResourceID: "image-10000", Message: "scan failed",
		Level: storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
		Type:  storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE,
	}
	s.Require().NoError(s.datastore.AddEvent(s.ctx, event))
	s.Require().NoError(s.datastore.Flush(s.ctx))
	filter := &v1.AdministrationEventsFilter{WorkloadQuery: `Cluster:large-cluster`}
	count, err := s.service.CountAdministrationEvents(s.ctx, &v1.CountAdministrationEventsRequest{Filter: filter})
	s.Require().NoError(err)
	s.Equal(int32(1), count.GetCount())
	resp, err := s.service.ListAdministrationEvents(s.ctx, &v1.ListAdministrationEventsRequest{
		Filter: filter, Pagination: &v1.Pagination{Limit: 1},
	})
	s.Require().NoError(err)
	s.Require().Len(resp.GetEvents(), 1)
	s.Equal(events.GenerateEventID(event), resp.GetEvents()[0].GetId())
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
