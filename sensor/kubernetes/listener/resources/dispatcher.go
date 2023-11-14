package resources

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/stackrox/rox/generated/internalapi/central"
	metricsPkg "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/process/filter"
	"github.com/stackrox/rox/sensor/common/awscredentials"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/metrics"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	complianceOperatorDispatchers "github.com/stackrox/rox/sensor/kubernetes/listener/resources/complianceoperator/dispatchers"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/rbac"
	"k8s.io/client-go/kubernetes"
	v1Listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// Dispatcher is responsible for processing resource events, and returning the sensor events that should be emitted
// in response.
//
//go:generate mockgen-wrapper
type Dispatcher interface {
	ProcessEvent(obj, oldObj interface{}, action central.ResourceAction) *component.ResourceEvent
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
	ForRegistryMirrors() Dispatcher

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
	processFilter filter.Filter,
	configHandler config.Handler,
	credentialsManager awscredentials.RegistryCredentialsManager,
	traceWriter io.Writer,
	storeProvider *StoreProvider,
	k8sAPI kubernetes.Interface,
) DispatcherRegistry {
	serviceStore := storeProvider.serviceStore
	rbacUpdater := storeProvider.rbacStore
	serviceAccountStore := storeProvider.serviceAccountStore
	deploymentStore := storeProvider.deploymentStore
	podStore := storeProvider.podStore
	nsStore := storeProvider.nsStore
	netPolicyStore := storeProvider.networkPolicyStore
	endpointManager := storeProvider.endpointManager
	portExposureReconciler := newPortExposureReconciler(deploymentStore, storeProvider.Services())
	registryStore := storeProvider.registryStore
	registryMirrorStore := storeProvider.registryMirrorStore

	return &registryImpl{
		deploymentHandler: newDeploymentHandler(clusterID, storeProvider.Services(), deploymentStore, podStore, endpointManager, nsStore,
			rbacUpdater, podLister, processFilter, configHandler, storeProvider.orchestratorNamespaces, registryStore, credentialsManager),

		rbacDispatcher:             rbac.NewDispatcher(rbacUpdater, k8sAPI),
		namespaceDispatcher:        newNamespaceDispatcher(nsStore, serviceStore, deploymentStore, podStore, netPolicyStore),
		serviceDispatcher:          newServiceDispatcher(serviceStore, deploymentStore, endpointManager, portExposureReconciler),
		osRouteDispatcher:          newRouteDispatcher(serviceStore, portExposureReconciler),
		secretDispatcher:           newSecretDispatcher(registryStore),
		networkPolicyDispatcher:    newNetworkPolicyDispatcher(netPolicyStore, deploymentStore),
		nodeDispatcher:             newNodeDispatcher(deploymentStore, storeProvider.nodeStore, endpointManager),
		serviceAccountDispatcher:   newServiceAccountDispatcher(serviceAccountStore),
		clusterOperatorDispatcher:  newClusterOperatorDispatcher(storeProvider.orchestratorNamespaces),
		osRegistryMirrorDispatcher: newRegistryMirrorDispatcher(registryMirrorStore),

		traceWriter: traceWriter,

		complianceOperatorResultDispatcher:              complianceOperatorDispatchers.NewResultDispatcher(storeProvider.reconciliationStore),
		complianceOperatorRulesDispatcher:               complianceOperatorDispatchers.NewRulesDispatcher(storeProvider.reconciliationStore),
		complianceOperatorProfileDispatcher:             complianceOperatorDispatchers.NewProfileDispatcher(storeProvider.reconciliationStore),
		complianceOperatorScanSettingBindingsDispatcher: complianceOperatorDispatchers.NewScanSettingBindingsDispatcher(storeProvider.reconciliationStore),
		complianceOperatorScanDispatcher:                complianceOperatorDispatchers.NewScanDispatcher(storeProvider.reconciliationStore),
		complianceOperatorTailoredProfileDispatcher:     complianceOperatorDispatchers.NewTailoredProfileDispatcher(storeProvider.reconciliationStore, profileLister),
	}
}

type registryImpl struct {
	deploymentHandler *deploymentHandler

	rbacDispatcher             *rbac.Dispatcher
	namespaceDispatcher        *namespaceDispatcher
	serviceDispatcher          *serviceDispatcher
	osRouteDispatcher          *routeDispatcher
	secretDispatcher           *secretDispatcher
	networkPolicyDispatcher    *networkPolicyDispatcher
	nodeDispatcher             *nodeDispatcher
	serviceAccountDispatcher   *serviceAccountDispatcher
	clusterOperatorDispatcher  *clusterOperatorDispatcher
	osRegistryMirrorDispatcher *registryMirrorDispatcher
	traceWriter                io.Writer

	complianceOperatorResultDispatcher              *complianceOperatorDispatchers.ResultDispatcher
	complianceOperatorProfileDispatcher             *complianceOperatorDispatchers.ProfileDispatcher
	complianceOperatorScanSettingBindingsDispatcher *complianceOperatorDispatchers.ScanSettingBindings
	complianceOperatorRulesDispatcher               *complianceOperatorDispatchers.RulesDispatcher
	complianceOperatorScanDispatcher                *complianceOperatorDispatchers.ScanDispatcher
	complianceOperatorTailoredProfileDispatcher     *complianceOperatorDispatchers.TailoredProfileDispatcher
}

func wrapWithDumpingDispatcher(d Dispatcher, w io.Writer) Dispatcher {
	return dumpingDispatcher{
		writer:     w,
		Dispatcher: d,
	}
}

type dumpingDispatcher struct {
	writer io.Writer
	Dispatcher
}

// InformerK8sMsg is a message being recorded/replayed when collecting the traces with K8s events
type InformerK8sMsg struct {
	ObjectType   string
	Action       string
	Timestamp    int64
	Payload      interface{}
	EventsOutput []string
}

func (m dumpingDispatcher) ProcessEvent(obj, oldObj interface{}, action central.ResourceAction) *component.ResourceEvent {
	now := time.Now().Unix()
	dispType := strings.Trim(fmt.Sprintf("%T", obj), "*")
	events := m.Dispatcher.ProcessEvent(obj, oldObj, action)
	if events == nil {
		events = &component.ResourceEvent{}
	}

	if m.writer == nil {
		return events
	}

	var eventsOutput []string
	marshaler := jsonpb.Marshaler{}
	for _, e := range events.ForwardMessages {
		ev, err := marshaler.MarshalToString(e)
		if err != nil {
			log.Warnf("Error marshaling msg: %s", err.Error())
			return events
		}
		eventsOutput = append(eventsOutput, ev)
	}

	jsonLine, err := json.Marshal(InformerK8sMsg{
		ObjectType:   dispType,
		Timestamp:    now,
		Action:       action.String(),
		Payload:      obj,
		EventsOutput: eventsOutput,
	})
	if err != nil {
		log.Warnf("Error marshaling msg: %s", err.Error())
		return events
	}
	if _, err := m.writer.Write(jsonLine); err != nil {
		log.Warnf("Error writing msg: %s", err.Error())
	}
	return events
}

func wrapWithMetricDispatcher(d Dispatcher) Dispatcher {
	return metricDispatcher{
		Dispatcher: d,
	}
}

type metricDispatcher struct {
	Dispatcher
}

func (m metricDispatcher) ProcessEvent(obj, oldObj interface{}, action central.ResourceAction) *component.ResourceEvent {
	start := time.Now().UnixNano()
	dispatcher := strings.Trim(fmt.Sprintf("%T", obj), "*")

	events := m.Dispatcher.ProcessEvent(obj, oldObj, action)
	if events == nil {
		events = &component.ResourceEvent{}
	}

	for _, e := range events.ForwardMessages {
		e.Timing = &central.Timing{
			Dispatcher: dispatcher,
			Resource:   metricsPkg.GetResourceString(e),
			Nanos:      start,
		}
		metrics.SetResourceProcessingDurationForResource(e)
	}
	metrics.IncK8sEventCount(action.String(), dispatcher)

	events.DeploymentTiming = &central.Timing{
		Dispatcher: dispatcher,
		Resource:   "Deployment",
		Nanos:      start,
	}

	return events
}

func wrapDispatcher(dispatcher Dispatcher, w io.Writer) Dispatcher {
	if w == nil {
		return wrapWithMetricDispatcher(dispatcher)
	}
	return wrapWithMetricDispatcher(wrapWithDumpingDispatcher(dispatcher, w))
}

func (d *registryImpl) ForDeployments(deploymentType string) Dispatcher {
	return wrapDispatcher(newDeploymentDispatcher(deploymentType, d.deploymentHandler), d.traceWriter)
}

func (d *registryImpl) ForJobs() Dispatcher {
	return wrapDispatcher(newJobDispatcherImpl(d.deploymentHandler), d.traceWriter)
}

func (d *registryImpl) ForNamespaces() Dispatcher {
	return wrapDispatcher(d.namespaceDispatcher, d.traceWriter)
}

func (d *registryImpl) ForNetworkPolicies() Dispatcher {
	return wrapDispatcher(d.networkPolicyDispatcher, d.traceWriter)
}

func (d *registryImpl) ForNodes() Dispatcher {
	return wrapDispatcher(d.nodeDispatcher, d.traceWriter)
}

func (d *registryImpl) ForSecrets() Dispatcher {
	return wrapDispatcher(d.secretDispatcher, d.traceWriter)
}

func (d *registryImpl) ForServices() Dispatcher {
	return wrapDispatcher(d.serviceDispatcher, d.traceWriter)
}

func (d *registryImpl) ForOpenshiftRoutes() Dispatcher {
	return wrapDispatcher(d.osRouteDispatcher, d.traceWriter)
}

func (d *registryImpl) ForServiceAccounts() Dispatcher {
	return wrapDispatcher(d.serviceAccountDispatcher, d.traceWriter)
}

func (d *registryImpl) ForRBAC() Dispatcher {
	return wrapDispatcher(d.rbacDispatcher, d.traceWriter)
}

func (d *registryImpl) ForClusterOperators() Dispatcher {
	return wrapDispatcher(d.clusterOperatorDispatcher, d.traceWriter)
}

func (d *registryImpl) ForComplianceOperatorResults() Dispatcher {
	return wrapDispatcher(d.complianceOperatorResultDispatcher, d.traceWriter)
}

func (d *registryImpl) ForComplianceOperatorProfiles() Dispatcher {
	return wrapDispatcher(d.complianceOperatorProfileDispatcher, d.traceWriter)
}

func (d *registryImpl) ForComplianceOperatorTailoredProfiles() Dispatcher {
	return wrapDispatcher(d.complianceOperatorTailoredProfileDispatcher, d.traceWriter)
}

func (d *registryImpl) ForComplianceOperatorRules() Dispatcher {
	return wrapDispatcher(d.complianceOperatorRulesDispatcher, d.traceWriter)
}

func (d *registryImpl) ForComplianceOperatorScanSettingBindings() Dispatcher {
	return wrapDispatcher(d.complianceOperatorScanSettingBindingsDispatcher, d.traceWriter)
}

func (d *registryImpl) ForComplianceOperatorScans() Dispatcher {
	return wrapDispatcher(d.complianceOperatorScanDispatcher, d.traceWriter)
}

func (d *registryImpl) ForRegistryMirrors() Dispatcher {
	return wrapDispatcher(d.osRegistryMirrorDispatcher, d.traceWriter)
}
