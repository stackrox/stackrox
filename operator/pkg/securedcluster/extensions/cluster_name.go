package extensions

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// CheckClusterNameExtension is an extension that ensures the spec.clusterName and status.clusterName fields are
// in sync.
func CheckClusterNameExtension(client ctrlClient.Client) extensions.ReconcileExtension {
	return wrapExtension(checkClusterName, client)
}

func checkClusterName(_ context.Context, sc *platform.SecuredCluster, _ ctrlClient.Client, statusUpdater func(statusFunc updateStatusFunc), _ logr.Logger) error {
	if sc.DeletionTimestamp != nil {
		return nil // doesn't matter on deletion
	}
	if sc.Spec.ClusterName == "" {
		return errors.New("spec.clusterName is a required field")
	}
	if sc.Spec.ClusterName == sc.Status.ClusterName {
		return nil
	}
	if sc.Status.ClusterName != "" {
		return errors.Errorf("SecuredCluster instance was initially created with clusterName %q, but current value is %q. "+
			"Renaming clusters is not supported - you need to delete this object, and then recreate one with the correct cluster name.",
			sc.Status.ClusterName, sc.Spec.ClusterName)
	}

	statusUpdater(func(status *platform.SecuredClusterStatus) bool {
		status.ClusterName = sc.Spec.ClusterName
		return true
	})
	return nil
}
