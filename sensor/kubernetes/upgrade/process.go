package upgrade

import (
	"context"
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
	"github.com/stackrox/rox/pkg/timeutil"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common/clusterid"
	"google.golang.org/grpc/status"
	v1 "k8s.io/api/core/v1"
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

	sensorDeploymentName = `sensor`
	sensorContainerName  = `sensor`

	upgraderContainerName  = `upgrader`
	upgraderDeploymentName = `sensor-upgrader`
	processIDLabelKey      = `upgrader.sensor.stackrox.io/process-id`

	pollInterval = 10 * time.Second

	checkInTimeout       = 10 * time.Second
	checkInRetryInterval = 10 * time.Second
)

type process struct {
	trigger *central.SensorUpgradeTrigger

	doneSig   concurrency.ErrorSignal
	k8sClient kubernetes.Interface

	checkInReqC chan *central.UpgradeCheckInFromSensorRequest

	checkInClient central.SensorUpgradeControlServiceClient
}

func newProcess(trigger *central.SensorUpgradeTrigger, checkInClient central.SensorUpgradeControlServiceClient, baseConfig *rest.Config) (*process, error) {
	config := *baseConfig
	p := &process{
		trigger:       trigger,
		doneSig:       concurrency.NewErrorSignal(),
		checkInClient: checkInClient,
		checkInReqC:   make(chan *central.UpgradeCheckInFromSensorRequest, 1),
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
		return nil, utils.ShouldErr(err)
	}
	p.k8sClient = k8sClient

	return p, nil
}

func (p *process) Run() {
	go p.handleCentralCheckIns()
	p.doRun()
	p.doneSig.Signal()
}

func (p *process) ctx() context.Context {
	return concurrency.AsContext(&p.doneSig)
}

func (p *process) handleCentralCheckIns() {
	var reqRetry *central.UpgradeCheckInFromSensorRequest
	var retryTimer *time.Timer

	for {
		var req *central.UpgradeCheckInFromSensorRequest

		select {
		case <-p.doneSig.Done():
			timeutil.StopTimer(retryTimer)
			return

		case req = <-p.checkInReqC:

		case <-timeutil.TimerC(retryTimer):
			retryTimer = nil
			req = reqRetry
		}

		timeutil.StopTimer(retryTimer)
		retryTimer = nil
		reqRetry = nil

		if err := p.sendCheckInRequestSingle(req); err != nil {
			// This will not panic as .Details() gracefully handles a `nil` receiver.
			for _, detail := range status.Convert(err).Details() {
				if _, isNoUpgradeInProgress := detail.(*central.UpgradeCheckInResponseDetails_NoUpgradeInProgress); isNoUpgradeInProgress {
					log.Info("Central says there is no upgrade in progress. Exiting loop...")
					p.doneSig.Signal()
					return
				}
			}
			log.Errorf("Error: Could not check in with central on upgrade progress: %v", err)
			retryTimer = time.NewTimer(checkInRetryInterval)
			reqRetry = req
		}
	}
}

func (p *process) sendCheckInRequestSingle(req *central.UpgradeCheckInFromSensorRequest) error {
	ctx, cancel := context.WithTimeout(concurrency.AsContext(&p.doneSig), checkInTimeout)
	defer cancel()

	_, err := p.checkInClient.UpgradeCheckInFromSensor(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

// checkInWithCentral schedules a check in request for being sent to central. This is done on a best-effort basis; if
// it fails, NBD. We will keep retrying though while the upgrade process is in progress.
func (p *process) checkInWithCentral(req *central.UpgradeCheckInFromSensorRequest) {
	req.ClusterId = clusterid.Get()
	req.UpgradeProcessId = p.GetID()

	// If there is a currently pending request, remove it from the channel - it is now obsolete.
	select {
	case <-p.checkInReqC:
	default:
	}

	select {
	case <-p.doneSig.Done():
	case p.checkInReqC <- req:
	}
}

func (p *process) doRun() {
	imageToLog := p.trigger.GetImage()
	if imageToLog == "" {
		imageToLog = "same as sensor image"
	}
	log.Infof("Launching upgrade process %s with upgrader image %s", p.trigger.GetUpgradeProcessId(), imageToLog)
	err := p.createUpgraderDeploymentIfNecessary()

	var launchErrMsg string
	if err != nil {
		launchErrMsg = err.Error()
	}

	p.checkInWithCentral(&central.UpgradeCheckInFromSensorRequest{
		State: &central.UpgradeCheckInFromSensorRequest_LaunchError{
			LaunchError: launchErrMsg,
		},
	})

	p.watchUpgraderDeployment()
}

func (p *process) waitForDeploymentDeletion(name string, uid types.UID) error {
	err := p.waitForDeploymentDeletionOnce(name, uid)
	for err != nil && retry.IsRetryable(err) {
		err = p.waitForDeploymentDeletionOnce(name, uid)
	}
	return err
}

func (p *process) waitForDeploymentDeletionOnce(name string, uid types.UID) error {
	deploymentsClient := p.k8sClient.AppsV1().Deployments(namespaces.StackRox)
	listOpts := metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", name),
	}

	deploymentsList, err := deploymentsClient.List(p.ctx(), listOpts)
	if err != nil {
		return err
	}

	if len(deploymentsList.Items) == 0 || deploymentsList.Items[0].UID != uid {
		return nil // deleted
	}

	watchOpts := listOpts
	watchOpts.ResourceVersion = deploymentsList.ResourceVersion

	log.Infof("Deployment %s with UID %s is still present, watching for changes ...", name, uid)
	watcher, err := deploymentsClient.Watch(p.ctx(), watchOpts)
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
				return utils.ShouldErr(errors.Errorf("object returned by watch is a non-k8s object of type %T", ev.Object))
			}

			if obj.GetName() != name {
				utils.Should(errors.Errorf("received watch event for unexpected object %s of type %T", name, obj))
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

func (p *process) createUpgraderDeploymentIfNecessary() error {
	deploymentsClient := p.k8sClient.AppsV1().Deployments(namespaces.StackRox)

	upgraderDeployment, err := deploymentsClient.Get(p.ctx(), upgraderDeploymentName, metav1.GetOptions{})
	if err != nil {
		if !k8sErrors.IsNotFound(err) {
			return errors.Wrap(err, "retrieving existing upgrader deployment")
		}
		upgraderDeployment = nil
	}

	if upgraderDeployment != nil {
		if upgraderDeployment.GetLabels()[processIDLabelKey] == p.GetID() {
			log.Infof("Current upgrader deployment for process ID %s found", p.GetID())
			return nil
		}

		log.Info("Found leftover upgrader deployment. Deleting ...")
		err := deploymentsClient.Delete(p.ctx(), upgraderDeployment.GetName(), metav1.DeleteOptions{
			Preconditions:     &metav1.Preconditions{UID: &upgraderDeployment.UID},
			PropagationPolicy: &pkgKubernetes.DeletePolicyBackground,
		})
		if err != nil && !k8sErrors.IsNotFound(err) {
			return errors.Wrap(err, "deleting old upgrader deployment")
		}
		if err := p.waitForDeploymentDeletion(upgraderDeployment.GetName(), upgraderDeployment.GetUID()); err != nil {
			return errors.Wrap(err, "deleting old upgrader deployment")
		}
		log.Info("Deleted leftover upgrader deployment")
	}

	serviceAccountName := p.chooseServiceAccount()
	log.Infof("Using service account %s for upgrade process %s", serviceAccountName, p.GetID())

	// Fetch Sensor deployment to carry through some features of the pod spec
	sensorDeployment, err := deploymentsClient.Get(p.ctx(), sensorDeploymentName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "retrieving existing sensor deployment")
	}

	newDeployment, err := p.createDeployment(serviceAccountName, sensorDeployment)
	if err != nil {
		return errors.Wrap(err, "instantiating upgrader deployment object")
	}

	_, err = deploymentsClient.Create(p.ctx(), newDeployment, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "creating new upgrader deployment")
	}
	log.Infof("Successfully created new upgrader deployment for upgrade process %s", p.GetID())
	return nil
}

func (p *process) watchUpgraderDeployment() {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	log.Infof("Watching over upgrader deployment for upgrade process %s", p.GetID())

	for {
		select {
		case <-ticker.C:
			podStates, deploymentGone, err := p.pollAndUpdateProgress()
			var checkIn *central.UpgradeCheckInFromSensorRequest

			if err != nil {
				// Don't check in with central, that's fine - nothing to see here anyway.
				// There is no retry limit here. We can't tell Central anything useful, so the best course of action is
				// to let the timeout logic handle this.
				log.Errorf("Error polling upgrader deployment/pods: %v", err)
			} else if deploymentGone {
				checkIn = &central.UpgradeCheckInFromSensorRequest{
					State: &central.UpgradeCheckInFromSensorRequest_DeploymentGone{
						DeploymentGone: true,
					},
				}
			} else {
				// Regular check in with central on upgrader pods.
				checkIn = &central.UpgradeCheckInFromSensorRequest{
					State: &central.UpgradeCheckInFromSensorRequest_PodStates{
						PodStates: &central.UpgradeCheckInFromSensorRequest_UpgraderPodStates{
							States: podStates,
						},
					},
				}
			}

			if checkIn != nil {
				p.checkInWithCentral(checkIn)
			}

		case <-p.doneSig.Done():
			return
		}
	}
}

func (p *process) pollAndUpdateProgress() ([]*central.UpgradeCheckInFromSensorRequest_UpgraderPodState, bool, error) {
	errs := errorhelpers.NewErrorList("polling")

	deploymentsClient := p.k8sClient.AppsV1().Deployments(namespaces.StackRox)
	foundDeployment, err := deploymentsClient.Get(p.ctx(), upgraderDeploymentName, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, true, nil
		}
		errs.AddWrap(err, "upgrader deployment")
	} else if foundDeployment != nil && foundDeployment.Labels[processIDLabelKey] != p.GetID() {
		return nil, true, nil // new upgrader deployment
	}

	podsClient := p.k8sClient.CoreV1().Pods(foundDeployment.GetNamespace())
	pods, err := podsClient.List(p.ctx(), metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(foundDeployment.Spec.Selector),
	})
	if err != nil {
		errs.AddWrap(err, "upgrader pods")
		return nil, false, errs.ToError()
	}

	podStates := make([]*central.UpgradeCheckInFromSensorRequest_UpgraderPodState, 0, len(pods.Items))
	for i := range pods.Items {
		podStates = append(podStates, p.checkPodStatus(&pods.Items[i]))
	}
	return podStates, false, nil
}

func (p *process) checkPodStatus(pod *v1.Pod) *central.UpgradeCheckInFromSensorRequest_UpgraderPodState {
	var upgraderContainerStatus *v1.ContainerStatus
	for i, cs := range pod.Status.ContainerStatuses {
		if cs.Name == upgraderContainerName {
			upgraderContainerStatus = &pod.Status.ContainerStatuses[i]
			break
		}
	}

	s := &central.UpgradeCheckInFromSensorRequest_UpgraderPodState{
		PodName: pod.GetName(),
	}

	if upgraderContainerStatus == nil {
		log.Warnf("no upgrade container found for pod %s", pod.Name)
		s.Error = &central.UpgradeCheckInFromSensorRequest_PodErrorCondition{
			Message: "no upgrade container found",
		}
	} else if upgraderContainerStatus.State.Running != nil {
		log.Infof("Upgrader pod %s is running", pod.GetName())
		s.Started = true
	} else if terminatedState := upgraderContainerStatus.State.Terminated; terminatedState != nil {
		s.Started = true
		if terminatedState.ExitCode != 0 {
			s.Error = &central.UpgradeCheckInFromSensorRequest_PodErrorCondition{
				Message: fmt.Sprintf("Pod terminated: %s (%s)", terminatedState.Message, terminatedState.Reason),
			}
		}
		log.Infof("Upgrader pod %s terminated, reason: %s (%s)", pod.Name, terminatedState.Reason, terminatedState.Message)
	} else if waitingState := upgraderContainerStatus.State.Waiting; waitingState != nil {
		if isImagePullRelatedReason(waitingState.Reason) {
			s.Error = &central.UpgradeCheckInFromSensorRequest_PodErrorCondition{
				Message:      fmt.Sprintf("Error pulling image: %s (%s)", waitingState.Reason, waitingState.Message),
				ImageRelated: true,
			}
			log.Warnf("Upgrader pod %s seems to have trouble pulling the image, reason: %s (%s)", pod.Name, waitingState.Reason, waitingState.Message)
		}
	}

	return s
}

func (p *process) Terminate(err error) {
	p.doneSig.SignalWithError(err)
}

func (p *process) GetID() string {
	return p.trigger.GetUpgradeProcessId()
}

func (p *process) chooseServiceAccount() string {
	saClient := p.k8sClient.CoreV1().ServiceAccounts(namespaces.StackRox)

	sensorUpgraderSA, err := saClient.Get(p.ctx(), preferredServiceAccountName, metav1.GetOptions{})
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
