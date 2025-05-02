package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common/defaulting"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// This extension's purpose is to
//
//   1. apply defaults by mutating the SecuredCluster spec as a prerequisite for the value translator
//   2. persist any implicit Scanner AutoSense|Disabled setting in the SecuredCluster annotations for later usage during upgrade-reconcilliations.
//

func ReconcileScannerFeatureDefaultsExtension(client ctrlClient.Client) extensions.ReconcileExtension {
	return wrapExtension(reconcileScannerFeatureDefaults, client, nil)
}

func reconcileScannerFeatureDefaults(
	ctx context.Context, securedCluster *platform.SecuredCluster, client ctrlClient.Client, _ ctrlClient.Reader,
	_ func(updateStatusFunc), logger logr.Logger) error {

	spec := &platform.LocalScannerComponentSpec{}
	if securedCluster.Spec.Scanner != nil {
		spec = securedCluster.Spec.Scanner.DeepCopy()
	}

	componentPolicy, usedDefaulting := defaulting.SecuredClusterScannerComponentPolicy(logger, &securedCluster.Status, securedCluster.GetAnnotations(), spec)
	if !usedDefaulting {
		// User provided an explicit choice, nothing to do in this extension.
		return nil
	}

	// User is relying on defaults. Compute default and persist corresponding annotation.

	spec.ScannerComponent = &componentPolicy
	if securedCluster.Annotations == nil {
		securedCluster.Annotations = make(map[string]string)
	}
	if securedCluster.Annotations[defaulting.FeatureDefaultKeySecuredClusterScanner] != string(componentPolicy) {
		// Update feature default setting.
		// We do this immediately during (first-time) execution of this extension to make sure
		// that this information is already persisted in the Kubernetes resource before we
		// can realistically end up in a situation where reconcilliation might need to be retried.
		err := patchSecuredClusterAnnotation(ctx, logger, client, securedCluster, defaulting.FeatureDefaultKeySecuredClusterScanner, string(componentPolicy))
		if err != nil {
			return err
		}
	}

	// Mutates SecuredCluster spec for the following reconciler extensions and for the translator -- this is not persisted on the cluster.
	securedCluster.Spec.Scanner = spec
	return nil
}
