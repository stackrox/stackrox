package namespace

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/namespace/datastore"
	npDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
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

// ResolveMetadataOnlyByQuery resolves all namespaces based on a query. This will _not_ populate volatile runtime data and that must be requested separately.
func ResolveMetadataOnlyByQuery(ctx context.Context, q *v1.Query, dataStore datastore.DataStore) ([]*v1.Namespace, error) {
	metadataSlice, err := dataStore.SearchNamespaces(ctx, q)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving namespaces")
	}

	return populateNamespaceMetadataOnly(metadataSlice)
}

// Populates only the `Metadata` field of v1.Namespace. No volatile runtime data is filled. This should be used in queries where that information is unnecessary
// as fetching and calculating that can be very expensive.
func populateNamespaceMetadataOnly(metadataSlice []*storage.NamespaceMetadata) ([]*v1.Namespace, error) {
	if len(metadataSlice) == 0 {
		return nil, nil
	}
	namespaces := make([]*v1.Namespace, 0, len(metadataSlice))
	for _, metadata := range metadataSlice {
		// Don't populate the rest of the object (deploy, secrets and net pol count) as that's not performant when most of the time we just want namespace names
		namespaces = append(namespaces, &v1.Namespace{
			Metadata: metadata,
		})
	}
	return namespaces, nil
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
	q := search.NewQueryBuilder().AddExactMatches(search.Namespace, name).AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
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

// ResolveMetadataOnlyByID resolves namespace metadata only by id.
func ResolveMetadataOnlyByID(ctx context.Context, id string, dataStore datastore.DataStore) (*v1.Namespace, bool, error) {
	ns, exists, err := dataStore.GetNamespace(ctx, id)
	if err != nil {
		return nil, false, errors.Wrap(err, "retrieving namespace from store")
	}
	if !exists {
		return nil, false, nil
	}

	return &v1.Namespace{
		Metadata: ns,
	}, true, nil
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
