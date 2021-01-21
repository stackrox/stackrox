package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/node/index"
	"github.com/stackrox/rox/central/node/store"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/nodes/enricher"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/scancomponent"
)

var (
	errReadOnly = errors.New("data store does not allow write access")

	deleteRiskCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_WRITE_ACCESS), sac.ResourceScopeKeys(resources.Risk)))
)

//go:generate mockgen-wrapper

// DataStore is a wrapper around a store that provides search functionality
type DataStore interface {
	store.Store
}

// New returns a new datastore
func New(store store.Store, indexer index.Indexer, writeAccess bool, risks riskDS.DataStore,
	nodeRanker *ranking.Ranker, nodeComponentRanker *ranking.Ranker) DataStore {
	return &datastoreImpl{
		store:       store,
		indexer:     indexer,
		keyedMutex:  concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
		writeAccess: writeAccess,

		risks:               risks,
		nodeRanker:          nodeRanker,
		nodeComponentRanker: nodeComponentRanker,
	}
}

type datastoreImpl struct {
	indexer     index.Indexer
	store       store.Store
	keyedMutex  *concurrency.KeyedMutex
	writeAccess bool

	risks               riskDS.DataStore
	nodeRanker          *ranking.Ranker
	nodeComponentRanker *ranking.Ranker
}

// ListNodes returns all nodes in the store
func (d *datastoreImpl) ListNodes() ([]*storage.Node, error) {
	nodes, err := d.store.ListNodes()
	if err != nil {
		return nil, err
	}

	if features.HostScanning.Enabled() {
		d.updateNodePriority(nodes...)
	}

	return nodes, nil
}

// GetNode returns an individual node
func (d *datastoreImpl) GetNode(id string) (*storage.Node, error) {
	node, err := d.store.GetNode(id)
	if err != nil {
		return nil, err
	}

	if node == nil {
		return nil, nil
	}

	if features.HostScanning.Enabled() {
		d.updateNodePriority(node)
	}

	return node, nil
}

// CountNodes returns the number of nodes
func (d *datastoreImpl) CountNodes() (int, error) {
	return d.store.CountNodes()
}

// UpsertNode adds a node to the store and the indexer
func (d *datastoreImpl) UpsertNode(node *storage.Node) error {
	if !d.writeAccess {
		return errReadOnly
	}

	d.keyedMutex.Lock(node.GetId())
	defer d.keyedMutex.Unlock(node.GetId())

	if features.HostScanning.Enabled() {
		d.updateComponentRisk(node)
		enricher.FillScanStats(node)
	}

	if err := d.store.UpsertNode(node); err != nil {
		return err
	}
	if err := d.indexer.AddNode(node); err != nil {
		return err
	}

	if features.HostScanning.Enabled() {
		// If the node in db is latest, this node object will be carrying its risk score
		d.nodeRanker.Add(node.GetId(), node.GetRiskScore())
	}

	return nil
}

// RemoveNode deletes a node from the store and the indexer
func (d *datastoreImpl) RemoveNode(id string) error {
	if !d.writeAccess {
		return errReadOnly
	}

	d.keyedMutex.Lock(id)
	defer d.keyedMutex.Unlock(id)
	if err := d.store.RemoveNode(id); err != nil {
		return err
	}
	if err := d.indexer.DeleteNode(id); err != nil {
		return err
	}

	if features.HostScanning.Enabled() {
		// removing component risk will be handled by pruning
		return d.risks.RemoveRisk(deleteRiskCtx, id, storage.RiskSubjectType_NODE)
	}

	return nil
}

func (d *datastoreImpl) updateComponentRisk(node *storage.Node) {
	for _, component := range node.GetScan().GetComponents() {
		component.RiskScore = d.nodeComponentRanker.GetScoreForID(scancomponent.ComponentID(component.GetName(), component.GetVersion()))
	}
}

func (d *datastoreImpl) updateNodePriority(nodes ...*storage.Node) {
	for _, node := range nodes {
		node.Priority = d.nodeRanker.GetRankForID(node.GetId())
	}
}
