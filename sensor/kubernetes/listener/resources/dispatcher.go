package resources

import (
	"fmt"
	"strings"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	metricsPkg "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/sensor/common/awscredentials"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/metrics"
	"github.com/stackrox/rox/sensor/common/registry"
	complianceOperatorDispatchers "github.com/stackrox/rox/sensor/kubernetes/listener/resources/complianceoperator/dispatchers"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/rbac"
	"github.com/stackrox/rox/sensor/kubernetes/orchestratornamespaces"
	v1Listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// Dispatcher is responsible for processing resource events, and returning the sensor events that should be emitted
// in response.
//go:generate mockgen-wrapper
type Dispatcher interface {
	ProcessEvent(obj, oldObj interface{}, action central.ResourceAction) []*central.SensorEvent
}

// DispatcherRegistry provides dispatchers to use.
type DispatcherRegistry interface {
	ForDeployments(deploymentType string) Dispatcher
	ForJobs() Dispatcher

	ForNamespaces() Dispatcher
	ForNetworkPolicies() Dispatcher
	ForNodes() Dispatcher
	ForSecrets() Dispatcher
	ForServices() Dispatcher
	ForOpenshiftRoutes() Dispatcher
	ForServiceAccounts() Dispatcher
	ForRBAC() Dispatcher
	ForClusterOperators() Dispatcher

	ForComplianceOperatorResults() Dispatcher
	ForComplianceOperatorProfiles() Dispatcher
	ForComplianceOperatorRules() Dispatcher
	ForComplianceOperatorScanSettingBindings() Dispatcher
	ForComplianceOperatorScans() Dispatcher
	ForComplianceOperatorTailoredProfiles() Dispatcher
}

// NewDispatcherRegistry creates and returns a new DispatcherRegistry.
func NewDispatcherRegistry(
	clusterID string,
	podLister v1Listers.PodLister,
	profileLister cache.GenericLister,
	entityStore *clusterentities.Store,
	processFilter filter.Filter,
	configHandler config.Handler,
	detector detector.Detector,
	namespaces *orchestratornamespaces.OrchestratorNamespaces,
	credentialsManager awscredentials.RegistryCredentialsManager,
) DispatcherRegistry {
	serviceStore := newServiceStore()
	deploymentStore := DeploymentStoreSingleton()
	podStore := PodStoreSingleton()
	nodeStore := newNodeStore()
	nsStore := newNamespaceStore()
	netPolicyStore := NetworkPolicySingleton()
	endpointManager := newEndpointManager(serviceStore, deploymentStore, podStore, nodeStore, entityStore)
	rbacUpdater := rbac.NewStore()
	portExposureReconciler := newPortExposureReconciler(deploymentStore, serviceStore)
	registryStore := registry.Singleton()

	return &registryImpl{
		deploymentHandler: newDeploymentHandler(clusterID, serviceStore, deploymentStore, podStore, endpointManager, nsStore,
			rbacUpdater, podLister, processFilter, configHandler, detector, namespaces, registryStore, credentialsManager),

		rbacDispatcher:            rbac.NewDispatcher(rbacUpdater),
		namespaceDispatcher:       newNamespaceDispatcher(nsStore, serviceStore, deploymentStore, podStore, netPolicyStore),
		serviceDispatcher:         newServiceDispatcher(serviceStore, deploymentStore, endpointManager, portExposureReconciler),
		osRouteDispatcher:         newRouteDispatcher(serviceStore, portExposureReconciler),
		secretDispatcher:          newSecretDispatcher(registryStore),
		networkPolicyDispatcher:   newNetworkPolicyDispatcher(netPolicyStore, deploymentStore, detector),
		nodeDispatcher:            newNodeDispatcher(serviceStore, deploymentStore, nodeStore, endpointManager),
		serviceAccountDispatcher:  newServiceAccountDispatcher(),
		clusterOperatorDispatcher: newClusterOperatorDispatcher(namespaces),

		complianceOperatorResultDispatcher:              complianceOperatorDispatchers.NewResultDispatcher(),
		complianceOperatorRulesDispatcher:               complianceOperatorDispatchers.NewRulesDispatcher(),
		complianceOperatorProfileDispatcher:             complianceOperatorDispatchers.NewProfileDispatcher(),
		complianceOperatorScanSettingBindingsDispatcher: complianceOperatorDispatchers.NewScanSettingBindingsDispatcher(),
		complianceOperatorScanDispatcher:                complianceOperatorDispatchers.NewScanDispatcher(),
		complianceOperatorTailoredProfileDispatcher:     complianceOperatorDispatchers.NewTailoredProfileDispatcher(profileLister),
	}
}

type registryImpl struct {
	deploymentHandler *deploymentHandler

	rbacDispatcher            *rbac.Dispatcher
	namespaceDispatcher       *namespaceDispatcher
	serviceDispatcher         *serviceDispatcher
	osRouteDispatcher         *routeDispatcher
	secretDispatcher          *secretDispatcher
	networkPolicyDispatcher   *networkPolicyDispatcher
	nodeDispatcher            *nodeDispatcher
	serviceAccountDispatcher  *serviceAccountDispatcher
	clusterOperatorDispatcher *clusterOperatorDispatcher

	complianceOperatorResultDispatcher              *complianceOperatorDispatchers.ResultDispatcher
	complianceOperatorProfileDispatcher             *complianceOperatorDispatchers.ProfileDispatcher
	complianceOperatorScanSettingBindingsDispatcher *complianceOperatorDispatchers.ScanSettingBindings
	complianceOperatorRulesDispatcher               *complianceOperatorDispatchers.RulesDispatcher
	complianceOperatorScanDispatcher                *complianceOperatorDispatchers.ScanDispatcher
	complianceOperatorTailoredProfileDispatcher     *complianceOperatorDispatchers.TailoredProfileDispatcher
}

func wrapWithMetricDispatcher(d Dispatcher) Dispatcher {
	return metricDispatcher{
		Dispatcher: d,
	}
}

type metricDispatcher struct {
	Dispatcher
}

func (m metricDispatcher) ProcessEvent(obj, oldObj interface{}, action central.ResourceAction) []*central.SensorEvent {
	start := time.Now().UnixNano()
	dispatcher := strings.Trim(fmt.Sprintf("%T", obj), "*")

	events := m.Dispatcher.ProcessEvent(obj, oldObj, action)
	for _, e := range events {
		e.Timing = &central.Timing{
			Dispatcher: dispatcher,
			Resource:   metricsPkg.GetResourceString(e),
			Nanos:      start,
		}
		metrics.SetResourceProcessingDurationForResource(e)
	}
	metrics.IncK8sEventCount(action.String(), dispatcher)
	return events
}

func (d *registryImpl) ForDeployments(deploymentType string) Dispatcher {
	return wrapWithMetricDispatcher(newDeploymentDispatcher(deploymentType, d.deploymentHandler))
}

func (d *registryImpl) ForJobs() Dispatcher {
	return wrapWithMetricDispatcher(newJobDispatcherImpl(d.deploymentHandler))
}

func (d *registryImpl) ForNamespaces() Dispatcher {
	return wrapWithMetricDispatcher(d.namespaceDispatcher)
}

func (d *registryImpl) ForNetworkPolicies() Dispatcher {
	return wrapWithMetricDispatcher(d.networkPolicyDispatcher)
}

func (d *registryImpl) ForNodes() Dispatcher {
	return wrapWithMetricDispatcher(d.nodeDispatcher)
}

func (d *registryImpl) ForSecrets() Dispatcher {
	return wrapWithMetricDispatcher(d.secretDispatcher)
}

func (d *registryImpl) ForServices() Dispatcher {
	return wrapWithMetricDispatcher(d.serviceDispatcher)
}

func (d *registryImpl) ForOpenshiftRoutes() Dispatcher {
	return wrapWithMetricDispatcher(d.osRouteDispatcher)
}

func (d *registryImpl) ForServiceAccounts() Dispatcher {
	return wrapWithMetricDispatcher(d.serviceAccountDispatcher)
}

func (d *registryImpl) ForRBAC() Dispatcher {
	return wrapWithMetricDispatcher(d.rbacDispatcher)
}

func (d *registryImpl) ForClusterOperators() Dispatcher {
	return wrapWithMetricDispatcher(d.clusterOperatorDispatcher)
}

func (d *registryImpl) ForComplianceOperatorResults() Dispatcher {
	return wrapWithMetricDispatcher(d.complianceOperatorResultDispatcher)
}

func (d *registryImpl) ForComplianceOperatorProfiles() Dispatcher {
	return wrapWithMetricDispatcher(d.complianceOperatorProfileDispatcher)
}

func (d *registryImpl) ForComplianceOperatorTailoredProfiles() Dispatcher {
	return wrapWithMetricDispatcher(d.complianceOperatorTailoredProfileDispatcher)
}

func (d *registryImpl) ForComplianceOperatorRules() Dispatcher {
	return wrapWithMetricDispatcher(d.complianceOperatorRulesDispatcher)
}

func (d *registryImpl) ForComplianceOperatorScanSettingBindings() Dispatcher {
	return wrapWithMetricDispatcher(d.complianceOperatorScanSettingBindingsDispatcher)
}

func (d *registryImpl) ForComplianceOperatorScans() Dispatcher {
	return wrapWithMetricDispatcher(d.complianceOperatorScanDispatcher)
}
