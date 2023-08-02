package fake

import (
	"context"
	"os"
	"time"

	"github.com/cockroachdb/pebble"
	appVersioned "github.com/openshift/client-go/apps/clientset/versioned"
	configVersioned "github.com/openshift/client-go/config/clientset/versioned"
	operatorVersioned "github.com/openshift/client-go/operator/clientset/versioned"
	routeVersioned "github.com/openshift/client-go/route/clientset/versioned"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/networkflow/manager"
	"github.com/stackrox/rox/sensor/common/signal"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apimachinery/pkg/watch"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	workloadPath = "/var/scale/stackrox/workload.yaml"

	defaultNamespaceNum = 30
)

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
	kubernetes kubernetes.Interface
}

// Kubernetes returns the fake Kubernetes clientset
func (c *clientSetImpl) Kubernetes() kubernetes.Interface {
	return c.kubernetes
}

// OpenshiftApps returns nil for the openshift client for config
func (c *clientSetImpl) OpenshiftApps() appVersioned.Interface {
	return nil
}

// OpenshiftConfig returns nil for the openshift client for apps
func (c *clientSetImpl) OpenshiftConfig() configVersioned.Interface {
	return nil
}

// Dynamic returns nil
func (c *clientSetImpl) Dynamic() dynamic.Interface {
	return nil
}

// OpenshiftRoute implements the client interface.
func (c *clientSetImpl) OpenshiftRoute() routeVersioned.Interface {
	return nil
}

// OpenshiftOperator implements the client interface.
func (c *clientSetImpl) OpenshiftOperator() operatorVersioned.Interface {
	return nil
}

// WorkloadManager encapsulates running a fake Kubernetes client
type WorkloadManager struct {
	db         *pebble.DB
	fakeClient *fake.Clientset
	client     client.Interface
	workload   *Workload

	// signals services
	servicesInitialized concurrency.Signal
	processes           signal.Pipeline
	networkManager      manager.Manager
}

// WorkloadManagerConfig WorkloadManager's configuration
type WorkloadManagerConfig struct {
	workloadFile string
}

// ConfigDefaults default configuration
func ConfigDefaults() *WorkloadManagerConfig {
	return &WorkloadManagerConfig{
		workloadFile: workloadPath,
	}
}

// WithWorkloadFile configures the WorkloadManagerConfig's WorkloadFile field
func (c *WorkloadManagerConfig) WithWorkloadFile(file string) *WorkloadManagerConfig {
	c.workloadFile = file
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

	var db *pebble.DB
	if storagePath := env.FakeWorkloadStoragePath.Setting(); storagePath != "" {
		db, err = pebble.Open(storagePath, &pebble.Options{})
		if err != nil {
			log.Panicf("could not open id storage")
		}
	}

	mgr := &WorkloadManager{
		db:                  db,
		workload:            &workload,
		servicesInitialized: concurrency.NewSignal(),
	}
	mgr.initializePreexistingResources()

	log.Infof("Created Workload manager for workload")
	log.Infof("Workload: %s", string(data))
	log.Infof("Rendered workload: %+v", workload)
	return mgr
}

// SetSignalHandlers sets the handlers that will accept runtime data to be mocked from collector
func (w *WorkloadManager) SetSignalHandlers(processPipeline signal.Pipeline, networkManager manager.Manager) {
	w.processes = processPipeline
	w.networkManager = networkManager
	w.servicesInitialized.Signal()
}

// clearActions periodically cleans up the fake client we're using. This needs to exist because we aren't
// using the client for its original purpose of unit testing. Essentially, it stores the actions
// so you can check which actions were run. We don't care about these actions so clear them every 10s
func (w *WorkloadManager) clearActions() {
	t := time.NewTicker(10 * time.Second)
	for range t.C {
		w.fakeClient.ClearActions()
	}
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

	labelsPool.matchLabels = w.workload.MatchLabels

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

	w.fakeClient = fake.NewSimpleClientset(objects...)
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
	w.client = &clientSetImpl{
		kubernetes: w.fakeClient,
	}

	go w.clearActions()

	// Fork management of deployment resources
	for _, resource := range resources {
		go w.manageDeployment(context.Background(), resource)
	}

	// Fork management of networkPolicy resources
	for _, resource := range npResources {
		go w.manageNetworkPolicy(context.Background(), resource)
	}

	go w.manageFlows(context.Background(), w.workload.NetworkWorkload)
}
