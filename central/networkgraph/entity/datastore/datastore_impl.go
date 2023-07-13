package datastore

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	graphConfigDS "github.com/stackrox/rox/central/networkgraph/config/datastore"
	"github.com/stackrox/rox/central/networkgraph/entity/datastore/internal/store"
	pgStore "github.com/stackrox/rox/central/networkgraph/entity/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/networkgraph/entity/networktree"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/networkgraph/externalsrcs"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
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
	networkGraphSAC       = sac.ForResource(resources.NetworkGraph)
	administrationReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Administration)))
)

type dataStoreImpl struct {
	storage       store.EntityStore
	graphConfig   graphConfigDS.DataStore
	sensorConnMgr connection.Manager
	treeMgr       networktree.Manager

	netEntityLock sync.Mutex
}

// NewEntityDataStore returns a new instance of EntityDataStore using the input storage underneath.
func NewEntityDataStore(storage store.EntityStore, graphConfig graphConfigDS.DataStore, treeMgr networktree.Manager, sensorConnMgr connection.Manager) EntityDataStore {
	ds := &dataStoreImpl{
		storage:       storage,
		graphConfig:   graphConfig,
		treeMgr:       treeMgr,
		sensorConnMgr: sensorConnMgr,
	}

	go ds.initNetworkTrees(sac.WithAllAccess(context.Background()))
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t *testing.T, pool postgres.DB) (EntityDataStore, error) {
	dbstore := pgStore.New(pool)
	graphConfigStore, err := graphConfigDS.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	treeMgr := networktree.Singleton()
	sensorCnxMgr := connection.ManagerSingleton()
	return NewEntityDataStore(dbstore, graphConfigStore, treeMgr, sensorCnxMgr), nil
}

// GetBenchPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetBenchPostgresDataStore(t testing.TB, pool postgres.DB) (EntityDataStore, error) {
	dbstore := pgStore.New(pool)
	graphConfigStore, err := graphConfigDS.GetBenchPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	treeMgr := networktree.Singleton()
	sensorCnxMgr := connection.ManagerSingleton()
	return NewEntityDataStore(dbstore, graphConfigStore, treeMgr, sensorCnxMgr), nil
}

func (ds *dataStoreImpl) initNetworkTrees(ctx context.Context) {
	// Create tree for default ones.
	entitiesByCluster := make(map[string][]*storage.NetworkEntityInfo)
	// If network tree for a cluster is not found, it means it must orphan which shall be cleaned at next garbage collection.
	walkFn := func() error {
		entitiesByCluster = make(map[string][]*storage.NetworkEntityInfo)
		return ds.storage.Walk(ctx, func(obj *storage.NetworkEntity) error {
			entitiesByCluster[obj.GetScope().GetClusterId()] = append(entitiesByCluster[obj.GetScope().GetClusterId()], obj.GetInfo())
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		log.Errorf("Failed to initialize network tree: %v", err)
	}

	if err := ds.treeMgr.Initialize(entitiesByCluster); err != nil {
		log.Errorf("Failed to initialize network trees for stored entities. "+
			"Some known external network entities may be excluded from network graph: %v", err)
	}
}

func (ds *dataStoreImpl) RegisterCluster(ctx context.Context, clusterID string) {
	ds.getNetworkTree(ctx, clusterID, true)

	go ds.doPushExternalNetworkEntitiesToSensor(clusterID)
}

func (ds *dataStoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	if ok, err := ds.readAllowed(ctx, id); err != nil || !ok {
		return false, err
	}
	return ds.storage.Exists(ctx, id)
}

func (ds *dataStoreImpl) GetIDs(ctx context.Context) ([]string, error) {
	ids, err := ds.storage.GetIDs(ctx)
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
	return ds.storage.Get(ctx, id)
}

func (ds *dataStoreImpl) GetAllEntitiesForCluster(ctx context.Context, clusterID string) ([]*storage.NetworkEntity, error) {
	if clusterID == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "cannot get network entities. Cluster ID not specified")
	}

	graphConfig, err := ds.graphConfig.GetNetworkGraphConfig(administrationReadCtx)
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
	graphConfig, err := ds.graphConfig.GetNetworkGraphConfig(administrationReadCtx)
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
	walkFn := func() error {
		entities = entities[:0]
		allowed = make(map[string]bool)
		return ds.storage.Walk(ctx, func(entity *storage.NetworkEntity) error {
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
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		return nil, errors.Wrap(err, "fetching network entities from storage")
	}
	return entities, nil
}

func (ds *dataStoreImpl) CreateExternalNetworkEntity(ctx context.Context, entity *storage.NetworkEntity, skipPush bool) error {
	if err := validateExternalNetworkEntity(entity); err != nil {
		return err
	}

	if ok, err := ds.writeAllowed(ctx, entity.GetInfo().GetId()); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if err := ds.create(ctx, entity); err != nil {
		return err
	}

	if !skipPush {
		go ds.doPushExternalNetworkEntitiesToSensor(entity.GetScope().GetClusterId())
	}

	return nil
}

func (ds *dataStoreImpl) CreateExtNetworkEntitiesForCluster(ctx context.Context, cluster string, entities ...*storage.NetworkEntity) ([]string, error) {
	var errs errorhelpers.ErrorList

	skipList := set.NewStringSet()
	for _, e := range entities {
		if err := validateExternalNetworkEntity(e); err != nil {
			errs.AddError(err)
			skipList.Add(e.GetInfo().GetId())
		}
	}

	allowed := make(map[string]bool)
	for _, e := range entities {
		if skipList.Contains(e.GetInfo().GetId()) {
			continue
		}

		clusterID := e.GetScope().GetClusterId()
		ok, found := allowed[clusterID]
		if !found {
			var err error
			ok, err = ds.writeAllowed(ctx, e.GetInfo().GetId())
			if err != nil {
				errs.AddError(err)
				skipList.Add(e.GetInfo().GetId())
				continue
			}
			allowed[clusterID] = ok
		}

		if !ok {
			errs.AddError(errors.Errorf("permission denied to create entity %s (CIDR=%s)",
				e.GetInfo().GetExternalSource().GetName(), e.GetInfo().GetExternalSource().GetCidr()))
			skipList.Add(e.GetInfo().GetId())
		}
	}

	inserted := make([]string, 0, len(entities)-len(skipList))
	for _, entity := range entities {
		if skipList.Contains(entity.GetInfo().GetId()) {
			continue
		}

		inserted = append(inserted, entity.GetInfo().GetId())
		if err := ds.create(ctx, entity); err != nil {
			errs.AddError(err)
		}
	}

	go ds.doPushExternalNetworkEntitiesToSensor(cluster)

	return inserted, errs.ToError()
}

func (ds *dataStoreImpl) create(ctx context.Context, entity *storage.NetworkEntity) error {
	network, err := externalsrcs.NetworkFromID(entity.GetInfo().GetId())
	if err != nil {
		return err
	}

	ds.netEntityLock.Lock()
	defer ds.netEntityLock.Unlock()

	if stored, found, err := ds.storage.Get(ctx, entity.GetInfo().GetId()); err != nil {
		return errors.Wrapf(err, "could not determine if network entity %s already exists in DB. SKIPPING",
			entity.GetInfo().GetExternalSource().GetName())
	} else if found {
		return errors.Wrapf(errox.AlreadyExists,
			"network %s of entity %s (CIDR=%s) conflicts with network of stored entity %s (CIDR=%s)",
			network,
			entity.GetInfo().GetExternalSource().GetName(), entity.GetInfo().GetExternalSource().GetCidr(),
			stored.GetInfo().GetExternalSource().GetName(), stored.GetInfo().GetExternalSource().GetCidr())
	}

	if err := ds.storage.Upsert(ctx, entity); err != nil {
		return errors.Wrapf(err, "upserting network entity %s into storage", entity.GetInfo().GetId())
	}

	return ds.getNetworkTree(ctx, entity.GetScope().GetClusterId(), true).Insert(entity.GetInfo())
}

func (ds *dataStoreImpl) UpdateExternalNetworkEntity(ctx context.Context, entity *storage.NetworkEntity, skipPush bool) error {
	if err := validateExternalNetworkEntity(entity); err != nil {
		return err
	}

	if ok, err := ds.writeAllowed(ctx, entity.GetInfo().GetId()); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	ds.netEntityLock.Lock()
	defer ds.netEntityLock.Unlock()

	_, err := ds.validateNoCIDRUpdate(ctx, entity)
	if err != nil {
		return err
	}

	if err := ds.storage.Upsert(ctx, entity); err != nil {
		return errors.Wrapf(err, "upserting network entity %s into storage", entity.GetInfo().GetId())
	}

	if !skipPush {
		go ds.doPushExternalNetworkEntitiesToSensor(entity.GetScope().GetClusterId())
	}

	t := ds.getNetworkTree(ctx, entity.GetScope().GetClusterId(), true)
	t.Remove(entity.GetInfo().GetId())
	return t.Insert(entity.GetInfo())
}

func (ds *dataStoreImpl) DeleteExternalNetworkEntity(ctx context.Context, id string) error {
	if ok, err := ds.writeAllowed(ctx, id); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	ds.netEntityLock.Lock()
	defer ds.netEntityLock.Unlock()

	// Check if the entity actually exists to avoid unnecessary push to Sensor.
	found, err := ds.storage.Exists(ctx, id)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	if err := ds.storage.Delete(ctx, id); err != nil {
		return errors.Wrapf(err, "deleting network entity %s from storage", id)
	}

	// Error is not expected since it has already been validated.
	decodedID, err := sac.ParseResourceID(id)
	if err != nil {
		return err
	}

	go ds.doPushExternalNetworkEntitiesToSensor(decodedID.ClusterID())

	if networkTree := ds.getNetworkTree(ctx, decodedID.ClusterID(), false); networkTree != nil {
		networkTree.Remove(id)
	}
	return nil
}

func (ds *dataStoreImpl) DeleteExternalNetworkEntitiesForCluster(ctx context.Context, clusterID string) error {
	if clusterID == "" {
		return errors.Wrap(errox.InvalidArgs, "external network entities cannot be deleted. Cluster ID not specified")
	}

	if ok, err := networkGraphSAC.WriteAllowed(ctx, sac.ClusterScopeKey(clusterID)); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	ds.netEntityLock.Lock()
	defer ds.netEntityLock.Unlock()

	var ids []string
	if err := ds.storage.Walk(ctx, func(obj *storage.NetworkEntity) error {
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

	if err := ds.storage.DeleteMany(ctx, ids); err != nil {
		return errors.Wrapf(err, "deleting network entities for cluster %s from storage", clusterID)
	}

	// If we are here, it means all the network entities for the `clusterID` are removed.
	ds.treeMgr.DeleteNetworkTree(ctx, clusterID)
	go ds.doPushExternalNetworkEntitiesToSensor(clusterID)

	return nil
}

func (ds *dataStoreImpl) validateNoCIDRUpdate(ctx context.Context, newEntity *storage.NetworkEntity) (bool, error) {
	old, found, err := ds.storage.Get(ctx, newEntity.GetInfo().GetId())
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

func (ds *dataStoreImpl) getNetworkTree(ctx context.Context, clusterID string, createIfNotFound bool) tree.NetworkTree {
	networkTree := ds.treeMgr.GetNetworkTree(ctx, clusterID)
	if networkTree == nil && createIfNotFound {
		networkTree = ds.treeMgr.CreateNetworkTree(ctx, clusterID)
	}
	return networkTree
}

func (ds *dataStoreImpl) doPushExternalNetworkEntitiesToSensor(clusters ...string) {
	// If push request if for a global network entity, push to all known clusters once and return.
	elevateCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
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

func (ds *dataStoreImpl) readAllowed(ctx context.Context, id string) (bool, error) {
	return allowed(ctx, storage.Access_READ_ACCESS, id)
}

func (ds *dataStoreImpl) writeAllowed(ctx context.Context, id string) (bool, error) {
	return allowed(ctx, storage.Access_READ_WRITE_ACCESS, id)
}

func allowed(ctx context.Context, access storage.Access, id string) (bool, error) {
	scopeKey, err := getScopeKey(id)
	if err != nil {
		return false, err
	}
	return networkGraphSAC.ScopeChecker(ctx, access).IsAllowed(scopeKey...), nil
}

func validateExternalNetworkEntity(entity *storage.NetworkEntity) error {
	if _, err := parseAndValidateID(entity.GetInfo().GetId()); err != nil {
		return err
	}

	if entity.GetInfo().GetType() != storage.NetworkEntityInfo_EXTERNAL_SOURCE {
		return errors.Wrap(errox.InvalidArgs, "only external network graph sources can be created")
	}

	if entity.GetInfo().GetExternalSource() == nil {
		return errors.Wrap(errox.InvalidArgs, "network entity must be specified")
	}

	if _, err := networkgraph.ValidateCIDR(entity.GetInfo().GetExternalSource().GetCidr()); err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}

	if entity.GetInfo().GetExternalSource().GetName() == "" {
		entity.Info.GetExternalSource().Name = entity.GetInfo().GetExternalSource().GetCidr()
	}
	// CIDR Block uniqueness is handled by unique key CRUD. Refer to `UpsertExternalNetworkEntity(...)`.
	return nil
}

func parseAndValidateID(id string) (sac.ResourceID, error) {
	if id == "" {
		return sac.ResourceID{}, errors.Wrap(errox.InvalidArgs, "network entity ID must be specified")
	}

	decodedID, err := sac.ParseResourceID(id)
	if err != nil {
		return sac.ResourceID{}, errors.Wrapf(errox.InvalidArgs, "failed to parse network entity id %s", id)
	}

	if !decodedID.IsValid() || decodedID.NamespaceScoped() {
		return sac.ResourceID{}, errors.Wrapf(errox.InvalidArgs, "invalid network entity id %s. Must be cluster-scoped or global-scoped", id)
	}
	return decodedID, nil
}

func excludeEntityForGraphConfig(graphConfig *storage.NetworkGraphConfig, entity *storage.NetworkEntity) bool {
	return graphConfig.GetHideDefaultExternalSrcs() && entity.GetInfo().GetExternalSource().GetDefault()
}

func getScopeKey(id string) ([]sac.ScopeKey, error) {
	decodedID, err := sac.ParseResourceID(string(id))
	if err != nil {
		return nil, err
	}

	if decodedID.ClusterScoped() {
		return []sac.ScopeKey{sac.ClusterScopeKey(decodedID.ClusterID())}, nil
	}

	return []sac.ScopeKey{}, nil // all clusters
}
