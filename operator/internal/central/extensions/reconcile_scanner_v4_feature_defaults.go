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
)

// This extension's purpose is to
//
//   1. apply defaults by mutating the Central spec as a prerequisite for the value translator
//   2. persist any implicit Scanner V4 Enabled|Disabled setting in the Central annotations for later usage during upgrade-reconcilliations.
//

func ReconcileScannerV4FeatureDefaultsExtension(client ctrlClient.Client) extensions.ReconcileExtension {
	return wrapExtension(reconcileScannerV4FeatureDefaults, client, nil)
}

func reconcileScannerV4FeatureDefaults(
	ctx context.Context, central *platform.Central, client ctrlClient.Client, _ ctrlClient.Reader,
	_ func(updateStatusFunc), logger logr.Logger) error {
	logger = logger.WithName("extension-feature-defaults") // Already using a generic log name due to the planned generalization of the feature-defaults extension.

	if central.GetDeletionTimestamp() != nil {
		logger.Info("skipping extension run due to deletionTimestamp being present on Central custom resource")
		return nil
	}

	scannerV4Spec := initializedDeepCopy(central.Spec.ScannerV4)
	componentPolicy, usedDefaulting := defaulting.ScannerV4ComponentPolicy(logger, &central.Status, central.GetAnnotations(), scannerV4Spec)
	if !usedDefaulting {
		// User provided an explicit choice, nothing to do in this extension.
		return nil
	}

	// User is relying on defaults. Set in-memory default and persist corresponding annotation.

	if central.Annotations == nil {
		central.Annotations = make(map[string]string)
	}
	if central.Annotations[annotationKey] != string(componentPolicy) {
		// Update feature default setting.
		// We do this immediately during (first-time) execution of this extension to make sure
		// that this information is already persisted in the Kubernetes resource before we
		// can realistically end up in a situation where reconcilliation might need to be retried.
		if err := patchCentralAnnotation(ctx, logger, client, central, annotationKey, string(componentPolicy)); err != nil {
			return err
		}
	}

	// Mutates Central spec for the following reconciler extensions and for the translator -- this is not persisted on the cluster.
	// Note that we need to mutate Central's spec after the patching, because otherwise it would be overwritten again,
	// when -- as part of the patching -- the resulting cluster resource gets pulled and the provided `central` is updated based on the
	// cluster version.
	scannerV4Spec.ScannerComponent = &componentPolicy
	central.Spec.ScannerV4 = scannerV4Spec
	return nil
}

func initializedDeepCopy(spec *platform.ScannerV4Spec) *platform.ScannerV4Spec {
	if spec == nil {
		return &platform.ScannerV4Spec{}
	}
	return spec.DeepCopy()
}

func patchCentralAnnotation(ctx context.Context, logger logr.Logger, client ctrlClient.Client, central *platform.Central, key string, val string) error {
	// MergeFromWithOptimisticLock causes the resourceVersion to be checked prior to patching.
	origCentral := central.DeepCopy()
	centralPatch := ctrlClient.MergeFromWithOptions(origCentral, ctrlClient.MergeFromWithOptimisticLock{})
	central.Annotations[key] = val
	err := client.Patch(ctx, central, centralPatch)
	if err != nil {
		return err
	}

	logger.Info("patched Central object annotation",
		"annotationKey", key,
		"annotationValue", val,
		"oldResourceVersion", origCentral.GetResourceVersion(),
		"newResourceVersion", central.GetResourceVersion())
	return nil
}
