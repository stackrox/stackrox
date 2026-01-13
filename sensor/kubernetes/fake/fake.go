package fake

import (
	"context"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/cockroachdb/pebble/v2"
	appVersioned "github.com/openshift/client-go/apps/clientset/versioned"
	configVersioned "github.com/openshift/client-go/config/clientset/versioned"
	operatorVersioned "github.com/openshift/client-go/operator/clientset/versioned"
	routeVersioned "github.com/openshift/client-go/route/clientset/versioned"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/fixtures/vmindexreport"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	vmPkg "github.com/stackrox/rox/pkg/virtualmachine"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/networkflow/manager"
	"github.com/stackrox/rox/sensor/common/signal"
	"github.com/stackrox/rox/sensor/common/virtualmachine/index"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	vmStore "github.com/stackrox/rox/sensor/kubernetes/listener/resources/virtualmachine/store"
	"go.yaml.in/yaml/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apimachinery/pkg/watch"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/dynamic"
	fakeDynamic "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	workloadPath = "/var/scale/stackrox/workload.yaml"

	defaultNamespaceNum = 30

	// Starting CID for VM population. This is used as a part of the name and its value does not matter
	// as long as it is unique and different than 0, 1, and 2 (reserved values).
	vmBaseVSOCKCID = uint32(1000)

	// reportGeneratorSeed is the seed for deterministic package selection in VM index reports.
	reportGeneratorSeed = int64(42)
)

// vmReadiness encapsulates the three readiness signals needed before VM workload can start
type vmReadiness struct {
	handlerReady concurrency.Signal
	storeReady   concurrency.Signal
	centralReady concurrency.Signal
}

func newVMReadiness() *vmReadiness {
	return &vmReadiness{
		handlerReady: concurrency.NewSignal(),
		storeReady:   concurrency.NewSignal(),
		centralReady: concurrency.NewSignal(),
	}
}

func (r *vmReadiness) signalHandlerReady() { r.handlerReady.Signal() }
func (r *vmReadiness) signalStoreReady()   { r.storeReady.Signal() }
func (r *vmReadiness) signalCentralReady() { r.centralReady.Signal() }
func (r *vmReadiness) resetCentralReady()  { r.centralReady.Reset() }

// Wait blocks until all three signals are ready. Returns true if all ready, false if context cancelled.
func (r *vmReadiness) Wait(ctx context.Context) bool {
	if !concurrency.WaitInContext(&r.handlerReady, ctx) {
		return false
	}
	if !concurrency.WaitInContext(&r.storeReady, ctx) {
		return false
	}
	if !concurrency.WaitInContext(&r.centralReady, ctx) {
		return false
	}
	return true
}

var (
	log = logging.LoggerForModule()
)

func init() {
	// This needs to be increased in order to prevent the fake watcher from panicking.
	// Note that as this is a global variable, it _must_ be set in an init() in order to
	// ensure race-freeness. While it may look weird that we are setting this unconditionally
	// whenever the fake package is imported (including in prod), this doesn't hurt and is
	// actually WAI since `DefaultChanSize` is only applied to *fake* watchers in the first
	// place, even if that is not at all apparent from the name.
	watch.DefaultChanSize = 100000
}

// clientSetImpl implements our client.Interface
type clientSetImpl struct {
	kubernetes        kubernetes.Interface
	dynamic           dynamic.Interface
	openshiftApps     appVersioned.Interface
	openshiftConfig   configVersioned.Interface
	openshiftRoute    routeVersioned.Interface
	openshiftOperator operatorVersioned.Interface
}

// Kubernetes returns the fake Kubernetes clientset
func (c *clientSetImpl) Kubernetes() kubernetes.Interface {
	return c.kubernetes
}

// OpenshiftApps returns the fake openshift client for apps
func (c *clientSetImpl) OpenshiftApps() appVersioned.Interface {
	return c.openshiftApps
}

// OpenshiftConfig returns the fake openshift client for config
func (c *clientSetImpl) OpenshiftConfig() configVersioned.Interface {
	return c.openshiftConfig
}

// Dynamic returns the fake dynamic client
func (c *clientSetImpl) Dynamic() dynamic.Interface {
	return c.dynamic
}

// OpenshiftRoute returns the fake openshift client for route
func (c *clientSetImpl) OpenshiftRoute() routeVersioned.Interface {
	return c.openshiftRoute
}

// OpenshiftOperator returns the fake openshift client for operator
func (c *clientSetImpl) OpenshiftOperator() operatorVersioned.Interface {
	return c.openshiftOperator
}

// WorkloadManager encapsulates running a fake Kubernetes client
type WorkloadManager struct {
	db                        *pebble.DB
	fakeClient                *fake.Clientset
	dynamicClient             *fakeDynamic.FakeDynamicClient
	client                    client.Interface
	processPool               *ProcessPool
	labelsPool                *labelsPoolPerNamespace
	endpointPool              *EndpointPool
	ipPool                    *pool
	externalIpPool            *pool
	containerPool             *pool
	registeredHostConnections []manager.HostNetworkInfo
	workload                  *Workload
	originatorCache           *OriginatorCache

	// signals services
	servicesInitialized  concurrency.Signal
	processes            signal.Pipeline
	networkManager       manager.Manager
	vmIndexReportHandler index.Handler
	vmStore              *vmStore.VirtualMachineStore

	// VM readiness coordinator
	vmPrerequisitesReady *vmReadiness

	// shutdown coordination
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc
	wg             sync.WaitGroup
}

// WorkloadManagerConfig WorkloadManager's configuration
type WorkloadManagerConfig struct {
	workloadFile   string
	labelsPool     *labelsPoolPerNamespace
	processPool    *ProcessPool
	endpointPool   *EndpointPool
	ipPool         *pool
	externalIpPool *pool
	containerPool  *pool
	storagePath    string
}

// ConfigDefaults default configuration
func ConfigDefaults() *WorkloadManagerConfig {
	return &WorkloadManagerConfig{
		workloadFile:   workloadPath,
		labelsPool:     newLabelsPool(),
		processPool:    newProcessPool(),
		endpointPool:   newEndpointPool(),
		ipPool:         newPool(),
		externalIpPool: newPool(),
		containerPool:  newPool(),
		storagePath:    env.FakeWorkloadStoragePath.Setting(),
	}
}

// WithWorkloadFile configures the WorkloadManagerConfig's WorkloadFile field
func (c *WorkloadManagerConfig) WithWorkloadFile(file string) *WorkloadManagerConfig {
	c.workloadFile = file
	return c
}

// WithLabelsPool configures the WorkloadManagerConfig's LabelsPool field
func (c *WorkloadManagerConfig) WithLabelsPool(pool *labelsPoolPerNamespace) *WorkloadManagerConfig {
	c.labelsPool = pool
	return c
}

// WithProcessPool configures the WorkloadManagerConfig's ProcessPool field
func (c *WorkloadManagerConfig) WithProcessPool(pool *ProcessPool) *WorkloadManagerConfig {
	c.processPool = pool
	return c
}

// WithEndpointPool configures the WorkloadManagerConfig's EndpointPool field
func (c *WorkloadManagerConfig) WithEndpointPool(pool *EndpointPool) *WorkloadManagerConfig {
	c.endpointPool = pool
	return c
}

// WithIpPool configures the WorkloadManagerConfig's IpPool field
func (c *WorkloadManagerConfig) WithIpPool(pool *pool) *WorkloadManagerConfig {
	c.ipPool = pool
	return c
}

// WithExternalIpPool configures the WorkloadManagerConfig's ExternalIpPool field
func (c *WorkloadManagerConfig) WithExternalIpPool(pool *pool) *WorkloadManagerConfig {
	c.externalIpPool = pool
	return c
}

// WithContainerPool configures the WorkloadManagerConfig's ContainerPool field
func (c *WorkloadManagerConfig) WithContainerPool(pool *pool) *WorkloadManagerConfig {
	c.containerPool = pool
	return c
}

// WithStoragePath configures the WorkloadManagerConfig's StoragePath field
func (c *WorkloadManagerConfig) WithStoragePath(path string) *WorkloadManagerConfig {
	c.storagePath = path
	return c
}

// Client returns the mock client
func (w *WorkloadManager) Client() client.Interface {
	return w.client
}

// NewWorkloadManager returns a fake kubernetes client interface that will be managed with the passed Workload
func NewWorkloadManager(config *WorkloadManagerConfig) *WorkloadManager {
	data, err := os.ReadFile(config.workloadFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		log.Debugf("error opening fake scale workload config: %v", err)
		return nil
	}
	var workload Workload
	if err := yaml.Unmarshal(data, &workload); err != nil {
		log.Panicf("could not unmarshal workload from file due to error (%v): %s", err, data)
	}
	var warning error
	workload.VirtualMachineWorkload, warning = validateVMWorkload(workload.VirtualMachineWorkload)
	if warning != nil {
		log.Warnf("Validating workload: %s", warning)
	}

	var db *pebble.DB
	if config.storagePath != "" {
		db, err = pebble.Open(config.storagePath, &pebble.Options{})
		if err != nil {
			log.Panic("could not open id storage")
		}
	}
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	mgr := &WorkloadManager{
		db:                   db,
		workload:             &workload,
		originatorCache:      NewOriginatorCache(),
		labelsPool:           config.labelsPool,
		endpointPool:         config.endpointPool,
		ipPool:               config.ipPool,
		externalIpPool:       config.externalIpPool,
		containerPool:        config.containerPool,
		processPool:          config.processPool,
		servicesInitialized:  concurrency.NewSignal(),
		vmPrerequisitesReady: newVMReadiness(),
		shutdownCtx:          shutdownCtx,
		shutdownCancel:       shutdownCancel,
	}
	mgr.initializePreexistingResources()

	if warn := validateWorkload(&workload); warn != nil {
		log.Warnf("Validaing workload: %s", warn)
	}

	log.Info("Created Workload manager for workload")
	log.Infof("Workload: %s", string(data))
	log.Infof("Rendered workload: %+v", workload)
	return mgr
}

func validateWorkload(workload *Workload) error {
	if workload.NetworkWorkload.OpenPortReuseProbability < 0.0 || workload.NetworkWorkload.OpenPortReuseProbability > 1.0 {
		corrected := math.Min(1.0, math.Max(0.0, workload.NetworkWorkload.OpenPortReuseProbability))
		workload.NetworkWorkload.OpenPortReuseProbability = corrected
		return fmt.Errorf("incorrect probability value %.2f for 'openPortReuseProbability', "+
			"rounding to %.2f", workload.NetworkWorkload.OpenPortReuseProbability, corrected)
	}
	// More validation checks can be added in the future
	return nil
}

// SetSignalHandlers sets the handlers that will accept runtime data to be mocked from collector
func (w *WorkloadManager) SetSignalHandlers(processPipeline signal.Pipeline, networkManager manager.Manager) {
	w.processes = processPipeline
	w.networkManager = networkManager
	w.servicesInitialized.Signal()
}

// SetVMIndexReportHandler sets the handler that will accept VM index reports
func (w *WorkloadManager) SetVMIndexReportHandler(handler index.Handler) {
	w.vmIndexReportHandler = handler
	w.vmPrerequisitesReady.signalHandlerReady()
}

// SetVMStore sets the VirtualMachineStore
func (w *WorkloadManager) SetVMStore(store *vmStore.VirtualMachineStore) {
	log.Debugf("SetVMStore called: store=%p, poolSize=%d", store, w.workload.VirtualMachineWorkload.PoolSize)
	w.vmStore = store
	w.vmPrerequisitesReady.signalStoreReady()
	log.Debugf("SetVMStore completed (VMs will be populated by informer events)")
}

// Notify implements common.Notifiable to receive Sensor component event notifications
func (w *WorkloadManager) Notify(e common.SensorComponentEvent) {
	switch e {
	case common.SensorComponentEventCentralReachable:
		log.Debugf("WorkloadManager: Central is reachable, signaling VM workload can start")
		w.vmPrerequisitesReady.signalCentralReady()
	case common.SensorComponentEventOfflineMode:
		log.Debugf("WorkloadManager: Central went offline, resetting reachability signal")
		w.vmPrerequisitesReady.resetCentralReady()
	}
}

// Stop gracefully stops all background goroutines managed by WorkloadManager.
// This should be called before shutting down the process pipeline to prevent
// sending signals on closed channels.
// Stop waits for all background goroutines to exit before returning.
func (w *WorkloadManager) Stop() {
	if w.shutdownCancel != nil {
		w.shutdownCancel()
	}
	// Wait for all background goroutines to exit
	w.wg.Wait()
}

// clearActions periodically cleans up the fake client we're using. This needs to exist because we aren't
// using the client for its original purpose of unit testing. Essentially, it stores the actions
// so you can check which actions were run. We don't care about these actions so clear them every 10s
func (w *WorkloadManager) clearActions() {
	defer w.wg.Done()
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			w.fakeClient.ClearActions()
			if w.dynamicClient != nil {
				w.dynamicClient.ClearActions()
			}
		case <-w.shutdownCtx.Done():
			return
		}
	}
}

// cleanupVMHistory trims dynamic fake tracker state after a VM lifecycle ends to prevent runaway memory use.
func (w *WorkloadManager) cleanupVMHistory(namespace, vmName, vmiName string) {
	if w.dynamicClient == nil {
		return
	}

	tracker := w.dynamicClient.Tracker()
	if vmName != "" {
		_ = tracker.Delete(vmGVR, namespace, vmName)
	}
	if vmiName != "" {
		_ = tracker.Delete(vmiGVR, namespace, vmiName)
	}
	// Note: We intentionally don't call ClearActions() here as it would clear
	// the entire fake client's action history, including for non-VM resources.
}

func (w *WorkloadManager) initializePreexistingResources() {
	var objects []runtime.Object

	numNamespaces := defaultNamespaceNum
	if num := w.workload.NumNamespaces; num != 0 {
		numNamespaces = num
	}
	for _, n := range getNamespaces(numNamespaces, w.getIDsForPrefix(namespacePrefix)) {
		w.writeID(namespacePrefix, n.UID)
		objects = append(objects, n)
	}

	nodes := w.getNodes(w.workload.NodeWorkload, w.getIDsForPrefix(nodePrefix))
	for _, node := range nodes {
		w.writeID(nodePrefix, node.UID)
		objects = append(objects, node)
	}

	w.labelsPool.matchLabels = w.workload.MatchLabels

	objects = append(objects, w.getRBAC(w.workload.RBACWorkload, w.getIDsForPrefix(serviceAccountPrefix), w.getIDsForPrefix(rolesPrefix), w.getIDsForPrefix(rolebindingsPrefix))...)
	var resources []*deploymentResourcesToBeManaged

	deploymentIDs := w.getIDsForPrefix(deploymentPrefix)
	replicaSetIDs := w.getIDsForPrefix(replicaSetPrefix)
	podIDs := w.getIDsForPrefix(podPrefix)
	for _, deploymentWorkload := range w.workload.DeploymentWorkload {
		for i := 0; i < deploymentWorkload.NumDeployments; i++ {
			resource := w.getDeployment(deploymentWorkload, i, deploymentIDs, replicaSetIDs, podIDs)
			resources = append(resources, resource)

			objects = append(objects, resource.deployment, resource.replicaSet)
			for _, p := range resource.pods {
				objects = append(objects, p)
			}
		}
	}

	objects = append(objects, w.getServices(w.workload.ServiceWorkload, w.getIDsForPrefix(servicePrefix))...)
	var npResources []*networkPolicyToBeManaged
	networkPolicyIDs := w.getIDsForPrefix(networkPolicyPrefix)
	for _, npWorkload := range w.workload.NetworkPolicyWorkload {
		for i := 0; i < npWorkload.NumNetworkPolicies; i++ {
			resource := w.getNetworkPolicy(npWorkload, getID(networkPolicyIDs, i))
			w.writeID(networkPolicyPrefix, resource.networkPolicy.UID)
			npResources = append(npResources, resource)

			objects = append(objects, resource.networkPolicy)
		}
	}

	w.fakeClient = fake.NewClientset(objects...)
	w.fakeClient.Discovery().(*fakediscovery.FakeDiscovery).FakedServerVersion = &version.Info{
		Major:        "1",
		Minor:        "14",
		GitVersion:   "v1.14.8",
		GitCommit:    "211047e9a1922595eaa3a1127ed365e9299a6c23",
		GitTreeState: "clean",
		BuildDate:    "2019-10-15T12:02:12Z",
		GoVersion:    "go1.12.10",
		Compiler:     "gc",
		Platform:     "linux/amd64",
	}
	scheme := runtime.NewScheme()
	crdGVR := schema.GroupVersionResource{
		Group:    "apiextensions.k8s.io",
		Version:  "v1",
		Resource: "customresourcedefinitions",
	}

	// Use centralized GVRs from virtualmachines.go for kubevirt resources
	customListKinds := map[schema.GroupVersionResource]string{
		crdGVR: "CustomResourceDefinitionList",
		vmGVR:  "VirtualMachineList",
		vmiGVR: "VirtualMachineInstanceList",
	}

	dynClient := fakeDynamic.NewSimpleDynamicClientWithCustomListKinds(scheme, customListKinds)

	clientSet := &clientSetImpl{
		kubernetes: w.fakeClient,
		dynamic:    dynClient,
	}
	w.dynamicClient = dynClient

	// Seed discovery API with kubevirt resources if VM workload is configured
	if w.workload.VirtualMachineWorkload.PoolSize > 0 {
		fakeDiscovery := w.fakeClient.Discovery().(*fakediscovery.FakeDiscovery)
		vmGV := vmPkg.GetGroupVersion()
		vmResources := vmPkg.GetRequiredResources()

		// Add kubevirt API group to discovery
		apiResourceList := &metav1.APIResourceList{
			GroupVersion: vmGV.String(),
			APIResources: make([]metav1.APIResource, 0, len(vmResources)),
		}
		for _, res := range vmResources {
			apiResourceList.APIResources = append(apiResourceList.APIResources, res.APIResource)
		}
		fakeDiscovery.Resources = append(fakeDiscovery.Resources, apiResourceList)
	}

	initializeOpenshiftClients(clientSet)
	w.client = clientSet

	w.wg.Add(1)
	go w.clearActions()

	// Fork management of deployment resources
	for _, resource := range resources {
		w.wg.Add(1)
		go w.manageDeployment(w.shutdownCtx, resource)
	}

	// Fork management of networkPolicy resources
	for _, resource := range npResources {
		w.wg.Add(1)
		go w.manageNetworkPolicy(w.shutdownCtx, resource)
	}

	w.wg.Add(1)
	go w.manageFlows(w.shutdownCtx)

	// Start VirtualMachine/VirtualMachineInstance workload if configured.
	// This unified workload handles both informer events AND index reports.
	// Index reports are only sent while VMs are "alive" in the lifecycle.
	if w.workload.VirtualMachineWorkload.PoolSize > 0 {
		// Initialize report generator if index reports are enabled
		var reportGen *vmindexreport.Generator
		if w.workload.VirtualMachineWorkload.ReportInterval > 0 {
			reportGen = vmindexreport.NewGeneratorWithSeed(
				w.workload.VirtualMachineWorkload.NumPackages,
				reportGeneratorSeed,
			)
			log.Infof("VM index reports enabled: interval=%s, packages=%d",
				w.workload.VirtualMachineWorkload.ReportInterval,
				w.workload.VirtualMachineWorkload.NumPackages)
		}

		// Fork management of VM/VMI resources (including index reports if enabled)
		workload := w.workload.VirtualMachineWorkload
		for i := range workload.PoolSize {
			w.wg.Add(1)
			cid := vmBaseVSOCKCID + uint32(i)
			go w.manageVirtualMachine(w.shutdownCtx, workload, cid, reportGen)
		}
	}
}
