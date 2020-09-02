package entities

import (
	"context"
	"net"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/networkflow/datastore/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	// Before external sources were added to network graph, network graph APIs were read-only. It does not make sense
	// to have additional resource type to define permissions for addition and deletion of external sources as they
	// are modifications to network graph.
	networkGraphSAC = sac.ForResource(resources.NetworkGraph)
)

type entityDataStoreImpl struct {
	storage store.EntityStore
}

// NewEntityDataStore returns a new instance of EntityDataStore using the input storage underneath.
func NewEntityDataStore(storage store.EntityStore) EntityDataStore {
	return &entityDataStoreImpl{
		storage: storage,
	}
}

func (ds *entityDataStoreImpl) GetEntity(ctx context.Context, id string) (*storage.NetworkEntity, bool, error) {
	decodedID, err := sac.GetClusterScopedResourceID(id)
	if err != nil {
		return nil, false, err
	}

	if ok, err := networkGraphSAC.ReadAllowed(ctx, sac.ClusterScopeKey(decodedID.ClusterID)); err != nil || !ok {
		return nil, false, err
	}
	return ds.storage.GetEntity(id)
}

func (ds *entityDataStoreImpl) GetAllEntitiesForCluster(ctx context.Context, clusterID string) ([]*storage.NetworkEntity, error) {
	if clusterID == "" {
		return nil, errors.Wrap(errorhelpers.ErrInvalidArgs, "cannot get network entities. Cluster ID not specified")
	}

	if ok, err := networkGraphSAC.ReadAllowed(ctx, sac.ClusterScopeKey(clusterID)); err != nil || !ok {
		return nil, err
	}

	var entities []*storage.NetworkEntity
	if err := ds.storage.Walk(func(obj *storage.NetworkEntity) error {
		if clusterID == obj.GetScope().GetClusterId() {
			entities = append(entities, obj)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return entities, nil
}

func (ds *entityDataStoreImpl) GetAllEntities(ctx context.Context) ([]*storage.NetworkEntity, error) {
	ids, err := ds.storage.GetIDs()
	if err != nil {
		return nil, err
	}

	ret := make([]*storage.NetworkEntity, 0, len(ids))
	for _, id := range ids {
		decodedID, err := sac.GetClusterScopedResourceID(id)
		if err != nil {
			return nil, err
		}

		if ok, err := networkGraphSAC.ReadAllowed(ctx, sac.ClusterScopeKey(decodedID.ClusterID)); err != nil {
			return nil, err
		} else if !ok {
			continue
		}

		entity, found, err := ds.storage.GetEntity(id)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		ret = append(ret, entity)
	}
	return ret, nil
}

func (ds *entityDataStoreImpl) UpsertExternalNetworkEntity(ctx context.Context, entity *storage.NetworkEntity) error {
	if ok, err := networkGraphSAC.WriteAllowed(ctx, sac.ClusterScopeKey(entity.GetScope().GetClusterId())); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	if err := ds.validateExternalNetworkEntity(entity); err != nil {
		return err
	}

	if err := ds.storage.UpsertEntity(entity); err != nil {
		return err
	}
	return nil
}

func (ds *entityDataStoreImpl) DeleteExternalNetworkEntity(ctx context.Context, id string) error {
	if id == "" {
		return errors.Wrap(errorhelpers.ErrInvalidArgs, "external network entity cannot be deleted. ID not specified")
	}

	decodedID, err := sac.GetClusterScopedResourceID(id)
	if err != nil {
		return err
	}

	if ok, err := networkGraphSAC.WriteAllowed(ctx, sac.ClusterScopeKey(decodedID.ClusterID)); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}
	return ds.storage.DeleteEntity(id)
}

func (ds *entityDataStoreImpl) DeleteExternalNetworkEntitiesForCluster(ctx context.Context, clusterID string) error {
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
		if clusterID == obj.GetScope().GetClusterId() {
			ids = append(ids, obj.GetInfo().GetId())
		}
		return nil
	}); err != nil {
		return err
	}

	return ds.storage.DeleteEntities(ids)
}

func (ds *entityDataStoreImpl) validateExternalNetworkEntity(entity *storage.NetworkEntity) error {
	if entity.GetInfo().GetId() == "" {
		return errors.Wrap(errorhelpers.ErrInvalidArgs, "network entity ID must be specified")
	}

	if _, err := sac.GetClusterScopedResourceID(entity.GetInfo().GetId()); err != nil {
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

	if _, _, err := net.ParseCIDR(entity.GetInfo().GetExternalSource().GetCidr()); err != nil {
		return errors.Wrap(errorhelpers.ErrInvalidArgs, err.Error())
	}

	if entity.GetInfo().GetExternalSource().GetName() == "" {
		entity.Info.GetExternalSource().Name = entity.GetInfo().GetExternalSource().GetCidr()
	}
	// CIDR Block uniqueness is handled by unique key CRUD. Refer to `UpsertExternalNetworkEntity(...)`.
	return nil
}
