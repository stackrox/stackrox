package scanner

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlLog "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// ClusterVersionDefaultName is a default name for the auto created ClusterVersion k8s custom resource on OpenShift.
	clusterVersionDefaultName = "version"
)

// AutoSenseLocalScannerSupport detects whether the local scanner should be enabled or not.
// Takes into account the setting in provided SecuredCluster CR as well as the presence of a Central instance in the same namespace.
// Modifies the provided SecuredCluster object to set a default Spec.Scanner if missing.
func AutoSenseLocalScannerSupport(ctx context.Context, client ctrlClient.Client, s platform.SecuredCluster) (bool, error) {
	SetScannerDefaults(&s.Spec)
	scannerComponent := *s.Spec.Scanner.ScannerComponent

	switch scannerComponent {
	case platform.LocalScannerComponentAutoSense:
		siblingCentralPresent, err := isSiblingCentralPresent(ctx, client, s.GetNamespace())
		if err != nil {
			return false, errors.Wrap(err, "detecting presence of a Central CR in the same namespace")
		}
		isOpenShift, err := isRunningOnOpenShift(ctx, client)
		if err != nil {
			return false, errors.Wrap(err, "cannot fetch OpenShift ClusterVersion resource")
		}
		if !isOpenShift {
			return false, nil
		}
		return !siblingCentralPresent, nil
	case platform.LocalScannerComponentDisabled:
		return false, nil
	}

	return false, errors.Errorf("invalid spec.scanner.scannerComponent %q", scannerComponent)
}

func isRunningOnOpenShift(ctx context.Context, client ctrlClient.Client) (bool, error) {
	log := ctrlLog.FromContext(ctx)

	clusterVersion := &unstructured.Unstructured{}
	clusterVersion.SetKind("ClusterVersion")
	clusterVersion.SetAPIVersion("config.openshift.io/v1")
	clusterVersion.SetName("name")
	key := ctrlClient.ObjectKey{Name: clusterVersionDefaultName}

	err := client.Get(ctx, key, clusterVersion)
	if err != nil && k8sErrors.IsNotFound(err) {
		log.Info("Running on Kubernetes, OpenShift ClusterVersion was not found")
		return false, err
	} else if err != nil && meta.IsNoMatchError(err) {
		log.Info(fmt.Sprintf("Running on Kubernetes, resource OpenShift ClusterVersion %q does not exist", clusterVersionDefaultName))
		return false, nil
	} else if err != nil {
		log.Error(err, "Failed to get ClusterVersion")
		return false, err
	}

	return true, nil
}

func isSiblingCentralPresent(ctx context.Context, client ctrlClient.Client, namespace string) (bool, error) {
	list := &platform.CentralList{}
	if err := client.List(ctx, list, ctrlClient.InNamespace(namespace)); err != nil {
		return false, errors.Wrapf(err, "cannot list centrals in namespace %q", namespace)
	}
	return len(list.Items) > 0, nil
}
