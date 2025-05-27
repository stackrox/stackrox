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

var defaultingFlows = []defaulting.SecuredClusterDefaultingFlow{
	defaulting.SecuredClusterScannerV4DefaultingFlow,
}

// This extension executes "defaulting flows". A Secured Cluster defaulting flow is of type
// defaulting.SecuredClusterDefaultingFlow, which is essentially a function of type
//
//	func(
//	  logger logr.Logger,
//	  status *platform.SecuredClusterStatus,
//	  annotations map[string]string,
//	  spec *platform.SecuredClusterSpec,
//	  defaults *platform.SecuredClusterSpec) error
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

	err := executeDefaultingFlows(ctx, logger, &securedCluster, client)
	if err != nil {
		return err
	}

	updatedObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&securedCluster)
	if err != nil {
		return errors.Wrap(err, "converting SecuredCluster to unstructured object after extension execution")
	}
	u.Object = updatedObj

	if err := platform.AddSecuredClusterDefaultsToUnstructured(u, &securedCluster); err != nil {
		return errors.Wrap(err, "enriching unstructured SecuredCluster object with defaults")
	}

	return nil
}

func executeDefaultingFlows(ctx context.Context, logger logr.Logger, securedCluster *platform.SecuredCluster, client ctrlClient.Client) error {
	origSecuredCluster := securedCluster.DeepCopy()

	// This may update securedCluster.Defaults and securedCluster's embedded annotations.
	for _, flow := range defaultingFlows {
		if err := executeSingleDefaultingFlow(logger, securedCluster, client, flow); err != nil {
			return err
		}
	}

	// We persist the annotations immediately during (first-time) execution of this extension to make sure
	// that this information is already persisted in the Kubernetes resource before we
	// can realistically end up in a situation where reconcilliation might need to be retried.
	newResourceVersion, err := patchSecuredClusterAnnotations(ctx, logger, client, origSecuredCluster, securedCluster.GetAnnotations())
	if err != nil {
		return errors.Wrap(err, "patching SecuredCluster annotations")
	}
	securedCluster.SetResourceVersion(newResourceVersion)
	return nil
}

func executeSingleDefaultingFlow(logger logr.Logger, securedCluster *platform.SecuredCluster, client ctrlClient.Client, flow defaulting.SecuredClusterDefaultingFlow) error {
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

func patchSecuredClusterAnnotations(ctx context.Context, logger logr.Logger, client ctrlClient.Client, securedCluster *platform.SecuredCluster, annotations map[string]string) (string, error) {
	// MergeFromWithOptimisticLock causes the resourceVersion to be checked prior to patching.
	patch := ctrlClient.MergeFromWithOptions(securedCluster, ctrlClient.MergeFromWithOptimisticLock{})
	newSecuredCluster := securedCluster.DeepCopy()
	newSecuredCluster.SetAnnotations(annotations)
	if err := client.Patch(ctx, newSecuredCluster, patch); err != nil {
		return "", err
	}

	newResourceVersion := newSecuredCluster.GetResourceVersion()
	logger.Info("patched SecuredCluster object",
		"oldResourceVersion", securedCluster.GetResourceVersion(),
		"newResourceVersion", newResourceVersion)

	return newResourceVersion, nil
}
