package aggregation

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/storage"
)

// namespaceIdentifierFromDeployment returns an identifier used to uniquely identify the namespace of a deployment.
// It consists of the cluster id and the namespace name.
// It is a temporary hack, and not to be used outside the context of compliance aggregation.
func namespaceIdentifierFromDeployment(deployment *storage.Deployment) string {
	return fmt.Sprintf("%s/%s", deployment.GetClusterId(), deployment.GetNamespace())
}

// ClusterIDAndNameFromNamespaceIdentifier returns the cluster ID and namespace from the namespace identifier.
// It fails silently, returning an empty string if the identifier was invalid.
// It is a temporary hack, and not to be used outside the context of compliance aggregation.
func ClusterIDAndNameFromNamespaceIdentifier(identifier string) (clusterID, name string) {
	splitString := strings.Split(identifier, "/")
	clusterID = splitString[0]
	if len(splitString) > 1 {
		name = splitString[1]
	}
	return
}
