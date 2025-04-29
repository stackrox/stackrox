package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common/defaulting"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	annotationKey = defaulting.FeatureDefaultKeyScannerV4
	fieldOwner    = "stackrox-operator"
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
	scannerV4Spec := initializedDeepCopy(central.Spec.ScannerV4)

	componentPolicy, usedDefaulting := defaulting.ScannerV4ComponentPolicy(logger, &central.Status, central.GetAnnotations(), scannerV4Spec)
	if !usedDefaulting {
		// User provided an explicit choice, nothing to do in this extension.
		return nil
	}

	// User is relying on defaults. Compute default and persist corresponding annotation.

	scannerV4Spec.ScannerComponent = &componentPolicy
	if central.Annotations == nil {
		central.Annotations = make(map[string]string)
	}
	if central.Annotations[annotationKey] == "" {
		// Update feature default setting.
		// We do this immediately during (first-time) execution of this extension to make sure
		// that this information is already persisted in the Kubernetes resource before we
		// can realistically end up in a situation where reconcilliation might need to be retried.
		err := patchCentralAnnotation(ctx, logger, client, central, annotationKey, string(componentPolicy))
		if err != nil {
			return err
		}
	}

	// Mutates Central spec for the following reconciler extensions and for the translator -- this is not persisted on the cluster.
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
	// Create a server-side-apply (SSA) patch which only patches the feature-default annotation.
	kind, apiVersion := getCentralKindAndVersion(central)
	centralPatch := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       kind,
			"apiVersion": apiVersion,
			"metadata": map[string]interface{}{
				"name":            central.GetName(),
				"namespace":       central.GetNamespace(),
				"resourceVersion": central.GetResourceVersion(),
				"annotations": map[string]interface{}{
					key: val,
				},
			},
		},
	}
	err := client.Patch(ctx, centralPatch, ctrlClient.Apply, ctrlClient.ForceOwnership, ctrlClient.FieldOwner(fieldOwner))
	if err != nil {
		return errors.Wrapf(err, "patching Central object with annotation %s=%s", key, val)
	}
	logger.Info("patched Central object annotation",
		"namespace", central.GetNamespace(),
		"central", central.GetName(),
		"annotationKey", key,
		"annotationValue", val,
		"oldResourceVersion", central.GetResourceVersion(),
		"newResourceVersion", centralPatch.GetResourceVersion())
	central.Annotations[key] = val
	central.SetResourceVersion(centralPatch.GetResourceVersion())
	return nil
}

func getCentralKindAndVersion(central *platform.Central) (kind, version string) {
	gvk := central.GetObjectKind().GroupVersionKind()
	kind = gvk.Kind
	version = gvk.GroupVersion().String()
	return
}
