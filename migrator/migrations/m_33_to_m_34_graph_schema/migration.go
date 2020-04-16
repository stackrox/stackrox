package m33tom34

import (
	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var (
	migration = types.Migration{
		StartingSeqNum: 33,
		VersionAfter:   storage.Version{SeqNum: 34},
		Run: func(databases *types.Databases) error {
			err := migrateSchema(databases.BadgerDB)
			if err != nil {
				return errors.Wrap(err, "updating dackbox graph schema")
			}
			return nil
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

/*
Changing the graph schema from:
Cluster -> (Namespace ID, Namespace SAC (Name))
Namespace ID -> Deployments
Namespace SAC (Name) -> Deployments

to:
Cluster -> Namespace ID
Namespace ID -> (Namespace SAC (Name), Deployments)
*/
func migrateSchema(db *badger.DB) error {
	// Remove the namespace sac keys from the cluster graph keys.
	err := removeNamespaceSACFromClusters(db)
	if err != nil {
		return err
	}

	// Move the namespace SAC keys that point to deployments under their matching namespace ID.
	err = moveNamespaceSACKeyToNamespace(db)
	if err != nil {
		return err
	}

	// Remove the namespace SAC keys that point to deployments.
	err = removeNamespaceSACMappings(db)
	if err != nil {
		return err
	}
	return nil
}

func removeNamespaceSACFromClusters(db *badger.DB) error {
	// We need to remove the namespace sac keys from the cluster graph keys.
	clusterGraphMappings, err := readMappings(db, getFullPrefix(clusterBucketName))
	if err != nil {
		return err
	}
	clusterGraphMappings = removeMappingsWithPrefix(namespaceSACBucketName, clusterGraphMappings)

	batch := db.NewWriteBatch()
	defer batch.Cancel()
	err = writeMappings(batch, clusterGraphMappings)
	if err != nil {
		return err
	}
	err = batch.Flush()
	if err != nil {
		return err
	}
	return nil
}

func moveNamespaceSACKeyToNamespace(db *badger.DB) error {
	deploymentIDToNamespaceID := make(map[string]string)
	namespaceIDToNamespaceName := make(map[string]string)

	// We need to add the namespace sac keys to the namespace graph keys.
	namespaceMappings, err := readMappings(db, getFullPrefix(namespaceBucketName))
	if err != nil {
		return err
	}

	// Map the namespace IDs by their deployments so we can match namespace names
	for namespaceID, values := range namespaceMappings {
		for _, value := range values {
			if hasPrefix(deploymentBucketName, value) {
				deploymentIDToNamespaceID[string(value)] = namespaceID
			}
		}
	}

	// We need to remove the namespace sac keys as forward keys.
	namespaceSACMappings, err := readMappings(db, getFullPrefix(namespaceSACBucketName))
	if err != nil {
		return err
	}

	// Map the namespace IDs by their deployments so we can match namespace names
	for namespaceName, values := range namespaceSACMappings {
		for _, value := range values {
			if !hasPrefix(deploymentBucketName, value) {
				continue
			}
			namespaceID, hasNamespaceID := deploymentIDToNamespaceID[string(value)]
			if !hasNamespaceID {
				continue
			}
			namespaceIDToNamespaceName[namespaceID] = namespaceName
		}
	}

	// update the namespace mappings to have the namespace names.
	for namespaceID, values := range namespaceMappings {
		namespaceName, hasNamespaceName := namespaceIDToNamespaceName[namespaceID]
		if !hasNamespaceName {
			// A namespace won't have a namespace name if it has no deployments, this is ok since we don't rely on
			// the mappings for anything above a deployment.
			continue
		}
		namespaceMappings[namespaceID], _ = values.Insert([]byte(namespaceName))
	}

	batch := db.NewWriteBatch()
	defer batch.Cancel()
	err = writeMappings(batch, namespaceMappings)
	if err != nil {
		return err
	}
	err = batch.Flush()
	if err != nil {
		return err
	}
	return nil
}

func removeNamespaceSACMappings(db *badger.DB) error {
	// We need to remove the namespace sac keys as forward keys.
	namespaceSACMappings, err := readMappings(db, getFullPrefix(namespaceSACBucketName))
	if err != nil {
		return err
	}

	batch := db.NewWriteBatch()
	defer batch.Cancel()
	for key := range namespaceSACMappings {
		if err := batch.Delete(getGraphKey([]byte(key))); err != nil {
			return err
		}
	}
	if err = batch.Flush(); err != nil {
		return err
	}
	return nil
}

func removeMappingsWithPrefix(prefix []byte, input map[string]SortedKeys) map[string]SortedKeys {
	ret := make(map[string]SortedKeys)
	for key, values := range input {
		ret[key] = removeValuesWithPrefix(prefix, values)
	}
	return ret
}

func removeValuesWithPrefix(prefix []byte, input SortedKeys) SortedKeys {
	ret := make([][]byte, 0, len(input))
	for _, value := range input {
		if !hasPrefix(prefix, value) {
			ret = append(ret, value)
		}
	}
	return ret
}
