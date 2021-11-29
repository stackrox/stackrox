package sensor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"strconv"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/namespaces"
	"gopkg.in/square/go-jose.v2/jwt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/api/core/v1"
)

const (
	namespaceFile = `/run/secrets/kubernetes.io/serviceaccount/namespace`
	tokenFile     = `/run/secrets/kubernetes.io/serviceaccount/token`

	namespaceClaimKey        = `kubernetes.io/serviceaccount/namespace`
	serviceAccountIDClaimKey = `kubernetes.io/serviceaccount/service-account.uid`

	fetchClusterIdentificationTimeout = 10 * time.Second

	// SecretTypeHelmReleaseV1 is where Helm stores the metadata for each
	// release starting with Helm 3.
	// See https://helm.sh/docs/faq/changes_since_helm2/#secrets-as-the-default-storage-driver
	secretTypeHelmReleaseV1 corev1.SecretType = "helm.sh/release.v1"
)

// populateFromServiceAccountTokenFile populates the information stored in out from the JWT token stored in the file
// /run/secrets/kubernetes.io/serviceaccount/token, which contains the namespace as well as the UID of the service
// account objects as a JWT.
func populateFromServiceAccountTokenFile(out *storage.SensorDeploymentIdentification) error {
	tokenBytes, err := os.ReadFile(tokenFile)
	if err != nil {
		return errors.Wrapf(err, "reading token from file %s", tokenFile)
	}

	saToken, err := jwt.ParseSigned(string(bytes.TrimSpace(tokenBytes)))
	if err != nil {
		return errors.Wrapf(err, "parsing service account JWT from file %s", tokenFile)
	}

	var claims map[string]interface{}
	if err := saToken.UnsafeClaimsWithoutVerification(&claims); err != nil {
		return errors.Wrapf(err, "obtaining claims from service account JWT from file %s", tokenFile)
	}
	out.AppNamespace, _ = claims[namespaceClaimKey].(string)
	out.AppServiceaccountId, _ = claims[serviceAccountIDClaimKey].(string)
	return nil
}

// populateFromServiceAccountNamespaceFile populates the app namespace information in out, reading the current namespace
// from the file /run/secrets/kubernetes.io/serviceaccount/namespace.
func populateFromServiceAccountNamespaceFile(out *storage.SensorDeploymentIdentification) error {
	if out.GetAppNamespace() != "" {
		return nil
	}

	appNamespaceBytes, err := os.ReadFile(namespaceFile)
	if err != nil {
		return errors.Wrapf(err, "reading application namespace from file %s", namespaceFile)
	}
	out.AppNamespace = string(bytes.TrimSpace(appNamespaceBytes))
	return nil
}

// populateFromKubernetes populates the system, default, and app namespace IDs in out from information returned by the
// Kubernetes API server.
func populateFromKubernetes(ctx context.Context, k8sClient kubernetes.Interface,
	helmManagedConfig *central.HelmManagedConfigInit, out *storage.SensorDeploymentIdentification) error {
	nsClient := k8sClient.CoreV1().Namespaces()

	out.K8SNodeName = k8sNodeName.Setting()

	var errResult error
	systemNS, err := k8sClient.CoreV1().Namespaces().Get(ctx, namespaces.KubeSystem, metav1.GetOptions{})
	if err != nil {
		errResult = multierror.Append(errResult, errors.Wrapf(err, "failed to look up system namespace %q", namespaces.KubeSystem))
	} else {
		out.SystemNamespaceId = string(systemNS.GetUID())
	}

	defaultNS, err := k8sClient.CoreV1().Namespaces().Get(ctx, namespaces.Default, metav1.GetOptions{})
	if err != nil {
		errResult = multierror.Append(errResult, errors.Wrap(err, "failed to look up default namespace"))
	} else {
		out.DefaultNamespaceId = string(defaultNS.GetUID())
	}

	appNS := out.GetAppNamespace()
	if appNS == "" {
		return errResult
	}

	appNSObj, err := nsClient.Get(ctx, appNS, metav1.GetOptions{})
	if err != nil {
		errResult = multierror.Append(errResult, errors.Wrapf(err, "failed to look up application namespace %q", appNS))
	} else {
		out.AppNamespaceId = string(appNSObj.GetUID())
	}

	if helmManagedConfig != nil {
		listOpts := metav1.ListOptions{FieldSelector: fmt.Sprintf("type=%s", secretTypeHelmReleaseV1)}
		secrets, err := k8sClient.CoreV1().Secrets(appNS).List(ctx, listOpts)
		if err != nil {
			errResult = multierror.Append(errResult, errors.Wrap(err, "failed to look up Helm release revision"))
		} else {
			var helmReleaseRevision uint64 = 0
			for _, secret := range secrets.Items {
				rev, err := extractHelmRevisionFromHelmSecret(helmManagedConfig.HelmReleaseName, secret)
				if err != nil {
					break
				}
				if rev > helmReleaseRevision {
					helmReleaseRevision = rev
				}
			}
			if err != nil {
				errResult = multierror.Append(errResult, errors.Wrap(err, "failed to look up Helm release revision"))
			} else {
				out.HelmReleaseRevision = helmReleaseRevision
			}
		}
	}

	return errResult
}

// Extracts the Helm release revision number from the secret where Helm stores
// the release metadata starting with Helm 3.
// Assuming the following naming conventions:
// - For secretTypeHelmReleaseV1: "sh.helm.release.v1.RELEASE_NAME.vREVISION"
// See https://helm.sh/docs/faq/changes_since_helm2/#secrets-as-the-default-storage-driver
func extractHelmRevisionFromHelmSecret(helmReleaseName string, secret corev1.Secret) (uint64, error) {
	if secret.Type == secretTypeHelmReleaseV1 {
		secretName := secret.Name
		splitSecretName := strings.Split(secretName, ".")
		if len(splitSecretName) != 6 || splitSecretName[4] != helmReleaseName {
			return 0, errors.Errorf("unexpected format for Helm release revision %s", secretName)
		}
		rev, err := strconv.Atoi(splitSecretName[5][1:])
		if err != nil || rev <= 0 {
			return 0, errors.Errorf("unexpected format for Helm release revision %s", secretName)
		}
		return uint64(rev), nil
	}
	return 0, errors.Errorf("unexpected type %s for secret with name %s", secret.Type, secret.Name)
}

// fetchDeploymentIdentification retrieves the identifying information for this sensor deployment, using a mixture of
// secret mounts and information from the Kubernetes API server.
func fetchDeploymentIdentification(ctx context.Context, k8sClient kubernetes.Interface, helmManagedConfig *central.HelmManagedConfigInit) *storage.SensorDeploymentIdentification {
	ctx, cancel := context.WithTimeout(ctx, fetchClusterIdentificationTimeout)
	defer cancel()

	var deploymentIdentification storage.SensorDeploymentIdentification

	if err := populateFromServiceAccountTokenFile(&deploymentIdentification); err != nil {
		log.Warnf("Could not populate cluster identification from service account token file: %s", err)
	}
	if err := populateFromServiceAccountNamespaceFile(&deploymentIdentification); err != nil {
		log.Warnf("Could not populate cluster identification from service account namespace file: %s", err)
	}
	if err := populateFromKubernetes(ctx, k8sClient, helmManagedConfig, &deploymentIdentification); err != nil {
		log.Warnf("Could not populate cluster identification from Kubernetes API: %s", err)
	}

	return &deploymentIdentification
}
