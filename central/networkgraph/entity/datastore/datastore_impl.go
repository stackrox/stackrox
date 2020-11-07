package datastore

import (
	"context"

	"github.com/pkg/errors"
	graphConfigDS "github.com/stackrox/rox/central/networkgraph/config/datastore"
	"github.com/stackrox/rox/central/networkgraph/entity/datastore/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	// Before external sources were added to network graph, network graph APIs were read-only. It does not make sense
	// to have additional resource type to define permissions for addition and deletion of external sources as they
	// are modifications to network graph.
	// Since system-generated external sources are immutable (per current implementation) and are the same across all
	// clusters, we allow them to be accessed if users have network graph permissions to any cluster.
	networkGraphSAC = sac.ForResource(resources.NetworkGraph)
	log             = logging.LoggerForModule()
)

type dataStoreImpl struct {
	storage     store.EntityStore
	graphConfig graphConfigDS.DataStore
	clusterIDs  []string
	// Network trees in networkTreesByCluster map holds user-created networks.
	networkTreesByCluster map[string]tree.NetworkTree
	// defaultNetworkTree holds the system-generated networks.
	defaultNetworkTree tree.NetworkTree

	lock sync.Mutex
}

// NewEntityDataStore returns a new instance of EntityDataStore using the input storage underneath.
func NewEntityDataStore(storage store.EntityStore, graphConfig graphConfigDS.DataStore) EntityDataStore {
	ds := &dataStoreImpl{
		storage:     storage,
		graphConfig: graphConfig,
	}

	// DO NOT change the order
	if err := ds.initNetworkTrees(); err != nil {
		utils.Must(err)
	}
	return ds
}

func (ds *dataStoreImpl) initNetworkTrees() error {
	ds.networkTreesByCluster = make(map[string]tree.NetworkTree)
	ds.defaultNetworkTree = tree.NewDefaultNetworkTreeWrapper()

	// If network tree for a cluster is not found, it means it must orphan which shall be cleaned at next garbage collection.
	if err := ds.storage.Walk(func(obj *storage.NetworkEntity) error {
		return ds.getNetworkTree(obj.GetScope().GetClusterId(), true).Insert(obj.GetInfo())
	}); err != nil {
		return errors.Wrap(err, "initializing network tree")
	}
	return nil
}

func (ds *dataStoreImpl) RegisterCluster(clusterID string) {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	clusterSet := set.NewStringSet(ds.clusterIDs...)
	clusterSet.Add(clusterID)
	ds.clusterIDs = clusterSet.AsSlice()

	ds.getNetworkTree(clusterID, true)
}

func (ds *dataStoreImpl) GetEntity(ctx context.Context, id string) (*storage.NetworkEntity, bool, error) {
	if ok, err := ds.readAllowed(ctx, id); err != nil || !ok {
		return nil, false, err
	}
	return ds.storage.Get(id)
}

func (ds *dataStoreImpl) GetAllEntitiesForCluster(ctx context.Context, clusterID string) ([]*storage.NetworkEntity, error) {
	if clusterID == "" {
		return nil, errors.Wrap(errorhelpers.ErrInvalidArgs, "cannot get network entities. Cluster ID not specified")
	}

	if ok, err := networkGraphSAC.ReadAllowed(ctx, sac.ClusterScopeKey(clusterID)); err != nil || !ok {
		return nil, err
	}

	graphConfig, err := ds.graphConfig.GetNetworkGraphConfig(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain network graph configuration")
	}

	var entities []*storage.NetworkEntity
	if err := ds.storage.Walk(func(entity *storage.NetworkEntity) error {
		// Default network entities do not have scope filled because they are global, hence, ensure that they are not excluded if
		// "hide default networks" setting is off.
		if entity.GetScope().GetClusterId() != "" && entity.GetScope().GetClusterId() != clusterID {
			return nil
		}

		if excludeEntityForGraphConfig(graphConfig, entity) {
			return nil
		}
		entities = append(entities, entity)
		return nil
	}); err != nil {
		return nil, errors.Wrapf(err, "fetching network entities for cluster %s from storage", clusterID)
	}
	return entities, nil
}

func (ds *dataStoreImpl) GetNetworkTreeForClusterNoDefaults(ctx context.Context, clusterID string) (tree.ReadOnlyNetworkTree, error) {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	if ok, err := networkGraphSAC.ReadAllowed(ctx, sac.ClusterScopeKey(clusterID)); err != nil || !ok {
		return nil, err
	}

	return ds.networkTreesByCluster[clusterID], nil
}

func (ds *dataStoreImpl) GetAllEntities(ctx context.Context) ([]*storage.NetworkEntity, error) {
	ids, err := ds.storage.GetIDs()
	if err != nil {
		return nil, err
	}

	graphConfig, err := ds.graphConfig.GetNetworkGraphConfig(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain network graph configuration")
	}

	ret := make([]*storage.NetworkEntity, 0, len(ids))
	for _, id := range ids {
		if ok, err := ds.readAllowed(ctx, id); err != nil {
			return nil, err
		} else if !ok {
			continue
		}

		entity, found, err := ds.storage.Get(id)
		if err != nil {
			return nil, errors.Wrap(err, "fetching network entities from storage")
		}
		if !found {
			continue
		}

		if excludeEntityForGraphConfig(graphConfig, entity) {
			continue
		}
		ret = append(ret, entity)
	}
	return ret, nil
}

func (ds *dataStoreImpl) UpsertExternalNetworkEntity(ctx context.Context, entity *storage.NetworkEntity) error {
	if ok, err := ds.writeAllowed(ctx, entity.GetInfo().GetId()); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	if err := ds.validateExternalNetworkEntity(entity); err != nil {
		return err
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	if err := ds.storage.Upsert(entity); err != nil {
		return errors.Wrapf(err, "upserting network entity %s from storage", entity.GetInfo().GetId())
	}

	return ds.getNetworkTree(entity.GetScope().GetClusterId(), true).Insert(entity.GetInfo())
}

func (ds *dataStoreImpl) DeleteExternalNetworkEntity(ctx context.Context, id string) error {
	if ok, err := ds.writeAllowed(ctx, id); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	if err := ds.storage.Delete(id); err != nil {
		return errors.Wrapf(err, "deleting network entity %s from storage", id)
	}

	// Error is not expected since it has already been validated.
	decodedID, err := sac.ParseResourceID(id)
	utils.Should(err)

	if networkTree := ds.getNetworkTree(decodedID.ClusterID(), false); networkTree != nil {
		networkTree.Remove(id)
	}
	return nil
}

func (ds *dataStoreImpl) DeleteExternalNetworkEntitiesForCluster(ctx context.Context, clusterID string) error {
	if clusterID == "" {
		return errors.Wrap(errorhelpers.ErrInvalidArgs, "external network entities cannot be deleted. Cluster ID not specified")
	}

	if ok, err := networkGraphSAC.WriteAllowed(ctx, sac.ClusterScopeKey(clusterID)); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	var ids []string
	if err := ds.storage.Walk(func(obj *storage.NetworkEntity) error {
		if obj.GetInfo().GetExternalSource().GetDefault() {
			return nil
		}
		if clusterID == obj.GetScope().GetClusterId() {
			ids = append(ids, obj.GetInfo().GetId())
		}
		return nil
	}); err != nil {
		return err
	}

	if err := ds.storage.DeleteMany(ids); err != nil {
		return errors.Wrapf(err, "deleting network entities for cluster %s from storage", clusterID)
	}

	// If we are here, it means all the network entities for the `clusterID` are removed.
	ds.networkTreesByCluster[clusterID] = nil
	delete(ds.networkTreesByCluster, clusterID)

	for i, id := range ds.clusterIDs {
		if id == clusterID {
			ds.clusterIDs[i] = ds.clusterIDs[len(ds.clusterIDs)-1]
			ds.clusterIDs = ds.clusterIDs[:len(ds.clusterIDs)-1]
			break
		}
	}
	return nil
}

func (ds *dataStoreImpl) validateExternalNetworkEntity(entity *storage.NetworkEntity) error {
	if _, err := parseAndValidateID(entity.GetInfo().GetId()); err != nil {
		return err
	}

	if entity.GetInfo().GetType() != storage.NetworkEntityInfo_EXTERNAL_SOURCE {
		return errors.Wrap(errorhelpers.ErrInvalidArgs, "only external network graph sources can be created")
	}

	if entity.GetScope().GetClusterId() == "" {
		return errors.Wrap(errorhelpers.ErrInvalidArgs, "external network entities must be scoped to a cluster")
	}

	if entity.GetInfo().GetExternalSource() == nil {
		return errors.Wrap(errorhelpers.ErrInvalidArgs, "network entity must be specified")
	}

	if _, err := networkgraph.ValidateCIDR(entity.GetInfo().GetExternalSource().GetCidr()); err != nil {
		return errors.Wrap(errorhelpers.ErrInvalidArgs, err.Error())
	}

	if entity.GetInfo().GetExternalSource().GetName() == "" {
		entity.Info.GetExternalSource().Name = entity.GetInfo().GetExternalSource().GetCidr()
	}
	// CIDR Block uniqueness is handled by unique key CRUD. Refer to `UpsertExternalNetworkEntity(...)`.
	return nil
}

func (ds *dataStoreImpl) getNetworkTree(clusterID string, createIfNotFound bool) tree.NetworkTree {
	networkTree := ds.networkTreesByCluster[clusterID]
	if networkTree == nil && createIfNotFound {
		networkTree = tree.NewDefaultNetworkTreeWrapper()
		ds.networkTreesByCluster[clusterID] = networkTree
	}
	return networkTree
}

func (ds *dataStoreImpl) readAllowed(ctx context.Context, id string) (bool, error) {
	decodedID, err := parseAndValidateID(id)
	if err != nil {
		return false, err
	}

	// If cluster part of resource ID is empty, it means the resource must be default one.
	var scopeKeys [][]sac.ScopeKey
	if decodedID.ClusterID() == "" {
		scopeKeys = [][]sac.ScopeKey{sac.ClusterScopeKeys(ds.clusterIDs...)}
	} else {
		scopeKeys = [][]sac.ScopeKey{sac.ClusterScopeKeys(decodedID.ClusterID())}
	}
	return networkGraphSAC.ScopeChecker(ctx, storage.Access_READ_ACCESS).AnyAllowed(ctx, scopeKeys)
}

func (ds *dataStoreImpl) writeAllowed(ctx context.Context, id string) (bool, error) {
	decodedID, err := parseAndValidateID(id)
	if err != nil {
		return false, err
	}

	// If cluster part of resource ID is empty, it means the resource must be default one.
	var scopeKeys [][]sac.ScopeKey
	if decodedID.ClusterID() == "" {
		scopeKeys = [][]sac.ScopeKey{sac.ClusterScopeKeys(ds.clusterIDs...)}
	} else {
		scopeKeys = [][]sac.ScopeKey{sac.ClusterScopeKeys(decodedID.ClusterID())}
	}
	return networkGraphSAC.ScopeChecker(ctx, storage.Access_READ_WRITE_ACCESS).AnyAllowed(ctx, scopeKeys)
}

func parseAndValidateID(id string) (sac.ResourceID, error) {
	if id == "" {
		return sac.ResourceID{}, errors.Wrap(errorhelpers.ErrInvalidArgs, "network entity ID must be specified")
	}

	decodedID, err := sac.ParseResourceID(id)
	if err != nil {
		return sac.ResourceID{}, errors.Wrapf(errorhelpers.ErrInvalidArgs, "failed to parse network entity id %s", id)
	}

	if decodedID.ClusterID() == "" && decodedID.NamespaceID() != "" {
		return sac.ResourceID{}, errors.Wrapf(errorhelpers.ErrInvalidArgs, "invalid network entity id %s. Must be cluster-scoped or global-scoped", id)
	}
	return decodedID, nil
}

func excludeEntityForGraphConfig(graphConfig *storage.NetworkGraphConfig, entity *storage.NetworkEntity) bool {
	return graphConfig.HideDefaultExternalSrcs && entity.GetInfo().GetExternalSource().GetDefault()
}
