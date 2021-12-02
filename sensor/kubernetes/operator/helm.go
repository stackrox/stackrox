package operator

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "k8s.io/api/core/v1"
)

const (
	fetchCurrentSensorHelmReleaseRevisionTimeout = 10 * time.Second
	helmReleaseNameMetadataKey = "name"

	// SecretTypeHelmReleaseV1 is where Helm stores the metadata for each
	// release starting with Helm 3.
	// See https://helm.sh/docs/faq/changes_since_helm2/#secrets-as-the-default-storage-driver
	secretTypeHelmReleaseV1 v1.SecretType = "helm.sh/release.v1"
)

func getHelmReleaseName(pod v1.Pod) string {
	return pod.GetObjectMeta().GetLabels()[helmReleaseNameMetadataKey]
}

// ExtractHelmRevisionFromHelmSecret Extracts the Helm release revision number from the secret where Helm stores
// the release metadata starting with Helm 3.
// Assuming the following naming conventions:
// - For secretTypeHelmReleaseV1: "sh.helm.release.v1.RELEASE_NAME.vREVISION".
// Returns
// - 0, nil if the secret corresponds to a release different to `helmReleaseName`.
// - 0, err if the secret is not a helm secret (see isHelmSecret), or the secret name doesn't have the expected format.
// See https://helm.sh/docs/faq/changes_since_helm2/#secrets-as-the-default-storage-driver
// FIXME: replace by usage of Helm action client
func (o *operatorImpl) extractHelmRevisionFromHelmSecret(secret *v1.Secret) (uint64, error) {
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

func (o *operatorImpl) fetchCurrentSensorHelmReleaseRevision(ctx context.Context) (uint64, error) {
	ctx, cancel := context.WithTimeout(ctx, fetchCurrentSensorHelmReleaseRevisionTimeout)
	defer cancel()

	var (
		errResult error
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
		return 0, errResult
	}
	return helmReleaseRevision, nil
}

func (o *operatorImpl) isSensorHelmManaged() bool {
	return o.helmReleaseRevision > 0 && o.helmReleaseName != ""
}

func (o *operatorImpl) processSecret(secret *v1.Secret) error {
	var processingError error
	if o.isSensorHelmManaged() && isHelmSecret(secret) {
		revision, err := o.extractHelmRevisionFromHelmSecret(secret)
		if err != nil {
			processingError = errors.Wrap(err, "failed to extract Helm revision from secret, ignoring potential new Helm release")
			log.Error(processingError)
		} else if revision > o.helmReleaseRevision {
			log.Warnf("Detected Helm revision %d higher than current revision %d, stopping sensor", revision, o.helmReleaseRevision)
			// FIXME send signal to stop the sensor
			// TODO error management
			// was: d.sensor.Stop()
		}
	}
	return processingError
}

// HelmSecretType is a secret type that Helm uses to store release information
type HelmSecretType interface {
	Type() v1.SecretType
	// ListOptions Options that can be used to retrieve all secrets for a Helm release
	ListOptions(helmReleaseName string) metav1.ListOptions
}

type helmSecretTypeReleaseV1 struct{}

func (*helmSecretTypeReleaseV1) Type() v1.SecretType {
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
func getHelmSecretTypes() map[v1.SecretType]HelmSecretType {
	return map[v1.SecretType]HelmSecretType{
		secretTypeHelmReleaseV1: &helmSecretTypeReleaseV1{},
	}
}

// isHelmSecret returns whether the secret is used by Helm to store release information.
func isHelmSecret(secret *v1.Secret) bool {
	_, ok := getHelmSecretTypes()[secret.Type]
	return ok
}