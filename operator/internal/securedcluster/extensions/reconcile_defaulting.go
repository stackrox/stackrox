package extensions

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/securedcluster/defaults"
	"github.com/stackrox/rox/pkg/features"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var defaultingFlows = []defaults.SecuredClusterDefaultingFlow{
	defaults.SecuredClusterStaticDefaults, // Must go first
	defaults.SecuredClusterScannerV4DefaultingFlow,
}

// FeatureDefaultingExtension executes "defaulting flows". A Secured Cluster defaulting flow is of type
// defaulting.SecuredClusterDefaultingFlow, which is essentially a function that acts on
// `status`, `metadata.annotations` as well as `spec` and `defaults` (both of type `SecuredClusterSpec`)
// of a SecuredCluster CR.
//
// A defaulting flow shall
//   - derive default values based on 'status', 'annotations' and 'spec' and store them in 'defaults'.
//   - add a new annotation in order to persist current defaulting choices.
func FeatureDefaultingExtension(client ctrlClient.Client) extensions.ReconcileExtension {
	return func(ctx context.Context, u *unstructured.Unstructured, _ func(extensions.UpdateStatusFunc), l logr.Logger) error {
		return reconcileFeatureDefaults(ctx, client, u, l)
	}
}

func reconcileFeatureDefaults(ctx context.Context, client ctrlClient.Client, u *unstructured.Unstructured, logger logr.Logger) error {
	logger = logger.WithName("extension-feature-defaults")
	if u.GroupVersionKind() != platform.SecuredClusterGVK {
		logger.Error(errUnexpectedGVK, "unable to reconcile SecuredCluster", "expectedGVK", platform.SecuredClusterGVK, "actualGVK", u.GroupVersionKind())
		return errUnexpectedGVK
	}

	if u.GetDeletionTimestamp() != nil {
		logger.Info("skipping extension run due to deletionTimestamp being present on SecuredCluster custom resource")
		return nil
	}

	securedCluster := platform.SecuredCluster{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &securedCluster); err != nil {
		return errors.Wrap(err, "converting unstructured object to SecuredCluster")
	}

	err := setDefaultsAndPersist(ctx, logger, &securedCluster, client)
	if err != nil {
		return err
	}

	u.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(&securedCluster)
	if err != nil {
		return errors.Wrap(err, "converting SecuredCluster to unstructured object after extension execution")
	}

	if err := platform.AddSecuredClusterDefaultsToUnstructured(u, &securedCluster); err != nil {
		return errors.Wrap(err, "enriching unstructured SecuredCluster object with defaults")
	}

	return nil
}

func setDefaultsAndPersist(ctx context.Context, logger logr.Logger, securedCluster *platform.SecuredCluster, client ctrlClient.Client) error {
	effectiveDefaultingFlows := defaultingFlows
	if features.AdmissionControllerConfig.Enabled() {
		effectiveDefaultingFlows = append(effectiveDefaultingFlows, defaults.SecuredClusterAdmissionControllerDefaultingFlow)
	}

	baseSecuredCluster := securedCluster.DeepCopy()
	patch := ctrlClient.MergeFrom(baseSecuredCluster)

	// This may update securedCluster.Defaults and securedCluster's embedded annotations.
	for _, flow := range effectiveDefaultingFlows {
		if err := executeSingleDefaultingFlow(logger, securedCluster, flow); err != nil {
			return err
		}
	}
	securedClusterDefaults := securedCluster.Defaults

	// We persist the annotations immediately during (first-time) execution of this extension to make sure
	// that this information is already persisted in the Kubernetes resource before we
	// can realistically end up in a situation where reconcilliation might need to be retried.
	//
	// This updates securedCluster both on the cluster and in memory, which is crucial since this object is used for the final
	// updating within helm-operator and we have concurrently running controllers (the status controller),
	// whose changes we must preserve.
	err := client.Patch(ctx, securedCluster, patch)
	if err != nil {
		return errors.Wrap(err, "patching SecuredCluster annotations")
	}

	logger.Info("patched SecuredCluster object",
		"oldResourceVersion", baseSecuredCluster.GetResourceVersion(),
		"newResourceVersion", securedCluster.GetResourceVersion(),
	)

	// Retain the defaults, which are not in the patched object after cluster refresh.
	securedCluster.Defaults = securedClusterDefaults
	return nil
}

func executeSingleDefaultingFlow(logger logr.Logger, securedCluster *platform.SecuredCluster, flow defaults.SecuredClusterDefaultingFlow) error {
	logger = logger.WithName(fmt.Sprintf("defaulting-flow-%s", flow.Name))
	annotations := securedCluster.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// to be thrown away after defaulting flow execution.
	spec := securedCluster.Spec.DeepCopy()
	status := securedCluster.Status.DeepCopy()

	err := flow.DefaultingFunc(logger, status, annotations, spec, &securedCluster.Defaults)
	if err != nil {
		return errors.Wrapf(err, "SecuredCluster defaulting flow %s failed", flow.Name)
	}
	securedCluster.SetAnnotations(annotations)

	return nil
}
