package operator

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

const (
	fetchSensorPodTimeout                        = 10 * time.Second
	fetchCurrentSensorHelmReleaseRevisionTimeout = 10 * time.Second
	helmReleaseNameAnnotationsKey                = "meta.helm.sh/release-name"

	sensorPodAppLabelKey   = "app.kubernetes.io/component"
	sensorPodAppLabelValue = "sensor"

	// SecretTypeHelmReleaseV1 is where Helm stores the metadata for each
	// release starting with Helm 3.
	// See https://helm.sh/docs/faq/changes_since_helm2/#secrets-as-the-default-storage-driver
	secretTypeHelmReleaseV1 corev1.SecretType = "helm.sh/release.v1"
)

// ExtractHelmRevisionFromHelmSecret Extracts the Helm release revision number from the secret where Helm stores
// the release metadata starting with Helm 3.
// Assuming the following naming conventions:
// - For secretTypeHelmReleaseV1: "sh.helm.release.v1.RELEASE_NAME.vREVISION".
// Returns
// - 0, nil if the secret corresponds to a release different to `helmReleaseName`.
// - 0, err if the secret is not a helm secret (see isHelmSecret), or the secret name doesn't have the expected format.
// See https://helm.sh/docs/faq/changes_since_helm2/#secrets-as-the-default-storage-driver
// FIXME: replace by usage of Helm action client
func (o *operatorImpl) extractHelmRevisionFromHelmSecret(secret *corev1.Secret) (uint64, error) {
	if secret.Type == secretTypeHelmReleaseV1 {
		secretName := secret.Name
		splitSecretName := strings.Split(secretName, ".")
		if len(splitSecretName) != 6 {
			return 0, errors.Errorf("unexpected format for Helm release revision %s", secretName)
		}
		if splitSecretName[4] != o.helmReleaseName {
			return 0, nil
		}
		rev, err := strconv.Atoi(splitSecretName[5][1:])
		if err != nil {
			return 0, errors.Wrapf(err, "unexpected format for Helm release revision %s, revision is not an int", secretName)
		}
		if rev <= 0 {
			return 0, errors.Errorf("unexpected format for Helm release revision %s, revision is not a positive int", secretName)
		}
		return uint64(rev), nil
	}
	return 0, errors.Errorf("unexpected type %s for secret with name %s", secret.Type, secret.Name)
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

	sensorPodLabel := fmt.Sprintf("%s=%s", sensorPodAppLabelKey, sensorPodAppLabelValue)
	retrySleepTime := time.Second
	// Here finding 0 pods or more than 1 pod are treated as retryable errors, while all other errors are permanent
	for {
		select {
		case <-ctx.Done():
			return pod, errors.New("timeout fetching Sensor pod")
		default:
		}

		var sensorPods []corev1.Pod
		for _, phase := range []string{"Running", "Pending"} {
			listOpts := metav1.ListOptions{
				LabelSelector: sensorPodLabel,
				FieldSelector: fmt.Sprintf("status.phase=%s", phase),
			}
			podList, err := o.k8sClient.CoreV1().Pods(o.appNamespace).List(ctx, listOpts)
			if err != nil {
				return pod, err
			}
			sensorPods = append(sensorPods, podList.Items...)
		}
		switch numPodsFound := len(sensorPods); numPodsFound {
		case 0:
			log.Infof("no sensor pod found yet for namespace %s and label %s, will retry in %s",
				o.appNamespace, sensorPodLabel, retrySleepTime)
			time.Sleep(retrySleepTime)
		case 1:
			return sensorPods[0], nil
		default:
			podNamesStr := fmt.Sprintf("%s, %s, ...", sensorPods[0].GetName(), sensorPods[1].GetName())
			log.Infof("more than 1 pod found for namespace %s and label %s, will retry in %s: %s",
				o.appNamespace, sensorPodLabel, retrySleepTime, podNamesStr)
			time.Sleep(retrySleepTime)
		}
	}
}

// Should be called after `o.helmReleaseName` is initialized with `fetchHelmReleaseName`
func (o *operatorImpl) fetchCurrentSensorHelmReleaseRevision(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, fetchCurrentSensorHelmReleaseRevisionTimeout)
	defer cancel()

	var (
		errResult           error
		helmReleaseRevision uint64
	)
	for _, secretType := range getHelmSecretTypes() {
		listOpts := secretType.ListOptions(o.helmReleaseName)
		secrets, err := o.k8sClient.CoreV1().Secrets(o.appNamespace).List(ctx, listOpts)
		if err != nil {
			errResult = errors.Wrap(err, "failed to look up Helm release revision")
			break
		} else {
			for _, secret := range secrets.Items {
				rev, extractionErr := o.extractHelmRevisionFromHelmSecret(&secret)
				if extractionErr != nil {
					err = extractionErr
					break
				}
				if rev > helmReleaseRevision {
					helmReleaseRevision = rev
				}
			}
			if err != nil {
				errResult = errors.Wrap(err, "failed to look up Helm release revision")
				break
			}
		}
	}
	if errResult != nil {
		return errResult
	}

	o.helmReleaseRevision = helmReleaseRevision
	log.Infof("Detected helm release revision %d", o.helmReleaseRevision)
	return nil
}

func (o *operatorImpl) isSensorHelmManaged() bool {
	return o.helmReleaseRevision > 0 && o.helmReleaseName != ""
}

func (o *operatorImpl) watchSecrets(sif informers.SharedInformerFactory) chan struct{} {
	secretInformerStopper := make(chan struct{})
	secretInformer := sif.Core().V1().Secrets().Informer()
	secretInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(newObj interface{}) {
			secret := newObj.(*corev1.Secret)
			err := o.processSecret(secret, secretInformerStopper)
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

func (o *operatorImpl) processSecret(secret *corev1.Secret, secretInformerStopper chan struct{}) error {
	var processingError error
	if o.isSensorHelmManaged() && isHelmSecret(secret) {
		revision, err := o.extractHelmRevisionFromHelmSecret(secret)
		if err != nil {
			log.Errorf("Failed to extract Helm revision from secret, ignoring potential new Helm release: %s", processingError)
		} else if revision > o.helmReleaseRevision {
			log.Warnf("Detected Helm revision %d higher than current revision %d, stopping sensor", revision, o.helmReleaseRevision)
			secretInformerStopper <- struct{}{}
			o.stop(nil)
		}
	}
	return processingError
}

// HelmSecretType is a secret type that Helm uses to store release information
type HelmSecretType interface {
	Type() corev1.SecretType
	// ListOptions Options that can be used to retrieve all secrets for a Helm release
	ListOptions(helmReleaseName string) metav1.ListOptions
}

type helmSecretTypeReleaseV1 struct{}

func (*helmSecretTypeReleaseV1) Type() corev1.SecretType {
	return secretTypeHelmReleaseV1
}

func (h *helmSecretTypeReleaseV1) ListOptions(helmReleaseName string) metav1.ListOptions {
	return metav1.ListOptions{
		FieldSelector: fmt.Sprintf("type=%s", h.Type()),
		LabelSelector: fmt.Sprintf("name=%s", helmReleaseName),
	}
}

// GetHelmSecretTypes returns all secret types that Helm uses to store
// release information.
func getHelmSecretTypes() map[corev1.SecretType]HelmSecretType {
	return map[corev1.SecretType]HelmSecretType{
		secretTypeHelmReleaseV1: &helmSecretTypeReleaseV1{},
	}
}

// isHelmSecret returns whether the secret is used by Helm to store release information.
func isHelmSecret(secret *corev1.Secret) bool {
	_, ok := getHelmSecretTypes()[secret.Type]
	return ok
}
