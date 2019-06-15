package orchestrator

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
	pkgKubernetes "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/orchestrators"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/kubernetes"
)

const (
	ownershipLabel = `owner.stackrox.io/sensor`

	namespace = "stackrox"
)

var (
	log = logging.LoggerForModule()
)

type kubernetesOrchestrator struct {
	client    *kubernetes.Clientset
	namespace string

	sensorInstanceID string
}

// MustCreate returns a new Kubernetes orchestrator client, or panics.
func MustCreate(sensorInstanceID string) orchestrators.Orchestrator {
	o, err := New(sensorInstanceID)
	if err != nil {
		panic(err)
	}
	return o
}

// New returns a new kubernetes orchestrator client.
func New(sensorInstanceID string) (orchestrators.Orchestrator, error) {
	return &kubernetesOrchestrator{
		client:           client.MustCreateClientSet(),
		namespace:        namespace,
		sensorInstanceID: sensorInstanceID,
	}, nil
}

func (k *kubernetesOrchestrator) patchLabels(labels *map[string]string) {
	if *labels == nil {
		*labels = make(map[string]string)
	}
	(*labels)[ownershipLabel] = k.sensorInstanceID
}

func (k *kubernetesOrchestrator) logEvents(ds *v1beta1.DaemonSet) error {
	if ds == nil {
		return nil
	}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: ds.GetLabels(),
	})
	if err != nil {
		return errors.Wrapf(err, "error creating label selector")
	}
	eventList, err := k.client.CoreV1().Events(namespace).List(metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return errors.Wrapf(err, "could not get events for daemonset %q", ds.GetName())
	}
	log.Errorf("Events for daemonset %q", ds.GetName())
	for _, e := range eventList.Items {
		log.Errorf("\t%s", e.Message)
	}
	return nil
}

func (k *kubernetesOrchestrator) waitForDesired(name string) (int, error) {
	var ds *v1beta1.DaemonSet
	err := retry.WithRetry(
		func() error {
			var err error
			ds, err = k.client.ExtensionsV1beta1().DaemonSets(namespace).Get(name, metav1.GetOptions{})
			if err != nil {
				return errors.Wrapf(err, "could not get daemonset %q from Kubernetes", name)
			}

			if ds.Status.DesiredNumberScheduled == 0 {
				return errors.Errorf("compliance daemonset %q has 0 desired pods", name)
			}
			return nil
		},
		retry.Tries(30),
		retry.BetweenAttempts(func() {
			time.Sleep(2 * time.Second)
		}),
		retry.OnFailedAttempts(func(err error) {
			log.Warn(err)
		}),
	)
	if err != nil {
		if logErr := k.logEvents(ds); logErr != nil {
			log.Error(logErr)
		}
		return 0, err
	}

	return int(ds.Status.DesiredNumberScheduled), err
}

func (k *kubernetesOrchestrator) LaunchDaemonSet(service orchestrators.SystemService) (string, int, error) {
	ds := asDaemonSet(k.newServiceWrap(service))
	k.patchLabels(&ds.Labels)

	actual, err := k.client.ExtensionsV1beta1().DaemonSets(k.namespace).Create(ds)
	if err != nil {
		return "", 0, errors.Wrapf(err, "error creating compliance daemonset")
	}

	desired, err := k.waitForDesired(actual.GetName())
	return actual.GetName(), desired, err
}

func (k *kubernetesOrchestrator) newServiceWrap(service orchestrators.SystemService) *serviceWrap {
	return &serviceWrap{
		SystemService: service,
		namespace:     k.namespace,
	}
}

func (k *kubernetesOrchestrator) Kill(name string) error {
	if ds, err := k.client.ExtensionsV1beta1().DaemonSets(k.namespace).Get(name, metav1.GetOptions{}); err == nil && ds != nil {
		if err := k.client.ExtensionsV1beta1().DaemonSets(k.namespace).Delete(name, pkgKubernetes.DeleteOption); err != nil {
			log.Errorf("unable to delete daemonset %s: %s", name, err)
			return err
		}
		return nil
	}

	if deploy, err := k.client.ExtensionsV1beta1().Deployments(k.namespace).Get(name, metav1.GetOptions{}); err == nil && deploy != nil {
		if err := k.client.ExtensionsV1beta1().Deployments(k.namespace).Delete(name, pkgKubernetes.DeleteOption); err != nil {
			log.Errorf("unable to delete deployment %s: %s", name, err)
			return err
		}
		return nil
	}

	err := fmt.Errorf("unable to delete service %s; service not found", name)
	log.Error(err)
	return err
}

// WaitForCompletion currently cannot be implemented in Kubernetes because DaemonSet Restart Policy must be always
func (k *kubernetesOrchestrator) WaitForCompletion(_ string, timeout time.Duration) error {
	time.Sleep(timeout)
	return nil
}

func (k *kubernetesOrchestrator) labelSelector(ownedByThisInstance bool) (labels.Selector, error) {
	hasLabelReq, err := labels.NewRequirement(ownershipLabel, selection.Exists, nil)
	if err != nil {
		return nil, err
	}
	var op selection.Operator
	if ownedByThisInstance {
		op = selection.Equals
	} else {
		op = selection.NotEquals
	}
	labelMatchesReq, err := labels.NewRequirement(ownershipLabel, op, []string{k.sensorInstanceID})
	if err != nil {
		return nil, err
	}
	return labels.NewSelector().Add(*hasLabelReq, *labelMatchesReq), nil
}

func (k *kubernetesOrchestrator) CleanUp(ownedByThisInstance bool) error {
	ls, err := k.labelSelector(ownedByThisInstance)
	if err != nil {
		return errors.Wrap(err, "creating label selector")
	}
	listOpts := metav1.ListOptions{
		LabelSelector: ls.String(),
	}
	propagationPolicy := metav1.DeletePropagationBackground
	deleteOpts := &metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	}

	var errList errorhelpers.ErrorList
	err = k.client.ExtensionsV1beta1().DaemonSets(k.namespace).DeleteCollection(deleteOpts, listOpts)
	if err != nil {
		errList.AddStringf("deleting daemonsets: %v", err)
	}
	err = k.client.ExtensionsV1beta1().Deployments(k.namespace).DeleteCollection(deleteOpts, listOpts)
	if err != nil {
		errList.AddStringf("deleting deployments: %v", err)
	}

	return errList.ToError()
}
