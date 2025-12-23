package extensions

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/central/defaults"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var defaultingFlows = []defaults.CentralDefaultingFlow{
	defaults.CentralStaticDefaults, // Must go first.
	defaults.CentralScannerV4DefaultingFlow,
	defaults.CentralDBPersistenceDefaultingFlow,
}

// FeatureDefaultingExtension executes "defaulting flows". A Central defaulting flow is of type
// defaulting.CentralDefaultingFlow, which is essentially a function that acts on
// `status`, `metadata.annotations` as well as `spec` and `defaults` (both of type `CentralSpec`)
// of a Central CR.
//
// A defaulting flow shall
//   - derive default values based on 'status', 'annotations' and 'spec' and store them in 'defaults'.
//   - optionally, add a new annotation in order to persist current defaulting choices.
func FeatureDefaultingExtension(client ctrlClient.Client) extensions.ReconcileExtension {
	return func(ctx context.Context, u *unstructured.Unstructured, _ func(extensions.UpdateStatusFunc), l logr.Logger) error {
		return reconcileFeatureDefaults(ctx, client, u, l)
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

	err := setDefaultsAndPersist(ctx, logger, u, &central, client)
	if err != nil {
		return err
	}

	if err := platform.AddCentralDefaultsToUnstructured(u, &central); err != nil {
		return errors.Wrap(err, "enriching unstructured Central object with defaults")
	}

	return nil
}

// This may update central.Defaults and central's embedded annotations in the unstructured u -- NOT in central.
func setDefaultsAndPersist(ctx context.Context, logger logr.Logger, u *unstructured.Unstructured, central *platform.Central, client ctrlClient.Client) error {
	uBase := u.DeepCopy()
	uBaseGeneration := uBase.GetGeneration()
	patch := ctrlClient.MergeFrom(uBase)

	for _, flow := range defaultingFlows {
		if err := executeSingleDefaultingFlow(logger, u, central, flow); err != nil {
			return err
		}
	}

	if reflect.DeepEqual(uBase.Object, u.Object) {
		return nil
	}

	// We persist the annotations immediately during (first-time) execution of this extension to make sure
	// that this information is already persisted in the Kubernetes resource before we
	// can realistically end up in a situation where reconcilliation might need to be retried.
	//
	// This updates central both on the cluster and in memory, which is crucial since this object is used for the final
	// updating within helm-operator and we have concurrently running controllers (the status controller),
	// whose changes we must preserve.
	logger.Info("updating defaulting annotations on Central object")
	err := client.Patch(ctx, u, patch)
	if err != nil {
		return errors.Wrap(err, "patching Central annotations")
	}
	logger.Info("patched Central object",
		"oldResourceVersion", uBase.GetResourceVersion(),
		"newResourceVersion", u.GetResourceVersion(),
	)

	// If we would not react to a generation mismatch here, this effectively means that the CR spec
	// currently under reconciliation has changed during the reconciliation flow, specifically during
	// execution of pre-extensions.
	// This might not pose a problem given that by our convention the defaulting extension is
	// expected to run first, but it is cleaner to just abort reconciliation and start over with the new spec.
	uGeneration := u.GetGeneration()
	if uGeneration != uBaseGeneration {
		return fmt.Errorf("Central resource spec was modified (generation: %d -> %d), aborting reconciliation to start over with new spec",
			uBaseGeneration, uGeneration)
	}

	return nil
}

// Defaulting flows have two side-effects:
// 1. They may update the metadata annotations (to be persisted on the cluster); this is happening in the unstructured u.
// 2. They may update central.Defaults (they only exist in-memory, not on the cluster); this is happening in the typed central object.
func executeSingleDefaultingFlow(logger logr.Logger, u *unstructured.Unstructured, central *platform.Central, flow defaults.CentralDefaultingFlow) error {
	logger = logger.WithName(fmt.Sprintf("defaulting-flow-%s", flow.Name))
	annotations := u.GetAnnotations()
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
	u.SetAnnotations(annotations)

	return nil
}
