package sensor

import (
	"bytes"
	"context"
	"os"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/namespaces"
	"gopkg.in/square/go-jose.v2/jwt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	namespaceFile = `/run/secrets/kubernetes.io/serviceaccount/namespace`
	//#nosec G101 -- This is a false positive
	tokenFile = `/run/secrets/kubernetes.io/serviceaccount/token`

	namespaceClaimKey        = `kubernetes.io/serviceaccount/namespace`
	serviceAccountIDClaimKey = `kubernetes.io/serviceaccount/service-account.uid`

	fetchClusterIdentificationTimeout = 10 * time.Second
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
func populateFromKubernetes(ctx context.Context, k8sClient kubernetes.Interface, out *storage.SensorDeploymentIdentification) error {
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

	return errResult
}

// fetchDeploymentIdentification retrieves the identifying information for this sensor deployment, using a mixture of
// secret mounts and information from the Kubernetes API server.
func fetchDeploymentIdentification(ctx context.Context, k8sClient kubernetes.Interface) *storage.SensorDeploymentIdentification {
	ctx, cancel := context.WithTimeout(ctx, fetchClusterIdentificationTimeout)
	defer cancel()

	var deploymentIdentification storage.SensorDeploymentIdentification

	if err := populateFromServiceAccountTokenFile(&deploymentIdentification); err != nil {
		log.Warnf("Could not populate cluster identification from service account token file: %s", err)
	}
	if err := populateFromServiceAccountNamespaceFile(&deploymentIdentification); err != nil {
		log.Warnf("Could not populate cluster identification from service account namespace file: %s", err)
	}
	if err := populateFromKubernetes(ctx, k8sClient, &deploymentIdentification); err != nil {
		log.Warnf("Could not populate cluster identification from Kubernetes API: %s", err)
	}

	return &deploymentIdentification
}
