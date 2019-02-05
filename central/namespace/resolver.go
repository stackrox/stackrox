package namespace

import (
	"fmt"

	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/namespace/datastore"
	networkPoliciesStore "github.com/stackrox/rox/central/networkpolicies/store"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/search"
)

// ResolveAll resolves all namespaces, populating volatile runtime data (like deployment and secret counts) by querying related stores.
func ResolveAll(dataStore datastore.DataStore, deploymentDataStore deploymentDataStore.DataStore,
	secretDataStore secretDataStore.DataStore, npStore networkPoliciesStore.Store) ([]*v1.Namespace, error) {
	metadataSlice, err := dataStore.GetNamespaces()
	if err != nil {
		return nil, fmt.Errorf("retrieving namespaces: %v", err)
	}
	return populateFromMetadataSlice(metadataSlice, deploymentDataStore, secretDataStore, npStore)
}

// ResolveByClusterID resolves all namespaces for the given cluster.
func ResolveByClusterID(clusterID string, datastore datastore.DataStore, deploymentDataStore deploymentDataStore.DataStore,
	secretDataStore secretDataStore.DataStore, npStore networkPoliciesStore.Store) ([]*v1.Namespace, error) {
	metadataSlice, err := datastore.SearchNamespaces(search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, clusterID).
		ProtoQuery())
	if err != nil {
		return nil, fmt.Errorf("searching namespace for cluster id %q: %v", clusterID, err)
	}
	return populateFromMetadataSlice(metadataSlice, deploymentDataStore, secretDataStore, npStore)
}

func populateFromMetadataSlice(metadataSlice []*storage.NamespaceMetadata, deploymentDataStore deploymentDataStore.DataStore,
	secretDataStore secretDataStore.DataStore, npStore networkPoliciesStore.Store) ([]*v1.Namespace, error) {
	if len(metadataSlice) == 0 {
		return nil, nil
	}
	namespaces := make([]*v1.Namespace, 0, len(metadataSlice))
	for _, metadata := range metadataSlice {
		ns, err := populate(metadata, deploymentDataStore, secretDataStore, npStore)
		if err != nil {
			return nil, fmt.Errorf("populating namespace '%s/%s': %v", metadata.GetClusterName(), metadata.GetName(), err)
		}
		namespaces = append(namespaces, ns)
	}
	return namespaces, nil
}

// ResolveByClusterIDAndName resolves a namespace given its cluster ID and its name.
func ResolveByClusterIDAndName(clusterID string, name string, dataStore datastore.DataStore, deploymentDataStore deploymentDataStore.DataStore,
	secretDataStore secretDataStore.DataStore, npStore networkPoliciesStore.Store) (*v1.Namespace, bool, error) {
	q := search.NewQueryBuilder().AddExactMatches(search.Namespace, name).AddStrings(search.ClusterID, clusterID).ProtoQuery()
	namespaces, err := dataStore.SearchNamespaces(q)
	if err != nil {
		return nil, false, err
	}
	if len(namespaces) == 0 {
		return nil, false, nil
	}
	if len(namespaces) > 1 {
		return nil, false, fmt.Errorf("found multiple namespaces for cluster ID %q and name %q: %+v", clusterID, name, namespaces)
	}
	populated, err := populate(namespaces[0], deploymentDataStore, secretDataStore, npStore)
	return populated, true, err
}

// ResolveByID resolves a namespace by id given all the stores.
func ResolveByID(id string, dataStore datastore.DataStore, deploymentDataStore deploymentDataStore.DataStore,
	secretDataStore secretDataStore.DataStore, npStore networkPoliciesStore.Store) (*v1.Namespace, bool, error) {
	ns, exists, err := dataStore.GetNamespace(id)
	if err != nil {
		return nil, false, fmt.Errorf("retrieving from store: %v", err)
	}
	if !exists {
		return nil, false, nil
	}
	populated, err := populate(ns, deploymentDataStore, secretDataStore, npStore)
	return populated, true, err
}

// populate takes the namespace and fills in data by querying related stores.
func populate(storageNamespace *storage.NamespaceMetadata, deploymentDataStore deploymentDataStore.DataStore,
	secretDataStore secretDataStore.DataStore, npStore networkPoliciesStore.Store) (*v1.Namespace, error) {
	q := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, storageNamespace.GetClusterId()).
		AddExactMatches(search.Namespace, storageNamespace.GetName()).
		ProtoQuery()
	deploymentResults, err := deploymentDataStore.Search(protoutils.CloneV1Query(q))
	if err != nil {
		return nil, fmt.Errorf("searching deployments: %v", err)
	}

	secretResults, err := secretDataStore.Search(q)
	if err != nil {
		return nil, fmt.Errorf("searching secrets: %v", err)
	}

	networkPolicyCount, err := npStore.CountMatchingNetworkPolicies(&v1.GetNetworkPoliciesRequest{
		ClusterId: storageNamespace.GetClusterId(),
		Namespace: storageNamespace.GetName(),
	})
	if err != nil {
		return nil, fmt.Errorf("counting network policies: %v", err)
	}

	return &v1.Namespace{
		Metadata:           storageNamespace,
		NumDeployments:     int32(len(deploymentResults)),
		NumSecrets:         int32(len(secretResults)),
		NumNetworkPolicies: int32(networkPolicyCount),
	}, nil
}
