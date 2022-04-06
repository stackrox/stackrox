package scanner

import (
	"context"
	"strings"

	osconfigv1 "github.com/openshift/api/config/v1"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	errorsK8s "k8s.io/apimachinery/pkg/api/errors"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlLog "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// ClusterVersionDefaultName is a default name for the auto created ClusterVersion k8s custom resource on OpenShift
	ClusterVersionDefaultName = "version"
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
		isOpenShift, err := isRunningOnOpenShift(ctx, client, s.GetNamespace())
		if err != nil {
			return false, errors.Wrap(err, "cannot fetch OpenShift ClusterVersion resource")
		}
		enableScanner := isOpenShift && !siblingCentralPresent
		return enableScanner, nil
	case platform.LocalScannerComponentDisabled:
		return false, nil
	}

	return false, errors.Errorf("invalid spec.scanner.scannerComponent %q", scannerComponent)
}

func isRunningOnOpenShift(ctx context.Context, client ctrlClient.Client, namespace string) (bool, error) {
	log := ctrlLog.FromContext(ctx)

	clusterVersion := &osconfigv1.ClusterVersion{}
	key := ctrlClient.ObjectKey{Namespace: namespace, Name: ClusterVersionDefaultName}
	err := client.Get(ctx, key, clusterVersion)
	if err != nil && errorsK8s.IsNotFound(err) {
		log.Error(err, "OpenShift ClusterVersion not found")
		return false, err
	} else if err != nil && strings.Contains(err.Error(), "no matches for kind") {
		log.Info("OpenShift ClusterVersion does not exist")
		return false, nil
	} else if err != nil {
		log.Error(err, "Failed to get ClusterVersion")
		return false, err
	}

	return clusterVersion.Spec.ClusterID != "", nil
}

func isSiblingCentralPresent(ctx context.Context, client ctrlClient.Client, namespace string) (bool, error) {
	list := &platform.CentralList{}
	if err := client.List(ctx, list, ctrlClient.InNamespace(namespace)); err != nil {
		return false, errors.Wrapf(err, "cannot list centrals in namespace %q", namespace)
	}
	return len(list.Items) > 0, nil
}
