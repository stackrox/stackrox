package extensions

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/common/defaulting"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var defaultingFlows = []defaulting.CentralDefaultingFlow{
	defaulting.CentralScannerV4DefaultingFlow,
}

// This extension's purpose is to
//
//   1. apply defaults by mutating the Central object as a prerequisite for the value translator
//   2. persist any implicit Scanner V4 Enabled|Disabled setting in the Central annotations for later usage during upgrade-reconcilliations.
//

func ReconcilerExtensionFeatureDefaulting(client ctrlClient.Client) extensions.ReconcileExtension {
	return func(ctx context.Context, u *unstructured.Unstructured, _ func(extensions.UpdateStatusFunc), l logr.Logger) error {
		return reconcileScannerV4FeatureDefaults(ctx, client, u, l)
	}
}

func reconcileFeatureDefaults(ctx context.Context, client ctrlClient.Client, u *unstructured.Unstructured, logger logr.Logger) error {
	logger = logger.WithName("extension-feature-defaults")
	if u.GroupVersionKind() != platform.CentralGVK {
		logger.Error(errUnexpectedGVK, "unable to reconcile central", "expectedGVK", platform.CentralGVK, "actualGVK", u.GroupVersionKind())
		return errUnexpectedGVK
	}

	if u.GetDeletionTimestamp() != nil {
		logger.Info("skipping extension run due to deletionTimestamp being present on Central custom resource")
		return nil
	}

	central := platform.Central{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &central); err != nil {
		return errors.Wrap(err, "converting unstructured object to Central")
	}
	origCentral := central.DeepCopy()

	// Execute defaulting flows.
	// This may update central.Defaults and central's embedded annotations.
	for _, flow := range defaultingFlows {
		if err := executeDefaultingFlow(logger, &central, client, flow); err != nil {
			return err
		}
	}

	// We persist the annotations immediately during (first-time) execution of this extension to make sure
	// that this information is already persisted in the Kubernetes resource before we
	// can realistically end up in a situation where reconcilliation might need to be retried.
	newResourceVersion, err := patchCentralAnnotations(ctx, logger, client, origCentral, central.GetAnnotations())
	if err != nil {
		return errors.Wrap(err, "patching Central annotations")
	}
	central.SetResourceVersion(newResourceVersion)
	updatedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&central)
	if err != nil {
		return errors.Wrap(err, "converting Central to unstructured object after extension execution")
	}
	u.Object = updatedObj

	if err := platform.AddCentralDefaultsToUnstructured(u, &central); err != nil {
		return errors.Wrap(err, "enriching unstructured Central object with defaults")
	}

	return nil
}

func executeDefaultingFlow(logger logr.Logger, central *platform.Central, client ctrlClient.Client, flow defaulting.CentralDefaultingFlow) error {
	logger = logger.WithName(fmt.Sprintf("defaulting-flow-%s", flow.Name))
	annotations := central.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// to be thrown away after defaulting flow execution.
	spec := central.Spec.DeepCopy()
	status := central.Status.DeepCopy()

	err := flow.DefaultingFunc(logger, status, annotations, spec, &central.Defaults)
	if err != nil {
		return errors.Wrapf(err, "Central defaulting flow %s failed", flow.Name)
	}
	central.SetAnnotations(annotations)

	return nil
}

func patchCentralAnnotations(ctx context.Context, logger logr.Logger, client ctrlClient.Client, central *platform.Central, annotations map[string]string) (string, error) {
	// MergeFromWithOptimisticLock causes the resourceVersion to be checked prior to patching.
	patch := ctrlClient.MergeFromWithOptions(central, ctrlClient.MergeFromWithOptimisticLock{})
	newCentral := central.DeepCopy()
	newCentral.SetAnnotations(annotations)
	if err := client.Patch(ctx, newCentral, patch); err != nil {
		return "", err
	}

	newResourceVersion := newCentral.GetResourceVersion()
	logger.Info("patched Central object",
		"oldResourceVersion", central.GetResourceVersion(),
		"newResourceVersion", newResourceVersion)

	return newResourceVersion, nil
}
