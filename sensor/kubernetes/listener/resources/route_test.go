package resources

import (
	"testing"

	routeV1 "github.com/openshift/api/route/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/selector"
	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func getSelector(svc *v1.Service) selector.Selector {
	return selector.CreateSelector(svc.Spec.Selector, selector.EmptyMatchesNothing())
}

func getTestService(name, namespace string) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{TargetPort: intstr.FromInt(8000), Port: 443},
			},
			Selector: map[string]string{
				"app": name,
			},
		},
	}
}

func getTestRoute(namespace, targetServiceName string) *routeV1.Route {
	return &routeV1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      uuid.NewV4().String(),
			Namespace: namespace,
		},
		Spec: routeV1.RouteSpec{To: routeV1.RouteTargetReference{Name: targetServiceName, Kind: "Service"}},
	}
}

type call struct {
	funcName string
	args     []interface{}
}

// Mock implementations of the two interfaces passed into our dispatchers.
// We cannot use mockgen for these types because they reference unexported types in their arguments.
type mockPortExposureReconciler struct {
	orderedCalls []call
}

func (m *mockPortExposureReconciler) UpdateExposuresForMatchingDeployments(namespace string, sel selector.Selector) []*central.SensorEvent {
	m.orderedCalls = append(m.orderedCalls,
		call{
			"UpdateExposuresForMatchingDeployments",
			[]interface{}{namespace, sel},
		})
	return nil
}

func (m *mockPortExposureReconciler) UpdateExposureOnServiceCreate(svc serviceWithRoutes) []*central.SensorEvent {
	m.orderedCalls = append(m.orderedCalls,
		call{
			"UpdateExposureOnServiceCreate",
			[]interface{}{svc},
		})

	return nil
}

type mockEndpointManager struct {
}

func (m *mockEndpointManager) OnDeploymentCreateOrUpdateByID(string) {
}

func (m *mockEndpointManager) OnDeploymentCreateOrUpdate(*deploymentWrap) {
}

func (m *mockEndpointManager) OnDeploymentRemove(*deploymentWrap) {
}

func (m *mockEndpointManager) OnServiceCreate(*serviceWrap) {
}

func (m *mockEndpointManager) OnServiceUpdateOrRemove(string, selector.Selector) {
}

func (m *mockEndpointManager) OnNodeCreate(*nodeWrap) {
}

func (m *mockEndpointManager) OnNodeUpdateOrRemove() {
}

func getSvcWithRoutes(svc *v1.Service, routes ...*routeV1.Route) serviceWithRoutes {
	return serviceWithRoutes{
		serviceWrap: wrapService(svc),
		routes:      routes,
	}
}

func TestRouteAndServiceDispatchers(t *testing.T) {
	suite.Run(t, new(RouteAndServiceDispatcherTestSuite))
}

type RouteAndServiceDispatcherTestSuite struct {
	suite.Suite

	depStore     *DeploymentStore
	serviceStore *serviceStore

	serviceDispatcher *serviceDispatcher
	routeDispatcher   *routeDispatcher

	mockReconciler      *mockPortExposureReconciler
	mockEndpointManager *mockEndpointManager
}

func (suite *RouteAndServiceDispatcherTestSuite) SetupTest() {
	suite.mockReconciler = &mockPortExposureReconciler{}

	suite.mockEndpointManager = &mockEndpointManager{}

	suite.depStore = newDeploymentStore()
	suite.serviceStore = newServiceStore()
	suite.serviceDispatcher = newServiceDispatcher(suite.serviceStore, suite.depStore, suite.mockEndpointManager, suite.mockReconciler)
	suite.routeDispatcher = newRouteDispatcher(suite.serviceStore, suite.mockReconciler)
}

func (suite *RouteAndServiceDispatcherTestSuite) TestServiceCreateNoRoute() {
	if env.ResyncDisabled.BooleanSetting() {
		// TODO(ROX-14310): remove the test
		suite.T().Skip("If re-sync is disabled we don't call EndpointManager for CREATE and UPDATE events in the dispatcher")
	}
	testService := getTestService("test-svc", "test-ns")
	suite.serviceDispatcher.ProcessEvent(testService, nil, central.ResourceAction_CREATE_RESOURCE)

	suite.Equal([]call{
		{
			"UpdateExposureOnServiceCreate",
			[]interface{}{getSvcWithRoutes(testService)},
		},
	}, suite.mockReconciler.orderedCalls)
}

func (suite *RouteAndServiceDispatcherTestSuite) TestServiceCreateWithPreexistingRoute() {
	if env.ResyncDisabled.BooleanSetting() {
		// TODO(ROX-14310): remove the test
		suite.T().Skip("If re-sync is disabled we don't call EndpointManager for CREATE and UPDATE events in the dispatcher")
	}
	testRoute := getTestRoute("test-ns", "test-svc")
	testService := getTestService("test-svc", "test-ns")
	suite.routeDispatcher.ProcessEvent(testRoute, nil, central.ResourceAction_CREATE_RESOURCE)
	suite.serviceDispatcher.ProcessEvent(testService, nil, central.ResourceAction_CREATE_RESOURCE)

	suite.Equal([]call{
		{
			"UpdateExposureOnServiceCreate",
			[]interface{}{getSvcWithRoutes(testService, testRoute)},
		},
	}, suite.mockReconciler.orderedCalls)
}

func (suite *RouteAndServiceDispatcherTestSuite) TestManyRoutesMatchingAndDeletions() {
	if env.ResyncDisabled.BooleanSetting() {
		// TODO(ROX-14310): remove the test
		suite.T().Skip("If re-sync is disabled we don't call EndpointManager for CREATE and UPDATE events in the dispatcher")
	}
	testRouteSvc1 := getTestRoute("test-ns", "test-svc")
	testSvc1 := getTestService("test-svc", "test-ns")
	testRoute1Svc2 := getTestRoute("test-ns", "test-svc-2")
	testRoute2Svc2 := getTestRoute("test-ns", "test-svc-2")
	testSvc2 := getTestService("test-svc-2", "test-ns")

	testRouteOtherNS := getTestRoute("other-ns", "test-svc")

	// Process some routes first
	suite.routeDispatcher.ProcessEvent(testRouteSvc1, nil, central.ResourceAction_CREATE_RESOURCE)
	suite.routeDispatcher.ProcessEvent(testRoute1Svc2, nil, central.ResourceAction_CREATE_RESOURCE)
	suite.routeDispatcher.ProcessEvent(testRouteOtherNS, nil, central.ResourceAction_CREATE_RESOURCE)

	// Process the services
	suite.serviceDispatcher.ProcessEvent(testSvc1, nil, central.ResourceAction_CREATE_RESOURCE)
	suite.serviceDispatcher.ProcessEvent(testSvc2, nil, central.ResourceAction_CREATE_RESOURCE)

	// Now create a new route.
	suite.routeDispatcher.ProcessEvent(testRoute2Svc2, nil, central.ResourceAction_CREATE_RESOURCE)

	// Now delete an old route
	suite.routeDispatcher.ProcessEvent(testRoute1Svc2, nil, central.ResourceAction_REMOVE_RESOURCE)

	suite.Equal([]call{
		{
			"UpdateExposureOnServiceCreate",
			[]interface{}{getSvcWithRoutes(testSvc1, testRouteSvc1)},
		},
		{
			"UpdateExposureOnServiceCreate",
			[]interface{}{getSvcWithRoutes(testSvc2, testRoute1Svc2)},
		},
		// After the creating of testRoute2Svc2
		{
			"UpdateExposuresForMatchingDeployments",
			[]interface{}{"test-ns", getSelector(testSvc2)},
		},
		// After the deletion of testRoute1Svc2
		{
			"UpdateExposuresForMatchingDeployments",
			[]interface{}{"test-ns", getSelector(testSvc2)},
		},
	}, suite.mockReconciler.orderedCalls)

}
