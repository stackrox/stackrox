package fake

import (
	"time"

	"github.com/openshift/client-go/apps/clientset/versioned"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/networkflow/manager"
	"github.com/stackrox/rox/sensor/common/signal"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	fake "github.com/stackrox/rox/sensor/kubernetes/fake/copied"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

var (
	log = logging.LoggerForModule()

	workloadRegistry = make(map[string]*workload)
)

// clientSetImpl implements our client.Interface
type clientSetImpl struct {
	kubernetes kubernetes.Interface
}

// Kubernetes returns the fake Kubernetes clientset
func (c *clientSetImpl) Kubernetes() kubernetes.Interface {
	return c.kubernetes
}

// Openshift returns nil for the openshift client
func (c *clientSetImpl) Openshift() versioned.Interface {
	return nil
}

// WorkloadManager encapsulates running a fake Kubernetes client
type WorkloadManager struct {
	fakeClient *fake.Clientset
	client     client.Interface
	workload   *workload

	// signals services
	processes      signal.Pipeline
	networkManager manager.Manager
}

// Client returns the mock client
func (w *WorkloadManager) Client() client.Interface {
	return w.client
}

// NewWorkloadManager returns a fake kubernetes client interface that will be managed with the passed workload
func NewWorkloadManager(workloadName string) *WorkloadManager {
	workload := workloadRegistry[workloadName]
	if workload == nil {
		log.Panicf("could not find workload with name %q", workloadName)
	}

	mgr := &WorkloadManager{
		workload: workload,
	}
	mgr.initializePreexistingResources()

	log.Infof("Created workload manager for workload %s", workloadName)
	return mgr
}

// SetSignalHandlers sets the handlers that will accept runtime data to be mocked from collector
func (w *WorkloadManager) SetSignalHandlers(processPipeline signal.Pipeline, networkManager manager.Manager) {
	w.processes = processPipeline
	w.networkManager = networkManager
}

// clearActions periodically cleans up the fake client we're using. This needs to exist because we aren't
// using the client for it's original purpose of unit testing. Essentially, it stores the actions
// so you can check which actions were run. We don't care about this actions so clear them every 10s
func (w *WorkloadManager) clearActions() {
	t := time.NewTicker(10 * time.Second)
	for range t.C {
		w.fakeClient.ClearActions()
	}
}

func (w *WorkloadManager) initializePreexistingResources() {
	var objects []runtime.Object

	namespace := getNamespace()
	namespace.Name = "default"
	objects = append(objects, namespace)

	nodes := w.getNodes(w.workload.NodeWorkload)
	for _, node := range nodes {
		objects = append(objects, node)
	}

	var resources []*deploymentResourcesToBeManaged
	for _, deploymentWorkload := range w.workload.DeploymentWorkload {
		for i := 0; i < deploymentWorkload.NumDeployments; i++ {
			resource := w.getDeployment(deploymentWorkload)
			resources = append(resources, resource)

			objects = append(objects, resource.deployment, resource.replicaSet)
			for _, p := range resource.pods {
				objects = append(objects, p)
			}
		}
	}

	w.fakeClient = fake.NewSimpleClientset(objects...)
	w.client = &clientSetImpl{
		kubernetes: w.fakeClient,
	}

	go w.clearActions()

	// Fork management of deployment resources
	for _, resource := range resources {
		go w.manageDeployment(resource)
	}

	go w.manageFlows(w.workload.NetworkWorkload)
}
