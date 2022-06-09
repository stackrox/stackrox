package k8s

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// CreateMode different types of creation modes for the FakeEventsManager
type CreateMode int

const (
	// Delay sleeps a specific duration between the creation of each event
	Delay CreateMode = iota
)

const (
	namespaceKind             string = "Namespace"
	secretKind                string = "Secret"
	serviceAccountsKind       string = "ServiceAccount"
	roleKind                  string = "Role"
	clusterRoleKind           string = "ClusterRole"
	roleBindingKind           string = "RoleBinding"
	clusterRoleBindingKind    string = "ClusterRoleBinding"
	networkPolicyKind         string = "NetworkPolicy"
	nodeKind                  string = "Node"
	serviceKind               string = "Service"
	jobKind                   string = "Job"
	replicaSetKind            string = "ReplicaSet"
	replicationControllerKind string = "ReplicationController"
	daemonSetKind             string = "DaemonSet"
	deploymentKind            string = "Deployment"
	statefulSetKind           string = "StatefulSet"
	cronJobKind               string = "CronJob"
	podKind                   string = "Pod"
)

var minimumResources = map[string]int{
	namespaceKind: 1,
	nodeKind:      1,
}

// FakeEventsManager reads k8s events from a jsonl file and creates reproduces them
type FakeEventsManager struct {
	// Delay the sleep duration between the creation of each event (if CreteMode is Delay)
	Delay time.Duration
	// Mode the creation mode (at the moment there is only one mode implemented)
	Mode CreateMode
	// Client the k8s ClientSet
	Client *ClientSet
	// Reader the TraceReader
	Reader *TraceReader
	// clientMap map with the k8s clients
	clientMap map[string]func(string) interface{}
	// resourceMap map with the k8s resources
	resourceMap map[string]interface{}
}

var actionToMethod = map[string]string{
	"CREATE_RESOURCE": "Create",
	"UPDATE_RESOURCE": "Update",
	"DELETE_RESOURCE": "Delete",
}

var actionToOptions = map[string]interface{}{
	"CREATE_RESOURCE": metav1.CreateOptions{},
	"UPDATE_RESOURCE": metav1.UpdateOptions{},
	"DELETE_RESOURCE": metav1.DeleteOptions{},
}

// Init initializes the FakeEventsManager
func (f *FakeEventsManager) Init() {
	f.clientMap = map[string]func(string) interface{}{
		namespaceKind:          func(string) interface{} { return f.Client.Kubernetes().CoreV1().Namespaces() },
		clusterRoleKind:        func(string) interface{} { return f.Client.Kubernetes().RbacV1().ClusterRoles() },
		clusterRoleBindingKind: func(string) interface{} { return f.Client.Kubernetes().RbacV1().ClusterRoleBindings() },
		nodeKind:               func(string) interface{} { return f.Client.Kubernetes().CoreV1().Nodes() },
		secretKind:             func(namespace string) interface{} { return f.Client.Kubernetes().CoreV1().Secrets(namespace) },
		serviceAccountsKind:    func(namespace string) interface{} { return f.Client.Kubernetes().CoreV1().ServiceAccounts(namespace) },
		roleKind:               func(namespace string) interface{} { return f.Client.Kubernetes().RbacV1().Roles(namespace) },
		roleBindingKind:        func(namespace string) interface{} { return f.Client.Kubernetes().RbacV1().RoleBindings(namespace) },
		networkPolicyKind: func(namespace string) interface{} {
			return f.Client.Kubernetes().NetworkingV1().NetworkPolicies(namespace)
		},
		serviceKind:    func(namespace string) interface{} { return f.Client.Kubernetes().CoreV1().Services(namespace) },
		jobKind:        func(namespace string) interface{} { return f.Client.Kubernetes().BatchV1().Jobs(namespace) },
		replicaSetKind: func(namespace string) interface{} { return f.Client.Kubernetes().AppsV1().ReplicaSets(namespace) },
		replicationControllerKind: func(namespace string) interface{} {
			return f.Client.Kubernetes().CoreV1().ReplicationControllers(namespace)
		},
		daemonSetKind:   func(namespace string) interface{} { return f.Client.Kubernetes().AppsV1().DaemonSets(namespace) },
		deploymentKind:  func(namespace string) interface{} { return f.Client.Kubernetes().AppsV1().Deployments(namespace) },
		statefulSetKind: func(namespace string) interface{} { return f.Client.Kubernetes().AppsV1().StatefulSets(namespace) },
		cronJobKind:     func(namespace string) interface{} { return f.Client.Kubernetes().BatchV1().CronJobs(namespace) },
		podKind:         func(namespace string) interface{} { return f.Client.Kubernetes().CoreV1().Pods(namespace) },
	}
	f.resourceMap = map[string]interface{}{
		namespaceKind:             &corev1.Namespace{},
		secretKind:                &corev1.Secret{},
		serviceAccountsKind:       &corev1.ServiceAccount{},
		roleKind:                  &rbacv1.Role{},
		clusterRoleKind:           &rbacv1.ClusterRole{},
		roleBindingKind:           &rbacv1.RoleBinding{},
		clusterRoleBindingKind:    &rbacv1.ClusterRoleBinding{},
		networkPolicyKind:         &networkingv1.NetworkPolicy{},
		nodeKind:                  &corev1.Node{},
		serviceKind:               &corev1.Service{},
		jobKind:                   &batchv1.Job{},
		replicaSetKind:            &appsv1.ReplicaSet{},
		replicationControllerKind: &corev1.ReplicationController{},
		daemonSetKind:             &appsv1.DaemonSet{},
		deploymentKind:            &appsv1.Deployment{},
		statefulSetKind:           &appsv1.StatefulSet{},
		cronJobKind:               &batchv1.CronJob{},
		podKind:                   &corev1.Pod{},
	}
}

// waitForMinimumResources waits for a minimum number of resources to be created or once all the events have been processed
func waitForMinimumResources(ch chan string, done concurrency.Signal) error {
	count := 0
	doneC := done.WaitC()
	for {
		select {
		case obj := <-ch:
			if _, ok := minimumResources[obj]; ok {
				minimumResources[obj]--
				if minimumResources[obj] == 0 {
					count++
				}
				if len(minimumResources) == count {
					return nil
				}
			}
		case <-doneC:
			return errors.New("the events file did not contain the minimum resources required to start sensor")
		}
	}
}

// CreateEvents creates the k8s events from a given jsonl file
func (f *FakeEventsManager) CreateEvents() error {
	ch := make(chan string)
	done := concurrency.NewSignal()
	objs, err := f.Reader.ReadFile()
	if err != nil {
		return err
	}
	f.Init()
	go func() {
		for _, obj := range objs {
			if len(obj) == 0 {
				continue
			}
			msg := resources.InformerK8sMsg{}
			if err := json.Unmarshal(obj, &msg); err != nil {
				log.Fatalln(err)
			}
			log.Printf("%s Event: %s", msg.Action, msg.ObjectType)
			if err := f.createEvent(msg, ch); err != nil {
				log.Fatalf("cannot create event for %s: %s", msg.ObjectType, err)
			}
		}
		done.Signal()
	}()
	return waitForMinimumResources(ch, done)
}

// runOp runs the create/update/delete operation
func runOp(action string, resourceClient, resourceObject reflect.Value) []reflect.Value {
	method, ok := actionToMethod[action]
	if !ok {
		log.Fatalf("method not found")
	}
	options, ok := actionToOptions[action]
	if !ok {
		log.Fatalf("options not found")
	}
	return resourceClient.MethodByName(method).Call([]reflect.Value{
		reflect.ValueOf(context.Background()),
		resourceObject,
		reflect.ValueOf(options),
	})
}

// getNamespace returns the namespace from a resource
func getNamespace(resource reflect.Value) string {
	values := resource.MethodByName("GetNamespace").Call([]reflect.Value{})
	if len(values) != 1 {
		return ""
	}
	return values[0].String()
}

// handleRunOp handles the execution of runOp
func (f *FakeEventsManager) handleRunOp(action, kind string, client, object reflect.Value, ch chan string) error {
	returnVals := runOp(action, client, object)
	if len(returnVals) == 0 {
		return fmt.Errorf("expected 1 or 2 values from %s. Received: %d", action, len(returnVals))
	}
	errInt := returnVals[len(returnVals)-1].Interface()
	if errInt == nil {
		select {
		case ch <- kind:
		default:
		}
		f.waitOnMode()
		return nil
	}
	return errInt.(error)
}

// createEvent creates a single k8s event
func (f *FakeEventsManager) createEvent(msg resources.InformerK8sMsg, ch chan string) error {
	obj := &unstructured.Unstructured{}
	objType := strings.Split(msg.ObjectType, ".")
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&msg.Payload)
	if err != nil {
		return fmt.Errorf("error constructing unstructured: %w", err)
	}
	Kind := objType[1]
	obj.Object = u

	r, ok := f.resourceMap[Kind]
	if !ok {
		return fmt.Errorf("resource %s not found", Kind)
	}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), r); err != nil {
		return fmt.Errorf("unable to construct %s from unstructured: %w", Kind, err)
	}

	clFunc, ok := f.clientMap[Kind]
	if !ok {
		return fmt.Errorf("resource %s not found", Kind)
	}
	cl := clFunc(getNamespace(reflect.ValueOf(r)))

	err = f.handleRunOp(msg.Action, Kind, reflect.ValueOf(cl), reflect.ValueOf(r), ch)
	if k8sErrors.IsNotFound(err) && msg.Action == "UPDATE_RESOURCE" {
		return f.handleRunOp("CREATE_RESOURCE", Kind, reflect.ValueOf(cl), reflect.ValueOf(r), ch)
	}
	if k8sErrors.IsAlreadyExists(err) && msg.Action == "CREATE_RESOURCE" {
		return f.handleRunOp("UPDATE_RESOURCE", Kind, reflect.ValueOf(cl), reflect.ValueOf(r), ch)
	}
	return err
}

// waitOnMode waits depending on the mode
func (f *FakeEventsManager) waitOnMode() {
	switch f.Mode {
	case Delay:
		time.Sleep(f.Delay)
	}
}
