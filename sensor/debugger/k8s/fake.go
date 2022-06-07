package k8s

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

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

type CreateMode int

const (
	Timestamps CreateMode = iota
	Delay
)

const (
	NamespaceKind             string = "Namespace"
	SecretKind                string = "Secret"
	ServiceAccountsKind       string = "ServiceAccount"
	RoleKind                  string = "Role"
	ClusterRoleKind           string = "ClusterRole"
	RoleBindingKind           string = "RoleBinding"
	ClusterRoleBindingKind    string = "ClusterRoleBinding"
	NetworkPolicyKind         string = "NetworkPolicy"
	NodeKind                  string = "Node"
	ServiceKind               string = "Service"
	JobKind                   string = "Job"
	ReplicaSetKind            string = "ReplicaSet"
	ReplicationControllerKind string = "ReplicationController"
	DaemonSetKind             string = "DaemonSet"
	DeploymentKind            string = "Deployment"
	StatefulSetKind           string = "StatefulSet"
	CronJobKind               string = "CronJob"
	PodKind                   string = "Pod"
)

var minimumResources = map[string]int{
	NamespaceKind: 1,
	NodeKind:      1,
}

// FakeEventsManager reads k8s events from a jsonl file and creates reproduces them
type FakeEventsManager struct {
	Delay  time.Duration
	Mode   CreateMode
	Client *ClientSet
	Reader *TraceReader
}

// WaitForMinimumResources waits for a minimum number of resources to be created or once all the events have been processed
func WaitForMinimumResources(ch chan string, done chan int, readerError chan error) error {
	count := 0
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
		case err := <-readerError:
			return err
		case <-done:
			return errors.New("the events file did not contain the minimum resources required to start sensor")
		}
	}
}

// executeAction executes an action and waits depending on the Mode
func (f *FakeEventsManager) executeAction(action string, kind string, ch chan string, create, update, delete func() error) error {
	switch action {
	case "CREATE_RESOURCE":
		err := create()
		if k8sErrors.IsAlreadyExists(err) {
			err := update()
			if err != nil {
				return err
			}
			f.waitOnMode()
			return nil
		}
		if err != nil {
			return err
		}
		select {
		case ch <- kind:
		default:
		}
		f.waitOnMode()
		return nil
	case "UPDATE_RESOURCE":
		err := update()
		if k8sErrors.IsNotFound(err) {
			err := create()
			if err != nil {
				return err
			}
			select {
			case ch <- kind:
			default:
			}
			f.waitOnMode()
			return nil
		}
		if err != nil {
			return err
		}
		f.waitOnMode()
		return nil
	case "DELETE_RESOURCE":
		err := delete()
		if err != nil {
			return err
		}
		f.waitOnMode()
		return nil
	}
	return errors.New("unknown action")
}

// CreateEvents creates the k8s events from a given jsonl file
func (f *FakeEventsManager) CreateEvents() error {
	ch := make(chan string)
	done := make(chan int)
	readerError := make(chan error)
	go f.Reader.ReadFile(f.Mode, done, readerError, func(line []byte, m CreateMode) {
		obj := resources.InformerK8sMsg{}
		if err := json.Unmarshal(line, &obj); err != nil {
			log.Fatalf("cannot unmarshal: %s\n", err)
		}
		log.Printf("%s Event: %s", obj.Action, obj.ObjectType)
		if err := f.createEvent(obj, ch); err != nil {
			log.Fatalf("cannot create event for %s %s", obj.ObjectType, err)
		}
	})
	return WaitForMinimumResources(ch, done, readerError)
}

// createEvent creates a single k8s event
func (f *FakeEventsManager) createEvent(msg resources.InformerK8sMsg, ch chan string) error {
	obj := &unstructured.Unstructured{}
	objType := strings.Split(msg.ObjectType, ".")
	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&msg.Payload)
	if err != nil {
		log.Printf("error constructing unstructured: %s", err)
		return err
	}
	Kind := objType[1]
	obj.Object = u

	switch Kind {
	case NamespaceKind:
		var r corev1.Namespace
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &r); err != nil {
			log.Printf("Unable to construct %s from unstructured", Kind)
			return err
		}
		return f.executeAction(msg.Action, Kind, ch, func() error {
			_, err := f.Client.Kubernetes().CoreV1().Namespaces().Create(context.Background(), &r, metav1.CreateOptions{})
			return err
		}, func() error {
			_, err := f.Client.Kubernetes().CoreV1().Namespaces().Update(context.Background(), &r, metav1.UpdateOptions{})
			return err
		}, func() error {
			err := f.Client.Kubernetes().CoreV1().Namespaces().Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
			return err
		})
	case SecretKind:
		var r corev1.Secret
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &r); err != nil {
			log.Printf("Unable to construct %s from unstructured", Kind)
			return err
		}
		return f.executeAction(msg.Action, Kind, ch, func() error {
			_, err := f.Client.Kubernetes().CoreV1().Secrets(obj.GetNamespace()).Create(context.Background(), &r, metav1.CreateOptions{})
			return err
		}, func() error {
			_, err := f.Client.Kubernetes().CoreV1().Secrets(obj.GetNamespace()).Update(context.Background(), &r, metav1.UpdateOptions{})
			return err
		}, func() error {
			return f.Client.Kubernetes().CoreV1().Secrets(obj.GetNamespace()).Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
		})
	case ServiceAccountsKind:
		var r corev1.ServiceAccount
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &r); err != nil {
			log.Printf("Unable to construct %s from unstructured", Kind)
			return err
		}
		return f.executeAction(msg.Action, Kind, ch, func() error {
			_, err := f.Client.Kubernetes().CoreV1().ServiceAccounts(obj.GetNamespace()).Create(context.Background(), &r, metav1.CreateOptions{})
			return err
		}, func() error {
			_, err := f.Client.Kubernetes().CoreV1().ServiceAccounts(obj.GetNamespace()).Update(context.Background(), &r, metav1.UpdateOptions{})
			return err
		}, func() error {
			return f.Client.Kubernetes().CoreV1().ServiceAccounts(obj.GetNamespace()).Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
		})
	case RoleKind:
		var r rbacv1.Role
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &r); err != nil {
			log.Printf("Unable to construct %s from unstructured", Kind)
			return err
		}
		return f.executeAction(msg.Action, Kind, ch, func() error {
			_, err := f.Client.Kubernetes().RbacV1().Roles(obj.GetNamespace()).Create(context.Background(), &r, metav1.CreateOptions{})
			return err
		}, func() error {
			_, err := f.Client.Kubernetes().RbacV1().Roles(obj.GetNamespace()).Update(context.Background(), &r, metav1.UpdateOptions{})
			return err
		}, func() error {
			return f.Client.Kubernetes().RbacV1().Roles(obj.GetNamespace()).Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
		})
	case RoleBindingKind:
		var r rbacv1.RoleBinding
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &r); err != nil {
			log.Printf("Unable to construct %s from unstructured", Kind)
			return err
		}
		return f.executeAction(msg.Action, Kind, ch, func() error {
			_, err := f.Client.Kubernetes().RbacV1().RoleBindings(obj.GetNamespace()).Create(context.Background(), &r, metav1.CreateOptions{})
			return err
		}, func() error {
			_, err := f.Client.Kubernetes().RbacV1().RoleBindings(obj.GetNamespace()).Update(context.Background(), &r, metav1.UpdateOptions{})
			return err
		}, func() error {
			return f.Client.Kubernetes().RbacV1().RoleBindings(obj.GetNamespace()).Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
		})
	case ClusterRoleKind:
		var r rbacv1.ClusterRole
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &r); err != nil {
			log.Printf("Unable to construct %s from unstructured", Kind)
			return err
		}
		return f.executeAction(msg.Action, Kind, ch, func() error {
			_, err := f.Client.Kubernetes().RbacV1().ClusterRoles().Create(context.Background(), &r, metav1.CreateOptions{})
			return err
		}, func() error {
			_, err := f.Client.Kubernetes().RbacV1().ClusterRoles().Update(context.Background(), &r, metav1.UpdateOptions{})
			return err
		}, func() error {
			return f.Client.Kubernetes().RbacV1().ClusterRoles().Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
		})
	case ClusterRoleBindingKind:
		var r rbacv1.ClusterRoleBinding
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &r); err != nil {
			log.Printf("Unable to construct %s from unstructured", Kind)
			return err
		}
		return f.executeAction(msg.Action, Kind, ch, func() error {
			_, err := f.Client.Kubernetes().RbacV1().ClusterRoleBindings().Create(context.Background(), &r, metav1.CreateOptions{})
			return err
		}, func() error {
			_, err := f.Client.Kubernetes().RbacV1().ClusterRoleBindings().Update(context.Background(), &r, metav1.UpdateOptions{})
			return err
		}, func() error {
			return f.Client.Kubernetes().RbacV1().ClusterRoleBindings().Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
		})
	case NetworkPolicyKind:
		var r networkingv1.NetworkPolicy
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &r); err != nil {
			log.Printf("Unable to construct %s from unstructured", Kind)
			return err
		}
		return f.executeAction(msg.Action, Kind, ch, func() error {
			_, err := f.Client.Kubernetes().NetworkingV1().NetworkPolicies(obj.GetNamespace()).Create(context.Background(), &r, metav1.CreateOptions{})
			return err
		}, func() error {
			_, err := f.Client.Kubernetes().NetworkingV1().NetworkPolicies(obj.GetNamespace()).Update(context.Background(), &r, metav1.UpdateOptions{})
			return err
		}, func() error {
			return f.Client.Kubernetes().NetworkingV1().NetworkPolicies(obj.GetNamespace()).Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
		})
	case NodeKind:
		var r corev1.Node
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &r); err != nil {
			log.Printf("Unable to construct %s from unstructured", Kind)
			return err
		}
		return f.executeAction(msg.Action, Kind, ch, func() error {
			_, err := f.Client.Kubernetes().CoreV1().Nodes().Create(context.Background(), &r, metav1.CreateOptions{})
			return err
		}, func() error {
			_, err := f.Client.Kubernetes().CoreV1().Nodes().Update(context.Background(), &r, metav1.UpdateOptions{})
			return err
		}, func() error {
			return f.Client.Kubernetes().CoreV1().Nodes().Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
		})
	case ServiceKind:
		var r corev1.Service
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &r); err != nil {
			log.Printf("Unable to construct %s from unstructured", Kind)
			return err
		}
		return f.executeAction(msg.Action, Kind, ch, func() error {
			_, err := f.Client.Kubernetes().CoreV1().Services(obj.GetNamespace()).Create(context.Background(), &r, metav1.CreateOptions{})
			return err
		}, func() error {
			_, err := f.Client.Kubernetes().CoreV1().Services(obj.GetNamespace()).Update(context.Background(), &r, metav1.UpdateOptions{})
			return err
		}, func() error {
			return f.Client.Kubernetes().CoreV1().Services(obj.GetNamespace()).Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
		})
	case JobKind:
		var r batchv1.Job
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &r); err != nil {
			log.Printf("Unable to construct %s from unstructured", Kind)
			return err
		}
		return f.executeAction(msg.Action, Kind, ch, func() error {
			_, err := f.Client.Kubernetes().BatchV1().Jobs(obj.GetNamespace()).Create(context.Background(), &r, metav1.CreateOptions{})
			return err
		}, func() error {
			_, err := f.Client.Kubernetes().BatchV1().Jobs(obj.GetNamespace()).Update(context.Background(), &r, metav1.UpdateOptions{})
			return err
		}, func() error {
			return f.Client.Kubernetes().BatchV1().Jobs(obj.GetNamespace()).Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
		})
	case ReplicaSetKind:
		var r appsv1.ReplicaSet
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &r); err != nil {
			log.Printf("Unable to construct %s from unstructured", Kind)
			return err
		}
		return f.executeAction(msg.Action, Kind, ch, func() error {
			_, err := f.Client.Kubernetes().AppsV1().ReplicaSets(obj.GetNamespace()).Create(context.Background(), &r, metav1.CreateOptions{})
			return err
		}, func() error {
			_, err := f.Client.Kubernetes().AppsV1().ReplicaSets(obj.GetNamespace()).Update(context.Background(), &r, metav1.UpdateOptions{})
			return err
		}, func() error {
			return f.Client.Kubernetes().AppsV1().ReplicaSets(obj.GetNamespace()).Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
		})
	case ReplicationControllerKind:
		var r corev1.ReplicationController
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &r); err != nil {
			log.Printf("Unable to construct %s from unstructured", Kind)
			return err
		}
		return f.executeAction(msg.Action, Kind, ch, func() error {
			_, err := f.Client.Kubernetes().CoreV1().ReplicationControllers(obj.GetNamespace()).Create(context.Background(), &r, metav1.CreateOptions{})
			return err
		}, func() error {
			_, err := f.Client.Kubernetes().CoreV1().ReplicationControllers(obj.GetNamespace()).Update(context.Background(), &r, metav1.UpdateOptions{})
			return err
		}, func() error {
			return f.Client.Kubernetes().CoreV1().ReplicationControllers(obj.GetNamespace()).Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
		})
	case DaemonSetKind:
		var r appsv1.DaemonSet
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &r); err != nil {
			log.Printf("Unable to construct %s from unstructured", Kind)
			return err
		}
		return f.executeAction(msg.Action, Kind, ch, func() error {
			_, err := f.Client.Kubernetes().AppsV1().DaemonSets(obj.GetNamespace()).Create(context.Background(), &r, metav1.CreateOptions{})
			return err
		}, func() error {
			_, err := f.Client.Kubernetes().AppsV1().DaemonSets(obj.GetNamespace()).Update(context.Background(), &r, metav1.UpdateOptions{})
			return err
		}, func() error {
			return f.Client.Kubernetes().AppsV1().DaemonSets(obj.GetNamespace()).Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
		})
	case DeploymentKind:
		var r appsv1.Deployment
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &r); err != nil {
			log.Printf("Unable to construct %s from unstructured", Kind)
			return err
		}
		return f.executeAction(msg.Action, Kind, ch, func() error {
			_, err := f.Client.Kubernetes().AppsV1().Deployments(obj.GetNamespace()).Create(context.Background(), &r, metav1.CreateOptions{})
			return err
		}, func() error {
			_, err := f.Client.Kubernetes().AppsV1().Deployments(obj.GetNamespace()).Update(context.Background(), &r, metav1.UpdateOptions{})
			return err
		}, func() error {
			return f.Client.Kubernetes().AppsV1().Deployments(obj.GetNamespace()).Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
		})
	case StatefulSetKind:
		var r appsv1.StatefulSet
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &r); err != nil {
			log.Printf("Unable to construct %s from unstructured", Kind)
			return err
		}
		return f.executeAction(msg.Action, Kind, ch, func() error {
			_, err := f.Client.Kubernetes().AppsV1().StatefulSets(obj.GetNamespace()).Create(context.Background(), &r, metav1.CreateOptions{})
			return err
		}, func() error {
			_, err := f.Client.Kubernetes().AppsV1().StatefulSets(obj.GetNamespace()).Update(context.Background(), &r, metav1.UpdateOptions{})
			return err
		}, func() error {
			return f.Client.Kubernetes().AppsV1().StatefulSets(obj.GetNamespace()).Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
		})
	case CronJobKind:
		var r batchv1.CronJob
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &r); err != nil {
			log.Printf("Unable to construct %s from unstructured", Kind)
			return err
		}
		return f.executeAction(msg.Action, Kind, ch, func() error {
			_, err := f.Client.Kubernetes().BatchV1().CronJobs(obj.GetNamespace()).Create(context.Background(), &r, metav1.CreateOptions{})
			return err
		}, func() error {
			_, err := f.Client.Kubernetes().BatchV1().CronJobs(obj.GetNamespace()).Update(context.Background(), &r, metav1.UpdateOptions{})
			return err
		}, func() error {
			return f.Client.Kubernetes().BatchV1().CronJobs(obj.GetNamespace()).Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
		})
	case PodKind:
		var r corev1.Pod
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), &r); err != nil {
			log.Printf("Unable to construct %s from unstructured", Kind)
			return err
		}
		return f.executeAction(msg.Action, Kind, ch, func() error {
			_, err := f.Client.Kubernetes().CoreV1().Pods(obj.GetNamespace()).Create(context.Background(), &r, metav1.CreateOptions{})
			return err
		}, func() error {
			_, err := f.Client.Kubernetes().CoreV1().Pods(obj.GetNamespace()).Update(context.Background(), &r, metav1.UpdateOptions{})
			return err
		}, func() error {
			return f.Client.Kubernetes().CoreV1().Pods(obj.GetNamespace()).Delete(context.Background(), obj.GetName(), metav1.DeleteOptions{})
		})
	default:
		return errors.New(fmt.Sprintf("could not create resource %s", Kind))
	}
}

// waitOnMode waits depending on the mode
func (f *FakeEventsManager) waitOnMode() {
	switch f.Mode {
	case Delay:
		time.Sleep(f.Delay)
		break
	case Timestamps:
		break
	}
}
