package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ReconcileProductVersionStatusExtension returns a reconcile extension that ensures an up-to-date product version status.
func ReconcileProductVersionStatusExtension(version string) extensions.ReconcileExtension {
	return func(ctx context.Context, obj *unstructured.Unstructured, statusUpdater func(statusFunc extensions.UpdateStatusFunc), _ logr.Logger) error {
		if obj.GetDeletionTimestamp() != nil {
			return nil
		}

		statusUpdater(func(uSt *unstructured.Unstructured) bool {
			productVersionChanged := updateProductVersion(uSt, version)
			reconciledVersionChanged := updateReconciledVersion(uSt)
			observedGenChanged := updateObservedGeneration(uSt, obj.GetGeneration())
			return productVersionChanged || reconciledVersionChanged || observedGenChanged
		})
		return nil
	}
}

func updateProductVersion(uSt *unstructured.Unstructured, version string) bool {
	pv, _, _ := unstructured.NestedString(uSt.Object, "productVersion")
	if pv == version {
		return false
	}
	if uSt.Object == nil {
		uSt.Object = make(map[string]interface{})
	}
	if err := unstructured.SetNestedField(uSt.Object, version, "productVersion"); err != nil {
		return false
	}
	return true
}

func updateReconciledVersion(uSt *unstructured.Unstructured) bool {
	// Get the deployed release version from status.deployedRelease.version
	deployedVersion, _, _ := unstructured.NestedString(uSt.Object, "deployedRelease", "version")

	// Get the current reconciledVersion
	currentReconciledVersion, _, _ := unstructured.NestedString(uSt.Object, "reconciledVersion")

	// Only update if the deployed version differs from the current reconciled version
	if deployedVersion == currentReconciledVersion {
		return false
	}

	if uSt.Object == nil {
		uSt.Object = make(map[string]interface{})
	}

	// Set reconciledVersion to the deployed release version
	if err := unstructured.SetNestedField(uSt.Object, deployedVersion, "reconciledVersion"); err != nil {
		return false
	}
	return true
}

func updateObservedGeneration(uSt *unstructured.Unstructured, generation int64) bool {
	// Get the current observedGeneration
	currentObservedGen, _, _ := unstructured.NestedInt64(uSt.Object, "observedGeneration")

	// Only update if generation changed
	if currentObservedGen == generation {
		return false
	}

	if uSt.Object == nil {
		uSt.Object = make(map[string]interface{})
	}

	// Set observedGeneration to the current generation
	if err := unstructured.SetNestedField(uSt.Object, generation, "observedGeneration"); err != nil {
		return false
	}
	return true
}
