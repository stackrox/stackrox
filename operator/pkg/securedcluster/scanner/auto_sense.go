package scanner

import (
	"context"

	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// AutoSenseResult represents the configurations which can be auto-sensed.
type AutoSenseResult struct {
	// DeployScannerResources indicates that Scanner resources should be deployed. If a Central instance is existing
	// inside the same namespace the existing Scanner instance should be used.
	DeployScannerResources bool
	// EnableLocalImageScanning enables the local image scanning feature in Sensor. If this setting is disabled Sensor
	// will not scan images locally.
	EnableLocalImageScanning bool
}

var (
	disabledAutoSenseResult = AutoSenseResult{
		DeployScannerResources:   false,
		EnableLocalImageScanning: false,
	}
)

// AutoSenseLocalScannerConfig detects whether the local scanner should be deployed or not.
// Takes into account the setting in provided SecuredCluster CR as well as the presence of a Central instance in the same namespace.
// Modifies the provided SecuredCluster object to set a default Spec.Scanner if missing.
func AutoSenseLocalScannerConfig(ctx context.Context, client ctrlClient.Client, s platform.SecuredCluster) (AutoSenseResult, error) {
	SetScannerDefaults(&s.Spec)
	scannerComponent := *s.Spec.Scanner.ScannerComponent

	switch scannerComponent {
	case platform.LocalScannerComponentAutoSense:
		siblingCentralPresent, err := IsSiblingCentralPresent(ctx, client, s.GetNamespace())
		if err != nil {
			return disabledAutoSenseResult, errors.Wrap(err, "detecting presence of a Central CR in the same namespace")
		}

		return AutoSenseResult{
			// Only deploy scanner resource if Central is not available in the same namespace.
			DeployScannerResources:   !siblingCentralPresent,
			EnableLocalImageScanning: true,
		}, nil
	case platform.LocalScannerComponentDisabled:
		return disabledAutoSenseResult, nil
	}

	return disabledAutoSenseResult, errors.Errorf("invalid spec.scanner.scannerComponent %q", scannerComponent)
}

func IsSiblingCentralPresent(ctx context.Context, client ctrlClient.Client, namespace string) (bool, error) {
	list := &platform.CentralList{}
	if err := client.List(ctx, list, ctrlClient.InNamespace(namespace)); err != nil {
		return false, errors.Wrapf(err, "cannot list centrals in namespace %q", namespace)
	}
	return len(list.Items) > 0, nil
}
