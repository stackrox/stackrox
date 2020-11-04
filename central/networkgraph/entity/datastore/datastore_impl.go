package datastore

import (
	"context"

	"github.com/pkg/errors"
	graphConfigDS "github.com/stackrox/rox/central/networkgraph/config/datastore"
	"github.com/stackrox/rox/central/networkgraph/entity/datastore/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/networkgraph/tree"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	// Before external sources were added to network graph, network graph APIs were read-only. It does not make sense
	// to have additional resource type to define permissions for addition and deletion of external sources as they
	// are modifications to network graph.
	networkGraphSAC = sac.ForResource(resources.NetworkGraph)
)

type dataStoreImpl struct {
	storage               store.EntityStore
	graphConfig           graphConfigDS.DataStore
	networkTreesByCluster map[string]*tree.NetworkTreeWrapper

	lock sync.Mutex
}

// NewEntityDataStore returns a new instance of EntityDataStore using the input storage underneath.
func NewEntityDataStore(storage store.EntityStore, graphConfig graphConfigDS.DataStore) EntityDataStore {
	ds := &dataStoreImpl{
		storage:               storage,
		graphConfig:           graphConfig,
		networkTreesByCluster: make(map[string]*tree.NetworkTreeWrapper),
	}

	if err := ds.initNetworkTree(); err != nil {
		utils.Should(err)
	}
	return ds
}

func (ds *dataStoreImpl) initNetworkTree() error {
	err := ds.storage.Walk(func(obj *storage.NetworkEntity) error {
		return ds.getNetworkTree(obj.GetScope().GetClusterId(), true).Insert(obj.GetInfo())
	})
	if err != nil {
		return errors.Wrap(err, "initializing network tree")
	}
	return nil
}

func (ds *dataStoreImpl) GetEntity(ctx context.Context, id string) (*storage.NetworkEntity, bool, error) {
	decodedID, err := sac.ParseResourceID(id)
	if err != nil {
		return nil, false, err
	}

	if ok, err := networkGraphSAC.ReadAllowed(ctx, sac.ClusterScopeKey(decodedID.ClusterID())); err != nil || !ok {
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
		if clusterID != entity.GetScope().GetClusterId() {
			return nil
		}
		if graphConfig.HideDefaultExternalSrcs && entity.GetInfo().GetExternalSource().GetDefault() {
			return nil
		}
		entities = append(entities, entity)
		return nil
	}); err != nil {
		return nil, errors.Wrapf(err, "fetching network entities for cluster %s from storage", clusterID)
	}
	return entities, nil
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
		decodedID, err := sac.ParseResourceID(id)
		if err != nil {
			return nil, err
		}

		if ok, err := networkGraphSAC.ReadAllowed(ctx, sac.ClusterScopeKey(decodedID.ClusterID())); err != nil {
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

		if graphConfig.HideDefaultExternalSrcs && entity.GetInfo().GetExternalSource().GetDefault() {
			continue
		}
		ret = append(ret, entity)
	}
	return ret, nil
}

func (ds *dataStoreImpl) UpsertExternalNetworkEntity(ctx context.Context, entity *storage.NetworkEntity) error {
	if ok, err := networkGraphSAC.WriteAllowed(ctx, sac.ClusterScopeKey(entity.GetScope().GetClusterId())); err != nil {
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
	if id == "" {
		return errors.Wrap(errorhelpers.ErrInvalidArgs, "external network entity cannot be deleted. ID not specified")
	}

	decodedID, err := sac.ParseResourceID(id)
	if err != nil {
		return errors.Wrap(errorhelpers.ErrInvalidArgs, err.Error())
	}

	if ok, err := networkGraphSAC.WriteAllowed(ctx, sac.ClusterScopeKey(decodedID.ClusterID())); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	if err := ds.storage.Delete(id); err != nil {
		return errors.Wrapf(err, "deleting network entity %s from storage", id)
	}

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
	return nil
}

func (ds *dataStoreImpl) validateExternalNetworkEntity(entity *storage.NetworkEntity) error {
	if entity.GetInfo().GetId() == "" {
		return errors.Wrap(errorhelpers.ErrInvalidArgs, "network entity ID must be specified")
	}

	if _, err := sac.ParseResourceID(entity.GetInfo().GetId()); err != nil {
		return errors.Wrap(errorhelpers.ErrInvalidArgs, err.Error())
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

func (ds *dataStoreImpl) getNetworkTree(clusterID string, createIfNotFound bool) *tree.NetworkTreeWrapper {
	networkTree := ds.networkTreesByCluster[clusterID]
	if networkTree == nil && createIfNotFound {
		networkTree = tree.NewDefaultNetworkTreeWrapper()
		ds.networkTreesByCluster[clusterID] = networkTree
	}
	return networkTree
}
