package namespace

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	deploymentDataStore "github.com/stackrox/stackrox/central/deployment/datastore"
	"github.com/stackrox/stackrox/central/namespace/datastore"
	npDS "github.com/stackrox/stackrox/central/networkpolicies/datastore"
	secretDataStore "github.com/stackrox/stackrox/central/secret/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
)

// ResolveAll resolves all namespaces, populating volatile runtime data (like deployment and secret counts) by querying related stores.
func ResolveAll(ctx context.Context, dataStore datastore.DataStore, deploymentDataStore deploymentDataStore.DataStore,
	secretDataStore secretDataStore.DataStore, npStore npDS.DataStore, query *v1.Query) ([]*v1.Namespace, error) {
	metadataSlice, err := dataStore.SearchNamespaces(ctx, query)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving namespaces")
	}
	return populateFromMetadataSlice(ctx, metadataSlice, deploymentDataStore, secretDataStore, npStore)
}

// ResolveByQuery resolves all namespaces based on a query, populating volatile runtime data (like deployment and secret counts) by querying related stores.
func ResolveByQuery(ctx context.Context, q *v1.Query, dataStore datastore.DataStore, deploymentDataStore deploymentDataStore.DataStore,
	secretDataStore secretDataStore.DataStore, npStore npDS.DataStore) ([]*v1.Namespace, error) {
	metadataSlice, err := dataStore.SearchNamespaces(ctx, q)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving namespaces")
	}
	return populateFromMetadataSlice(ctx, metadataSlice, deploymentDataStore, secretDataStore, npStore)
}

// ResolveByClusterID resolves all namespaces for the given cluster.
func ResolveByClusterID(ctx context.Context, clusterID string, datastore datastore.DataStore, deploymentDataStore deploymentDataStore.DataStore,
	secretDataStore secretDataStore.DataStore, npStore npDS.DataStore, q *v1.Query) ([]*v1.Namespace, error) {
	metadataSlice, err := datastore.SearchNamespaces(ctx, q)
	if err != nil {
		return nil, errors.Wrapf(err, "searching namespace for cluster id %q", clusterID)
	}
	return populateFromMetadataSlice(ctx, metadataSlice, deploymentDataStore, secretDataStore, npStore)
}

func populateFromMetadataSlice(ctx context.Context, metadataSlice []*storage.NamespaceMetadata, deploymentDataStore deploymentDataStore.DataStore,
	secretDataStore secretDataStore.DataStore, npStore npDS.DataStore) ([]*v1.Namespace, error) {
	if len(metadataSlice) == 0 {
		return nil, nil
	}
	namespaces := make([]*v1.Namespace, 0, len(metadataSlice))
	for _, metadata := range metadataSlice {
		ns, err := populate(ctx, metadata, deploymentDataStore, secretDataStore, npStore)
		if err != nil {
			return nil, errors.Wrapf(err, "populating namespace '%s/%s'", metadata.GetClusterName(), metadata.GetName())
		}
		namespaces = append(namespaces, ns)
	}
	return namespaces, nil
}

// ResolveByClusterIDAndName resolves a namespace given its cluster ID and its name.
func ResolveByClusterIDAndName(ctx context.Context, clusterID string, name string, dataStore datastore.DataStore, deploymentDataStore deploymentDataStore.DataStore,
	secretDataStore secretDataStore.DataStore, npStore npDS.DataStore) (*v1.Namespace, bool, error) {
	q := search.NewQueryBuilder().AddExactMatches(search.Namespace, name).AddStrings(search.ClusterID, clusterID).ProtoQuery()
	namespaces, err := dataStore.SearchNamespaces(ctx, q)
	if err != nil {
		return nil, false, err
	}
	if len(namespaces) == 0 {
		return nil, false, nil
	}
	if len(namespaces) > 1 {
		return nil, false, fmt.Errorf("found multiple namespaces for cluster ID %q and name %q: %+v", clusterID, name, namespaces)
	}
	populated, err := populate(ctx, namespaces[0], deploymentDataStore, secretDataStore, npStore)
	return populated, true, err
}

// ResolveByID resolves a namespace by id given all the stores.
func ResolveByID(ctx context.Context, id string, dataStore datastore.DataStore, deploymentDataStore deploymentDataStore.DataStore,
	secretDataStore secretDataStore.DataStore, npStore npDS.DataStore) (*v1.Namespace, bool, error) {
	ns, exists, err := dataStore.GetNamespace(ctx, id)
	if err != nil {
		return nil, false, errors.Wrap(err, "retrieving from store")
	}
	if !exists {
		return nil, false, nil
	}
	populated, err := populate(ctx, ns, deploymentDataStore, secretDataStore, npStore)
	return populated, true, err
}

// populate takes the namespace and fills in data by querying related stores.
func populate(ctx context.Context, storageNamespace *storage.NamespaceMetadata, deploymentDataStore deploymentDataStore.DataStore,
	secretDataStore secretDataStore.DataStore, npStore npDS.DataStore) (*v1.Namespace, error) {
	q := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, storageNamespace.GetClusterId()).
		AddExactMatches(search.Namespace, storageNamespace.GetName()).
		ProtoQuery()
	deploymentResults, err := deploymentDataStore.Search(ctx, q.Clone())
	if err != nil {
		return nil, errors.Wrap(err, "searching deployments")
	}

	secretResults, err := secretDataStore.Search(ctx, q)
	if err != nil {
		return nil, errors.Wrap(err, "searching secrets")
	}

	networkPolicyCount, err := npStore.CountMatchingNetworkPolicies(
		ctx,
		storageNamespace.GetClusterId(),
		storageNamespace.GetName(),
	)
	if err != nil {
		return nil, errors.Wrap(err, "counting network policies")
	}

	return &v1.Namespace{
		Metadata:           storageNamespace,
		NumDeployments:     int32(len(deploymentResults)),
		NumSecrets:         int32(len(secretResults)),
		NumNetworkPolicies: int32(networkPolicyCount),
	}, nil
}
