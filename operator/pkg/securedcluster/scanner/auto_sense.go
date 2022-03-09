package scanner

import (
	"context"

	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
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
		return !siblingCentralPresent, nil
	case platform.LocalScannerComponentDisabled:
		return false, nil
	}

	return false, errors.Errorf("invalid spec.scanner.scannerComponent %q", scannerComponent)
}

func isSiblingCentralPresent(ctx context.Context, client ctrlClient.Client, namespace string) (bool, error) {
	list := &platform.CentralList{}
	if err := client.List(ctx, list, ctrlClient.InNamespace(namespace)); err != nil {
		return false, errors.Wrapf(err, "cannot list centrals in namespace %q", namespace)
	}
	return len(list.Items) > 0, nil
}
