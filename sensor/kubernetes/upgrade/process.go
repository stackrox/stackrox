package upgrade

import (
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/httputil"
	pkgKubernetes "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/retry"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	preferredServiceAccountName = `sensor-upgrader`
	fallbackServiceAccountName  = `sensor`

	upgraderContainerName  = `upgrader`
	upgraderDeploymentName = `sensor-upgrader`
	processIDLabelKey      = `upgrader.sensor.stackrox.io/process-id`

	pollInterval = 10 * time.Second
)

type process struct {
	trigger *central.SensorUpgradeTrigger

	doneSig   concurrency.ErrorSignal
	k8sClient kubernetes.Interface
}

func newProcess(trigger *central.SensorUpgradeTrigger, baseConfig *rest.Config) (*process, error) {
	config := *baseConfig
	p := &process{
		trigger: trigger,
		doneSig: concurrency.NewErrorSignal(),
	}
	baseWrapTransport := baseConfig.WrapTransport
	config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		if baseWrapTransport != nil {
			rt = baseWrapTransport(rt)
		}
		return httputil.ContextBoundRoundTripper(&p.doneSig, rt)
	}
	k8sClient, err := kubernetes.NewForConfig(&config)
	if err != nil {
		return nil, errorhelpers.PanicOnDevelopment(err)
	}
	p.k8sClient = k8sClient

	return p, nil
}

func (p *process) Run() {
	p.doneSig.SignalWithError(p.doRun())
}

func (p *process) doRun() error {
	log.Infof("Launching upgrade process %s with upgrader image %s", p.trigger.GetUpgradeProcessId(), p.trigger.GetImage())
	deployment, err := p.getOrCreateUpgraderDeployment()
	if err != nil {
		return err
	}

	return p.watchUpgraderDeployment(deployment)
}

func (p *process) waitForDeploymentDeletion(name string, uid types.UID) error {
	err := p.waitForDeploymentDeletionOnce(name, uid)
	for err != nil && retry.IsRetryable(err) {
		err = p.waitForDeploymentDeletionOnce(name, uid)
	}
	return err
}

func (p *process) waitForDeploymentDeletionOnce(name string, uid types.UID) error {
	deploymentsClient := p.k8sClient.ExtensionsV1beta1().Deployments(namespaces.StackRox)
	listOpts := metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", name),
	}

	deploymentsList, err := deploymentsClient.List(listOpts)
	if err != nil {
		return err
	}

	if len(deploymentsList.Items) == 0 || deploymentsList.Items[0].UID != uid {
		return nil // deleted
	}

	watchOpts := listOpts
	watchOpts.ResourceVersion = deploymentsList.ResourceVersion

	log.Infof("Deployment %s with UID %s is still present, watching for changes ...", name, uid)
	watcher, err := deploymentsClient.Watch(watchOpts)
	if err != nil {
		return err
	}
	defer watcher.Stop()
	for {
		select {
		case ev := <-watcher.ResultChan():
			if ev.Type == watch.Error {
				return retry.MakeRetryable(errors.Errorf("error of during watch: %v", ev.Object))
			}

			obj, _ := ev.Object.(metav1.Object)
			if obj == nil {
				return errorhelpers.PanicOnDevelopment(errors.Errorf("object returned by watch is a non-k8s object of type %T", ev.Object))
			}

			if obj.GetName() != name {
				errorhelpers.PanicOnDevelopmentf("received watch event for unexpected object %s of type %T", name, obj)
				continue // should not happen
			}

			if obj.GetUID() == uid && ev.Type == watch.Deleted {
				log.Infof("Received delete event for %s", obj.GetUID())
				return nil // old object with this name was deleted
			} else if obj.GetUID() != uid && ev.Type != watch.Deleted {
				return nil // new object with this name exists
			}

		case <-p.doneSig.Done():
			return p.doneSig.Err()
		}
	}
}

func (p *process) getOrCreateUpgraderDeployment() (*v1beta1.Deployment, error) {
	deploymentsClient := p.k8sClient.ExtensionsV1beta1().Deployments(namespaces.StackRox)

	upgraderDeployment, err := deploymentsClient.Get(upgraderDeploymentName, metav1.GetOptions{})
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			return nil, errors.Wrap(err, "retrieving existing upgrader deployment")
		}
		upgraderDeployment = nil
	}

	if upgraderDeployment != nil {
		if upgraderDeployment.GetLabels()[processIDLabelKey] == p.GetID() {
			log.Infof("Current upgrader deployment for process ID %s found", p.GetID())
			return upgraderDeployment, nil
		}

		log.Infof("Found leftover upgrader deployment. Deleting ...")
		err := deploymentsClient.Delete(upgraderDeployment.GetName(), &metav1.DeleteOptions{
			Preconditions:     &metav1.Preconditions{UID: &upgraderDeployment.UID},
			PropagationPolicy: &pkgKubernetes.DeletePolicyBackground,
		})
		if err != nil && !k8sErrors.IsNotFound(err) {
			return nil, errors.Wrap(err, "deleting old upgrader deployment")
		}
		if err := p.waitForDeploymentDeletion(upgraderDeployment.GetName(), upgraderDeployment.GetUID()); err != nil {
			return nil, errors.Wrap(err, "deleting old upgrader deployment")
		}
		log.Infof("Deleted leftover upgrader deployment")
	}

	serviceAccountName := p.chooseServiceAccount()
	log.Infof("Using service account %s for upgrade process %s", serviceAccountName, p.GetID())

	newDeployment := createDeployment(p.trigger, serviceAccountName)

	createdDeployment, err := deploymentsClient.Create(newDeployment)
	if err != nil {
		return nil, errors.Wrap(err, "creating new upgrader deployment")
	}
	log.Infof("Successfully created new upgrader deployment for upgrade process %s", p.GetID())
	return createdDeployment, nil
}

func (p *process) watchUpgraderDeployment(deployment *v1beta1.Deployment) error {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	log.Infof("Watching over upgrader deployment for upgrade process %s", p.GetID())

	for {
		select {
		case <-ticker.C:
			done, err := p.pollAndUpdateProgress(deployment)
			if err != nil {
				log.Errorf("Error polling upgrader deployment/pods: %v", err)
			} else if done {
				return nil
			}

		case <-p.doneSig.Done():
			return p.doneSig.Err()
		}
	}
}

func (p *process) pollAndUpdateProgress(deployment *v1beta1.Deployment) (bool, error) {
	errs := errorhelpers.NewErrorList("polling")

	deploymentsClient := p.k8sClient.ExtensionsV1beta1().Deployments(deployment.GetNamespace())
	foundDeployment, err := deploymentsClient.Get(deployment.GetName(), metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return true, nil
		}
		errs.AddWrap(err, "upgrader deployment")
	} else if foundDeployment != nil && foundDeployment.GetUID() != deployment.GetUID() {
		return true, nil // new upgrader deployment
	}

	podsClient := p.k8sClient.CoreV1().Pods(deployment.GetNamespace())
	pods, err := podsClient.List(metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(deployment.Spec.Selector),
	})
	if err != nil {
		errs.AddWrap(err, "upgrader pods")
		return false, errs.ToError()
	}

	for _, pod := range pods.Items {
		p.checkPodStatus(&pod)
	}
	return false, nil
}

func (p *process) checkPodStatus(pod *v1.Pod) {
	var upgraderContainerStatus *v1.ContainerStatus
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name == upgraderContainerName {
			upgraderContainerStatus = &cs
			break
		}
	}

	if upgraderContainerStatus == nil {
		log.Warnf("no upgrade container found for pod %s", pod.Name)
	}

	if upgraderContainerStatus.State.Running != nil {
		log.Infof("Pod %s is running!", pod.Name)
	} else if terminatedState := upgraderContainerStatus.State.Terminated; terminatedState != nil {
		log.Infof("Pod %s terminated, reason: %s (%s)", pod.Name, terminatedState.Reason, terminatedState.Message)
	} else if waitingState := upgraderContainerStatus.State.Waiting; waitingState != nil {
		if isImagePullRelatedReason(waitingState.Reason) {
			log.Warnf("Pod %s seems to have trouble pulling the image, reason: %s (%s)", pod.Name, waitingState.Reason, waitingState.Message)
		} else {
			log.Warnf("Pod %s is waiting to start, reason: %s (%s)", pod.Name, waitingState.Reason, waitingState.Message)
		}
	}
}

func (p *process) Terminate(err error) {
	p.doneSig.SignalWithError(err)
}

func (p *process) GetID() string {
	return p.trigger.GetUpgradeProcessId()
}

func (p *process) chooseServiceAccount() string {
	saClient := p.k8sClient.CoreV1().ServiceAccounts(namespaces.StackRox)

	sensorUpgraderSA, err := saClient.Get(preferredServiceAccountName, metav1.GetOptions{})
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			log.Warnf("Could not check for existence of %q service account: %v. Performing upgrade with default %q service account", preferredServiceAccountName, err, fallbackServiceAccountName)
		}
		sensorUpgraderSA = nil
	}

	if sensorUpgraderSA != nil {
		return sensorUpgraderSA.GetName()
	}

	return fallbackServiceAccountName
}
