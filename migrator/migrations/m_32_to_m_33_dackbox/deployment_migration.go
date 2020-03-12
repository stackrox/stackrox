package m32tom33

import (
	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/generated/storage"
)

func rewriteDeployments(db *badger.DB) error {
	// Collect the keys of all of the images we need to rewrite
	deploymentKeys, err := getKeysWithPrefix(deploymentBucketName, db)
	if err != nil {
		return err
	}

	// Collect the mappings that need to be added for all the deployments in the DB.
	mappings := make(map[string]SortedKeys)
	for _, key := range deploymentKeys {
		if err := collectMappingsFromDeploymentKey(key, db, mappings); err != nil {
			return err
		}
	}

	// Add all of the generated mappings to the DB.
	batch := db.NewWriteBatch()
	defer batch.Cancel()
	if err := writeMappings(batch, mappings); err != nil {
		return err
	}
	return batch.Flush()
}

func collectMappingsFromDeploymentKey(deploymentKey []byte, db *badger.DB, mappings map[string]SortedKeys) error {
	// Load the deployment for the key.
	var deployment storage.Deployment
	if exists, err := readProto(db, deploymentKey, &deployment); err != nil {
		return err
	} else if !exists {
		return nil
	}

	// Generate the keys for the cluster, namespace, deployment and images.
	clusterKey := getClusterKey(deployment.GetClusterId())
	namespaceKey := getNamespaceKey(deployment.GetNamespaceId())
	namespaceSACKey := getNamespaceSACKey(deployment.GetNamespace())
	imageKeys := make([][]byte, 0, len(deployment.GetContainers()))
	for _, container := range deployment.GetContainers() {
		imageKeys = append(imageKeys, getImageKey(container.GetImage().GetId()))
	}

	// Add the mappings between the objects to the map.
	mappings[string(clusterKey)], _ = mappings[string(clusterKey)].Insert(namespaceKey)
	mappings[string(clusterKey)], _ = mappings[string(clusterKey)].Insert(namespaceSACKey)
	mappings[string(namespaceKey)], _ = mappings[string(namespaceKey)].Insert(deploymentKey)
	mappings[string(namespaceSACKey)], _ = mappings[string(namespaceSACKey)].Insert(deploymentKey)
	mappings[string(deploymentKey)] = SortedCopy(imageKeys)

	return nil
}
