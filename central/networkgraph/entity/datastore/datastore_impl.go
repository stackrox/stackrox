package datastore

import (
	"context"

	"github.com/pkg/errors"
	graphConfigDS "github.com/stackrox/rox/central/networkgraph/config/datastore"
	"github.com/stackrox/rox/central/networkgraph/entity/datastore/internal/store"
	"github.com/stackrox/rox/central/networkgraph/entity/networktree"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/sensor/service/connection"
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
	log = logging.LoggerForModule()

	// Before external sources were added to network graph, network graph APIs were read-only. It does not make sense
	// to have additional resource type to define permissions for addition and deletion of external sources as they
	// are modifications to network graph.
	// Since system-generated external sources are immutable (per current implementation) and are the same across all
	// clusters, we allow them to be accessed if users have network graph permissions to any cluster.
	networkGraphSAC    = sac.ForResource(resources.NetworkGraph)
	graphConfigReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph)))
)

type dataStoreImpl struct {
	storage       store.EntityStore
	graphConfig   graphConfigDS.DataStore
	sensorConnMgr connection.Manager
	treeMgr       networktree.Manager
	clusterIDs    []string

	lock sync.Mutex
}

// NewEntityDataStore returns a new instance of EntityDataStore using the input storage underneath.
func NewEntityDataStore(storage store.EntityStore, graphConfig graphConfigDS.DataStore, treeMgr networktree.Manager, sensorConnMgr connection.Manager) EntityDataStore {
	ds := &dataStoreImpl{
		storage:       storage,
		graphConfig:   graphConfig,
		treeMgr:       treeMgr,
		sensorConnMgr: sensorConnMgr,
	}

	// DO NOT change the order
	if err := ds.initNetworkTrees(); err != nil {
		utils.Must(err)
	}
	return ds
}

func (ds *dataStoreImpl) initNetworkTrees() error {
	// Create tree for default ones.
	ds.treeMgr.CreateNetworkTree("")

	// If network tree for a cluster is not found, it means it must orphan which shall be cleaned at next garbage collection.
	if err := ds.storage.Walk(func(obj *storage.NetworkEntity) error {
		return ds.getNetworkTree(obj.GetScope().GetClusterId(), true).Insert(obj.GetInfo())
	}); err != nil {
		return errors.Wrap(err, "initializing network tree")
	}
	return nil
}

func (ds *dataStoreImpl) RegisterCluster(clusterID string) {
	ds.registerCluster(clusterID)
	ds.getNetworkTree(clusterID, true)

	go ds.doPushExternalNetworkEntitiesToSensor(clusterID)
}

func (ds *dataStoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	if ok, err := ds.readAllowed(ctx, id); err != nil || !ok {
		return false, err
	}
	return ds.storage.Exists(id)
}

func (ds *dataStoreImpl) GetIDs(ctx context.Context) ([]string, error) {
	ids, err := ds.storage.GetIDs()
	if err != nil {
		return nil, err
	}

	allowed := make(map[string]bool)
	ret := make([]string, 0, len(ids))
	for _, id := range ids {
		resID, err := sac.ParseResourceID(id)
		utils.Should(err)

		ok, found := allowed[resID.ClusterID()]
		if !found {
			var err error
			ok, err = ds.readAllowed(ctx, id)
			if err != nil {
				return nil, err
			}
			allowed[resID.ClusterID()] = ok
		}

		if !ok {
			continue
		}

		ret = append(ret, id)
	}
	return ret, nil
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

	graphConfig, err := ds.graphConfig.GetNetworkGraphConfig(graphConfigReadCtx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain network graph configuration")
	}

	return ds.GetAllMatchingEntities(ctx, func(entity *storage.NetworkEntity) bool {
		// Default network entities do not have scope filled because they are global, hence, ensure that they are not excluded if
		// "hide default networks" setting is off.
		if entity.GetScope().GetClusterId() != "" && entity.GetScope().GetClusterId() != clusterID {
			return false
		}

		return !excludeEntityForGraphConfig(graphConfig, entity)
	})
}

func (ds *dataStoreImpl) GetAllEntities(ctx context.Context) ([]*storage.NetworkEntity, error) {
	graphConfig, err := ds.graphConfig.GetNetworkGraphConfig(graphConfigReadCtx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to obtain network graph configuration")
	}

	return ds.GetAllMatchingEntities(ctx, func(entity *storage.NetworkEntity) bool {
		return !excludeEntityForGraphConfig(graphConfig, entity)
	})
}

func (ds *dataStoreImpl) GetAllMatchingEntities(ctx context.Context, pred func(entity *storage.NetworkEntity) bool) ([]*storage.NetworkEntity, error) {
	var entities []*storage.NetworkEntity
	allowed := make(map[string]bool)
	if err := ds.storage.Walk(func(entity *storage.NetworkEntity) error {
		if !pred(entity) {
			return nil
		}

		clusterID := entity.GetScope().GetClusterId()
		ok, found := allowed[clusterID]
		if !found {
			var err error
			ok, err = ds.readAllowed(ctx, entity.GetInfo().GetId())
			if err != nil {
				return err
			}
			allowed[clusterID] = ok
		}

		if !ok {
			return nil
		}

		entities = append(entities, entity)
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "fetching network entities from storage")
	}
	return entities, nil
}

func (ds *dataStoreImpl) CreateExternalNetworkEntity(ctx context.Context, entity *storage.NetworkEntity, skipPush bool) error {
	if err := ds.validateExternalNetworkEntity(entity); err != nil {
		return err
	}

	if ok, err := ds.writeAllowed(ctx, entity.GetInfo().GetId()); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	if found, err := ds.storage.Exists(entity.GetInfo().GetId()); err != nil {
		return err
	} else if found {
		return errors.Wrapf(errorhelpers.ErrAlreadyExists, "network entity %s (CIDR=%s) already exists",
			entity.GetInfo().GetId(), entity.GetInfo().GetExternalSource().GetCidr())
	}

	if err := ds.storage.Upsert(entity); err != nil {
		return errors.Wrapf(err, "upserting network entity %s into storage", entity.GetInfo().GetId())
	}

	if !skipPush {
		go ds.doPushExternalNetworkEntitiesToSensor(entity.GetScope().GetClusterId())
	}

	return ds.getNetworkTree(entity.GetScope().GetClusterId(), true).Insert(entity.GetInfo())
}

func (ds *dataStoreImpl) UpdateExternalNetworkEntity(ctx context.Context, entity *storage.NetworkEntity, skipPush bool) error {
	if err := ds.validateExternalNetworkEntity(entity); err != nil {
		return err
	}

	if ok, err := ds.writeAllowed(ctx, entity.GetInfo().GetId()); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	_, err := ds.validateNoCIDRUpdate(entity)
	if err != nil {
		return err
	}

	if err := ds.storage.Upsert(entity); err != nil {
		return errors.Wrapf(err, "upserting network entity %s into storage", entity.GetInfo().GetId())
	}

	if !skipPush {
		go ds.doPushExternalNetworkEntitiesToSensor(entity.GetScope().GetClusterId())
	}

	ds.getNetworkTree(entity.GetScope().GetClusterId(), true).Remove(entity.GetInfo().GetId())
	return ds.getNetworkTree(entity.GetScope().GetClusterId(), true).Insert(entity.GetInfo())
}

func (ds *dataStoreImpl) DeleteExternalNetworkEntity(ctx context.Context, id string) error {
	if ok, err := ds.writeAllowed(ctx, id); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	// Check if the entity actually exists to avoid unnecessary push to Sensor.
	_, found, err := ds.storage.Get(id)
	if err != nil {
		return err
	}

	if !found {
		return nil
	}

	if err := ds.storage.Delete(id); err != nil {
		return errors.Wrapf(err, "deleting network entity %s from storage", id)
	}

	// Error is not expected since it has already been validated.
	decodedID, err := sac.ParseResourceID(id)
	utils.Should(err)

	go ds.doPushExternalNetworkEntitiesToSensor(decodedID.ClusterID())

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

	var ids []string
	if err := ds.storage.Walk(func(obj *storage.NetworkEntity) error {
		// Skip default ones.
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
	ds.treeMgr.DeleteNetworkTree(clusterID)
	ds.unregisterCluster(clusterID)
	go ds.doPushExternalNetworkEntitiesToSensor(clusterID)

	return nil
}

func (ds *dataStoreImpl) validateExternalNetworkEntity(entity *storage.NetworkEntity) error {
	if _, err := parseAndValidateID(entity.GetInfo().GetId()); err != nil {
		return err
	}

	if entity.GetInfo().GetType() != storage.NetworkEntityInfo_EXTERNAL_SOURCE {
		return errors.Wrap(errorhelpers.ErrInvalidArgs, "only external network graph sources can be created")
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

func (ds *dataStoreImpl) validateNoCIDRUpdate(newEntity *storage.NetworkEntity) (bool, error) {
	old, found, err := ds.storage.Get(newEntity.GetInfo().GetId())
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}
	if old.GetInfo().GetExternalSource().GetCidr() != newEntity.GetInfo().GetExternalSource().GetCidr() {
		return true, errors.Errorf("updating CIDR is not allowed. Please delete %s (name=%s) and recreate the network entity",
			newEntity.GetInfo().GetId(), newEntity.GetInfo().GetExternalSource().GetName())
	}
	return true, nil
}

func (ds *dataStoreImpl) getNetworkTree(clusterID string, createIfNotFound bool) tree.NetworkTree {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	networkTree := ds.treeMgr.GetNetworkTree(clusterID)
	if networkTree == nil && createIfNotFound {
		networkTree = ds.treeMgr.CreateNetworkTree(clusterID)
	}
	return networkTree
}

func (ds *dataStoreImpl) doPushExternalNetworkEntitiesToSensor(clusters ...string) {
	// If push request if for a global network entity, push to all known clusters once and return.
	elevateCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.NetworkGraph)))

	if set.NewStringSet(clusters...).Contains("") {
		if err := ds.sensorConnMgr.PushExternalNetworkEntitiesToAllSensors(elevateCtx); err != nil {
			log.Errorf("failed to sync external networks with some clusters: %v", err)
		}
		return
	}

	for _, cluster := range clusters {
		if err := ds.sensorConnMgr.PushExternalNetworkEntitiesToSensor(elevateCtx, cluster); err != nil {
			log.Errorf("failed to sync external networks with cluster %s: %v", cluster, err)
		}
	}
}

func (ds *dataStoreImpl) getRegisteredClusters() []string {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	return ds.clusterIDs
}

func (ds *dataStoreImpl) registerCluster(cluster string) {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	clusterSet := set.NewStringSet(ds.clusterIDs...)
	clusterSet.Add(cluster)
	ds.clusterIDs = clusterSet.AsSlice()
}

func (ds *dataStoreImpl) unregisterCluster(cluster string) {
	ds.lock.Lock()
	defer ds.lock.Unlock()

	for i, id := range ds.clusterIDs {
		if id == cluster {
			ds.clusterIDs[i] = ds.clusterIDs[len(ds.clusterIDs)-1]
			ds.clusterIDs = ds.clusterIDs[:len(ds.clusterIDs)-1]
			break
		}
	}
}

func (ds *dataStoreImpl) readAllowed(ctx context.Context, id string) (bool, error) {
	return ds.allowed(ctx, storage.Access_READ_ACCESS, id)
}

func (ds *dataStoreImpl) writeAllowed(ctx context.Context, id string) (bool, error) {
	return ds.allowed(ctx, storage.Access_READ_WRITE_ACCESS, id)
}

func (ds *dataStoreImpl) allowed(ctx context.Context, access storage.Access, id string) (bool, error) {
	scopeKeys, err := getScopeKeys(id, ds.getRegisteredClusters())
	if err != nil {
		return false, err
	}
	if len(scopeKeys) == 0 {
		return networkGraphSAC.ScopeChecker(ctx, access).Allowed(ctx)
	}
	return networkGraphSAC.ScopeChecker(ctx, access).AnyAllowed(ctx, scopeKeys)
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
	return graphConfig.GetHideDefaultExternalSrcs() && entity.GetInfo().GetExternalSource().GetDefault()
}

func getScopeKeys(id string, clusters []string) ([][]sac.ScopeKey, error) {
	decodedID, err := sac.ParseResourceID(string(id))
	if err != nil {
		return nil, err
	}

	// If cluster part of resource ID is empty, it means the resource must be default one.
	if decodedID.ClusterID() != "" {
		return [][]sac.ScopeKey{sac.ClusterScopeKeys(decodedID.ClusterID())}, nil
	}

	scopeKeys := make([][]sac.ScopeKey, 0, len(clusters))
	for _, cluster := range clusters {
		scopeKeys = append(scopeKeys, sac.ClusterScopeKeys(cluster))
	}

	return scopeKeys, nil
}
