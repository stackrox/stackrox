package common

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
)

func equalAndNonempty(a, b string) bool {
	return a != "" && a == b
}
func distinctAndNonempty(a, b string) bool {
	return a != "" && b != "" && a != b
}

// CheckConnReplace checks if a connection with a deployment identification of newId may replace an (active) connection
// with a  deployment identification of oldId.
// We only allow a replacement if we're confident it's coming from the same deployment.
func CheckConnReplace(newID, oldID *storage.SensorDeploymentIdentification) error {
	if newID == nil || oldID == nil {
		// Without cluster identification, default to the previous behavior, which is to always allow replacements.
		return nil
	}

	// First check by namespace. Even if we might be talking about namespaces in two different clusters, prefer this
	// check as it results in the most descriptive error message.
	if distinctAndNonempty(newID.GetAppNamespace(), oldID.GetAppNamespace()) {
		return errors.Errorf("new connection for cluster is coming from a different namespace (%s) than the current one (namespace %s); please take down the deployment in %s first", newID.GetAppNamespace(), oldID.GetAppNamespace(), oldID.GetAppNamespace())
	}
	// Try to identify the cluster by the UID of the `kube-system` or `default` namespace. These namespaces should generally not be deletable,
	// and hence may be used to distinguish cluster (even if it does not provide us with a means of identification suitable for human consumption).
	if distinctAndNonempty(newID.GetSystemNamespaceId(), oldID.GetSystemNamespaceId()) || distinctAndNonempty(newID.GetDefaultNamespaceId(), oldID.GetDefaultNamespaceId()) {
		return errors.Errorf("a sensor is already active from node with name: %s; please take down the old deployment first", oldID.GetK8SNodeName())
	}

	// Lastly, look at the UID of the `stackrox` namespace (or whatever namespace we deployed to) or the `stackrox` service account, respectively.
	if distinctAndNonempty(newID.GetAppNamespaceId(), oldID.GetAppNamespaceId()) || distinctAndNonempty(newID.GetAppServiceaccountId(), oldID.GetAppServiceaccountId()) {
		// If we _know_ that we are in the same cluster _and_ a namespace of the same name, we know that the old deployment has
		// been taken down. Hence, only return an error if we aren't sure that this is the case.
		if equalAndNonempty(newID.GetAppNamespace(), oldID.GetAppNamespace()) &&
			(equalAndNonempty(newID.GetSystemNamespaceId(), oldID.GetSystemNamespaceId()) ||
				equalAndNonempty(newID.GetDefaultNamespaceId(), oldID.GetDefaultNamespaceId())) {
			return nil
		}

		return errors.Errorf("new connection for cluster is coming from a different instance of namespace %s, please take down the old deployment first", newID.GetAppNamespace())
	}

	return nil
}
