package scanner

import (
	"context"

	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// AutoSenseLocalScannerSupport detects whether the local scanner is enabled or not. If a Central instance is found
// in the same namespace it returns false.
func AutoSenseLocalScannerSupport(ctx context.Context, client ctrlClient.Client, s platform.SecuredCluster) (bool, error) {
	if s.Spec.Scanner == nil {
		s.Spec.Scanner = &platform.LocalScannerComponentSpec{
			ScannerComponent: platform.LocalScannerComponentAutoSense.Pointer(),
		}
	}
	scannerComponent := s.Spec.Scanner.ScannerComponent

	siblingCentralPresent, err := isSiblingCentralPresent(ctx, client, s.GetNamespace())
	if err != nil {
		return false, errors.Wrap(err, "auto-sensing local scanner support")
	}

	switch *scannerComponent {
	case platform.LocalScannerComponentAutoSense:
		return !siblingCentralPresent, nil
	case platform.LocalScannerComponentDisabled:
		return false, nil
	}

	return false, errors.Errorf("invalid spec.scanner.scannerComponent %q", *scannerComponent)
}

func isSiblingCentralPresent(ctx context.Context, client ctrlClient.Client, namespace string) (bool, error) {
	list := &platform.CentralList{}
	if err := client.List(ctx, list, ctrlClient.InNamespace(namespace)); err != nil {
		return false, errors.Wrapf(err, "cannot list centrals in namespace %q", namespace)
	}
	return len(list.Items) > 0, nil
}
