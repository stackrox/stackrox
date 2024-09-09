package scanner

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlLog "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// clusterVersionDefaultName is a default name for the auto created ClusterVersion k8s custom resource on OpenShift.
	clusterVersionDefaultName = "version"
)

// AutoSenseResult represents the configurations which can be auto-sensed.
type AutoSenseResult struct {
	// DeployScannerResources indicates that Scanner resources should be deployed by the SecuredCluster controller.
	// Inside the same namespace the existing Scanner instance should be used.
	DeployScannerResources bool
	// EnableLocalImageScanning enables the local image scanning feature in Sensor. If this setting is disabled Sensor
	// will not scan images locally.
	EnableLocalImageScanning bool
}

// AutoSenseLocalScannerConfig detects whether the local scanner should be deployed and/or used by sensor.
// Takes into account the setting in provided SecuredCluster CR as well as the presence of a Central instance in the same namespace.
// Modifies the provided SecuredCluster object to set a default Spec.Scanner if missing.
func AutoSenseLocalScannerConfig(ctx context.Context, client ctrlClient.Client, s platform.SecuredCluster) (AutoSenseResult, error) {
	SetScannerDefaults(&s.Spec)
	scannerComponent := *s.Spec.Scanner.ScannerComponent

	return autoSenseScanner(ctx, client, scannerComponent, s.Namespace)
}

// AutoSenseLocalScannerV4Config detects whether the local Scanner V4 should be deployed and/or used by sensor.
// Takes into account the setting in provided SecuredCluster CR as well as the presence of a Central instance in the same namespace.
// Modifies the provided SecuredCluster object to set a default Spec.ScannerV4 if missing.
func AutoSenseLocalScannerV4Config(ctx context.Context, client ctrlClient.Client, s platform.SecuredCluster) (AutoSenseResult, error) {
	SetScannerV4Defaults(&s.Spec)
	scannerV4ComponentPolicy := *s.Spec.ScannerV4.ScannerComponent

	return autoSenseScannerV4(ctx, client, scannerV4ComponentPolicy, s.GetNamespace())

}

func autoSenseScanner(ctx context.Context, client ctrlClient.Client, scannerComponent platform.LocalScannerComponentPolicy, namespace string) (AutoSenseResult, error) {
	switch scannerComponent {
	case platform.LocalScannerComponentAutoSense:
		return autoSense(ctx, client, namespace)
	case platform.LocalScannerComponentDisabled:
		return AutoSenseResult{}, nil
	}

	return AutoSenseResult{}, errors.Errorf("invalid scannerComponent setting: %q", scannerComponent)
}

func autoSenseScannerV4(ctx context.Context, client ctrlClient.Client, deploymentPolicy platform.LocalScannerV4ComponentPolicy, namespace string) (AutoSenseResult, error) {
	switch deploymentPolicy {
	case platform.LocalScannerV4ComponentAutoSense:
		return autoSense(ctx, client, namespace)
	case platform.LocalScannerV4ComponentDisabled, platform.LocalScannerV4ComponentDefault:
		return AutoSenseResult{}, nil
	}

	return AutoSenseResult{}, errors.Errorf("invalid Scanner V4 deployment setting: %q", deploymentPolicy)
}

func autoSense(ctx context.Context, client ctrlClient.Client, namespace string) (AutoSenseResult, error) {
	siblingCentralPresent, err := isSiblingCentralPresent(ctx, client, namespace)
	if err != nil {
		return AutoSenseResult{}, errors.Wrap(err, "detecting presence of a Central CR in the same namespace")
	}
	isOpenShift, err := isRunningOnOpenShift(ctx, client)
	if err != nil {
		return AutoSenseResult{}, errors.Wrap(err, "cannot fetch OpenShift ClusterVersion resource")
	}
	if !isOpenShift {
		return AutoSenseResult{}, nil
	}
	return AutoSenseResult{
		// Only deploy scanner resource if Central is not available in the same namespace.
		DeployScannerResources:   !siblingCentralPresent,
		EnableLocalImageScanning: true,
	}, nil
}

func isRunningOnOpenShift(ctx context.Context, client ctrlClient.Client) (bool, error) {
	log := ctrlLog.FromContext(ctx)

	clusterVersion := &unstructured.Unstructured{}
	clusterVersion.SetKind("ClusterVersion")
	clusterVersion.SetAPIVersion("config.openshift.io/v1")
	key := ctrlClient.ObjectKey{Name: clusterVersionDefaultName}

	err := client.Get(ctx, key, clusterVersion)
	if err != nil && k8sErrors.IsNotFound(err) {
		log.Info(fmt.Sprintf("OpenShift ClusterVersion kind is present, but its %q object was not found (cluster not ready?)", clusterVersionDefaultName))
		return false, err
	} else if err != nil && meta.IsNoMatchError(err) {
		log.Info("Running on Kubernetes, OpenShift ClusterVersion kind does not exist")
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
