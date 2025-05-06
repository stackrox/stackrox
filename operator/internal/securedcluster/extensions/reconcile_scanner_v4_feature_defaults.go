package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common/defaulting"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	annotationKey = defaulting.FeatureDefaultKeyScannerV4
	fieldOwner    = "stackrox-operator"
)

// This extension's purpose is to
//
//   1. apply defaults by mutating the SecuredCluster spec as a prerequisite for the value translator
//   2. persist any implicit Scanner V4 AutoSense|Disabled setting in the SecuredCluster annotations for later usage during upgrade-reconcilliations.
//

func ReconcileScannerV4FeatureDefaultsExtension(client ctrlClient.Client) extensions.ReconcileExtension {
	return wrapExtension(reconcileScannerV4FeatureDefaults, client, nil)
}

func reconcileScannerV4FeatureDefaults(
	ctx context.Context, securedCluster *platform.SecuredCluster, client ctrlClient.Client, _ ctrlClient.Reader,
	_ func(updateStatusFunc), logger logr.Logger) error {
	logger = logger.WithName("extension-feature-defaults") // Already using a generic log name due to the planned generalization of the feature-defaults extension.

	if securedCluster.GetDeletionTimestamp() != nil {
		logger.Info("skipping extension run due to deletionTimestamp being present on SecuredCluster custom resource")
		return nil
	}

	scannerV4Spec := initializedDeepCopy(securedCluster.Spec.ScannerV4)
	componentPolicy, usedDefaulting := defaulting.SecuredClusterScannerV4ComponentPolicy(logger, &securedCluster.Status, securedCluster.GetAnnotations(), scannerV4Spec)
	if !usedDefaulting {
		// User provided an explicit choice, nothing to do in this extension.
		return nil
	}

	// User is relying on defaults. Set in-memory default and persist corresponding annotation.

	if securedCluster.Annotations == nil {
		securedCluster.Annotations = make(map[string]string)
	}
	if securedCluster.Annotations[annotationKey] != string(componentPolicy) {
		// Update feature default setting.
		// We do this immediately during (first-time) execution of this extension to make sure
		// that this information is already persisted in the Kubernetes resource before we
		// can realistically end up in a situation where reconcilliation might need to be retried.
		if err := patchSecuredClusterAnnotation(ctx, logger, client, securedCluster, annotationKey, string(componentPolicy)); err != nil {
			return err
		}
	}

	// Mutates SecuredCluster spec for the following reconciler extensions and for the translator -- this is not persisted on the cluster.
	// Note that we need to mutate SecuredCluster's spec after the patching, because otherwise it would be overwritten again,
	// when -- as part of the patching -- the resulting cluster resource gets pulled and the provided `securedCluster` is updated based on the
	// cluster version.
	scannerV4Spec.ScannerComponent = &componentPolicy
	securedCluster.Spec.ScannerV4 = scannerV4Spec
	return nil
}

func initializedDeepCopy(spec *platform.LocalScannerV4ComponentSpec) *platform.LocalScannerV4ComponentSpec {
	if spec == nil {
		return &platform.LocalScannerV4ComponentSpec{}
	}
	return spec.DeepCopy()
}

func patchSecuredClusterAnnotation(ctx context.Context, logger logr.Logger, client ctrlClient.Client, securedCluster *platform.SecuredCluster, key string, val string) error {
	// MergeFromWithOptimisticLock causes the resourceVersion to be checked prior to patching.
	origSecuredCluster := securedCluster.DeepCopy()
	patch := ctrlClient.MergeFromWithOptions(origSecuredCluster, ctrlClient.MergeFromWithOptimisticLock{})
	securedCluster.Annotations[key] = val
	err := client.Patch(ctx, securedCluster, patch)
	if err != nil {
		return err
	}

	logger.Info("patched SecuredCluster object annotation",
		"annotationKey", key,
		"annotationValue", val,
		"oldResourceVersion", origSecuredCluster.GetResourceVersion(),
		"newResourceVersion", securedCluster.GetResourceVersion())
	return nil
}
