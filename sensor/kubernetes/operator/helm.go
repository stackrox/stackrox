package operator

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

const (
	fetchSensorPodTimeout                          = 10 * time.Second
	fetchSensorPodSleepTime                        = time.Second
	fetchCurrentSensorHelmReleaseRevisionTimeout   = 1 * time.Minute
	fetchCurrentSensorHelmReleaseRevisionSleepTime = 5 * time.Second
	helmReleaseNameAnnotationsKey                  = "meta.helm.sh/release-name"

	sensorPodAppLabelKey   = "app.kubernetes.io/component"
	sensorPodAppLabelValue = "sensor"

	helmDriverEnvVar = "HELM_DRIVER"
	// SecretTypeHelmReleaseV1 is where Helm stores the metadata for each
	// release starting with Helm 3.
	// See https://helm.sh/docs/faq/changes_since_helm2/#secrets-as-the-default-storage-driver
	secretTypeHelmReleaseV1 corev1.SecretType = "helm.sh/release.v1"
)

func (o *operatorImpl) initializeHelmActionConfig() error {
	settings := cli.New()
	helmActionConfig := new(action.Configuration)
	helmDriver := os.Getenv(helmDriverEnvVar)
	if err := helmActionConfig.Init(settings.RESTClientGetter(), o.appNamespace, helmDriver, log.Debugf); err != nil {
		return err
	}
	o.helmGetClient = action.NewGet(helmActionConfig)

	return nil
}

// Should be called after `Operator.Initialize`
func (o *operatorImpl) fetchCurrentSensorHelmReleaseRevision(ctx context.Context) (int, error) {
	var releaseRevision int
	ctx, cancel := context.WithTimeout(ctx, fetchCurrentSensorHelmReleaseRevisionTimeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return releaseRevision, errors.Wrap(ctx.Err(), "timeout fetching Helm release revision")
		default:
		}

		sensorRelease, err := o.helmGetClient.Run(o.helmReleaseName)
		if err != nil {
			return releaseRevision, errors.Wrapf(err, "error fetching helm information for release %s", o.helmReleaseName)
		}
		sensorReleaseStatus := sensorRelease.Info.Status
		if sensorReleaseStatus != release.StatusDeployed {
			log.Infof("Latest Helm release for Sensor with name %s is in status %s, will wait %s for it to reach %s status",
				o.helmReleaseName, sensorReleaseStatus, fetchCurrentSensorHelmReleaseRevisionTimeout, release.StatusDeployed)
			time.Sleep(fetchCurrentSensorHelmReleaseRevisionSleepTime)
		} else {
			return sensorRelease.Version, nil
		}
	}
}

func (o *operatorImpl) fetchHelmReleaseName(ctx context.Context) error {
	sensorPod, err := o.fetchSensorPod(ctx)
	if err != nil {
		return err
	}
	if helmReleaseName, ok := sensorPod.GetObjectMeta().GetAnnotations()[helmReleaseNameAnnotationsKey]; ok {
		o.helmReleaseName = helmReleaseName
	} else {
		return errors.Errorf("Helm release name not found in annotations key %s", helmReleaseNameAnnotationsKey)
	}

	log.Infof("Detected helm release name %s", o.helmReleaseName)
	return nil
}

func (o *operatorImpl) fetchSensorPod(ctx context.Context) (corev1.Pod, error) {
	var pod corev1.Pod
	ctx, cancel := context.WithTimeout(ctx, fetchSensorPodTimeout)
	defer cancel()

	podsClient := o.k8sClient.CoreV1().Pods(o.appNamespace)
	sensorPodLabel := fmt.Sprintf("%s=%s", sensorPodAppLabelKey, sensorPodAppLabelValue)
	// Here finding 0 pods or more than 1 pod are treated as retryable errors, while all other errors are permanent
	for {
		select {
		case <-ctx.Done():
			return pod, errors.Wrap(ctx.Err(), "timeout fetching Sensor pod")
		default:
		}

		var sensorPods []corev1.Pod
		for _, phase := range []string{"Running", "Pending"} {
			listOpts := metav1.ListOptions{
				LabelSelector: sensorPodLabel,
				FieldSelector: fmt.Sprintf("status.phase=%s", phase),
			}
			podList, err := podsClient.List(ctx, listOpts)
			if err != nil {
				return pod, err
			}
			sensorPods = append(sensorPods, podList.Items...)
		}
		switch numPodsFound := len(sensorPods); numPodsFound {
		case 0:
			log.Infof("no sensor pod found yet for namespace %s and label %s, will retry in %s",
				o.appNamespace, sensorPodLabel, fetchSensorPodSleepTime)
			time.Sleep(fetchSensorPodSleepTime)
		case 1:
			return sensorPods[0], nil
		default:
			podNamesStr := fmt.Sprintf("%s, %s, ...", sensorPods[0].GetName(), sensorPods[1].GetName())
			log.Infof("more than 1 pod found for namespace %s and label %s, will retry in %s: %s",
				o.appNamespace, sensorPodLabel, fetchSensorPodSleepTime, podNamesStr)
			time.Sleep(fetchSensorPodSleepTime)
		}
	}
}

func (o *operatorImpl) isSensorHelmManaged() bool {
	return o.helmReleaseRevision > 0 && o.helmReleaseName != ""
}

func (o *operatorImpl) watchSecrets(ctx context.Context, sif informers.SharedInformerFactory) chan struct{} {
	secretInformerStopper := make(chan struct{})
	secretInformer := sif.Core().V1().Secrets().Informer()
	secretInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(newObj interface{}) {
			secret := newObj.(*corev1.Secret)
			err := o.processSecret(ctx, secret, secretInformerStopper)
			if err != nil {
				err := errors.Wrapf(err, "Error processing secret with name %s", secret.GetName())
				log.Error(err)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {},
		DeleteFunc: func(obj interface{}) {},
	})
	go secretInformer.Run(secretInformerStopper)
	return secretInformerStopper
}

func (o *operatorImpl) processSecret(ctx context.Context, secret *corev1.Secret, secretInformerStopper chan struct{}) error {
	if o.isSensorHelmManaged() && isHelmSecret(secret) {
		revision, err := o.fetchCurrentSensorHelmReleaseRevision(ctx)
		if err != nil {
			return err
		} else if revision > o.helmReleaseRevision {
			log.Warnf("Detected Helm revision %d higher than current revision %d, stopping sensor", revision, o.helmReleaseRevision)
			secretInformerStopper <- struct{}{}
			o.stop(nil)
		}
	}
	return nil
}

// GetHelmSecretTypes returns all secret types that Helm uses to store
// release information.
func getHelmSecretTypes() map[corev1.SecretType]bool {
	return map[corev1.SecretType]bool{
		secretTypeHelmReleaseV1: true,
	}
}

// isHelmSecret returns whether the secret is used by Helm to store release information.
func isHelmSecret(secret *corev1.Secret) bool {
	_, ok := getHelmSecretTypes()[secret.Type]
	return ok
}
