package globaldatastore

import (
	"context"

	"github.com/stackrox/rox/central/node/datastore"
	dackboxDatastore "github.com/stackrox/rox/central/node/datastore/dackbox/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
)

var (
	allNodeAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Node),
		))
)

type datastoreShim struct {
	clusterID string
	dacky     dackboxDatastore.DataStore
}

func newDatastoreShim(clusterID string, dacky dackboxDatastore.DataStore) datastore.DataStore {
	return &datastoreShim{
		clusterID: clusterID,
		dacky:     dacky,
	}
}

func (d *datastoreShim) clusterQuery() *v1.Query {
	return search.NewQueryBuilder().AddExactMatches(search.ClusterID, d.clusterID).ProtoQuery()
}

func (d *datastoreShim) ListNodes() ([]*storage.Node, error) {
	return d.dacky.SearchRawNodes(allNodeAccessCtx, d.clusterQuery())
}

func (d *datastoreShim) GetNode(id string) (*storage.Node, error) {
	node, exists, err := d.dacky.GetNode(allNodeAccessCtx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}
	return node, nil
}

func (d *datastoreShim) CountNodes() (int, error) {
	return d.dacky.Count(allNodeAccessCtx, d.clusterQuery())
}

func (d *datastoreShim) UpsertNode(node *storage.Node) error {
	return d.dacky.UpsertNode(allNodeAccessCtx, node)
}

func (d *datastoreShim) RemoveNode(id string) error {
	return d.dacky.DeleteNodes(allNodeAccessCtx, id)
}
