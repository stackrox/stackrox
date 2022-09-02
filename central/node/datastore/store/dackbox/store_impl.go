package dackbox

import (
	"context"
	"time"

	protoTypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	clusterDackBox "github.com/stackrox/rox/central/cluster/dackbox"
	componentCVEEdgeDackBox "github.com/stackrox/rox/central/componentcveedge/dackbox"
	cveDackBox "github.com/stackrox/rox/central/cve/dackbox"
	cveUtil "github.com/stackrox/rox/central/cve/utils"
	componentDackBox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	"github.com/stackrox/rox/central/metrics"
	nodeDackBox "github.com/stackrox/rox/central/node/dackbox"
	"github.com/stackrox/rox/central/node/datastore/store"
	"github.com/stackrox/rox/central/node/datastore/store/common"
	nodeComponentEdgeDackBox "github.com/stackrox/rox/central/nodecomponentedge/dackbox"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/dackbox/sortedkeys"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	typ          = "Node"
	metadataType = "NodeMetadata"
)

type storeImpl struct {
	dacky              *dackbox.DackBox
	keyFence           concurrency.KeyFence
	noUpdateTimestamps bool
}

// New returns a new Store instance using the provided DackBox instance.
func New(dacky *dackbox.DackBox, keyFence concurrency.KeyFence, noUpdateTimestamps bool) store.Store {
	return &storeImpl{
		dacky:              dacky,
		keyFence:           keyFence,
		noUpdateTimestamps: noUpdateTimestamps,
	}
}

// Exists returns if a node exists in the DB with the given id.
func (b *storeImpl) Exists(_ context.Context, id string) (bool, error) {
	branch, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return false, err
	}
	defer branch.Discard()

	exists, err := nodeDackBox.Reader.ExistsIn(nodeDackBox.BucketHandler.GetKey(id), branch)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// Count returns the number of nodes currently stored in the DB.
func (b *storeImpl) Count(_ context.Context) (int, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Count, typ)

	branch, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return 0, err
	}
	defer branch.Discard()

	count, err := nodeDackBox.Reader.CountIn(nodeDackBox.Bucket, branch)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// Get returns the node with given id.
func (b *storeImpl) Get(_ context.Context, id string) (*storage.Node, bool, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Get, typ)

	branch, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, false, err
	}
	defer branch.Discard()

	node, err := b.readNode(branch, id)
	if err != nil {
		return nil, false, err
	}
	return node, node != nil, err
}

// GetNodeMetadata returns the node with the given id without component/CVE data.
func (b *storeImpl) GetNodeMetadata(_ context.Context, id string) (*storage.Node, bool, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Get, metadataType)

	branch, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, false, err
	}
	defer branch.Discard()

	node, err := b.readNodeMetadata(branch, id)
	if err != nil {
		return nil, false, err
	}
	return node, node != nil, err
}

func (b *storeImpl) GetManyNodeMetadata(ctx context.Context, id []string) ([]*storage.Node, []int, error) {
	utils.Must(errors.New("Unexpected call to GetManyNodeMetadata in Dackbox when running on Postgres"))
	return nil, nil, nil
}

// GetMany returns nodes with given ids.
func (b *storeImpl) GetMany(_ context.Context, ids []string) ([]*storage.Node, []int, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.GetMany, typ)

	branch, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, nil, err
	}
	defer branch.Discard()

	nodes := make([]*storage.Node, 0, len(ids))
	var missingIndices []int
	for idx, id := range ids {
		node, err := b.readNode(branch, id)
		if err != nil {
			return nil, nil, err
		}
		if node != nil {
			nodes = append(nodes, node)
		} else {
			missingIndices = append(missingIndices, idx)
		}
	}
	return nodes, missingIndices, nil
}

// Upsert writes a node to the DB, overwriting previous data.
func (b *storeImpl) Upsert(_ context.Context, node *storage.Node) error {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Upsert, typ)

	iTime := protoTypes.TimestampNow()
	if !b.noUpdateTimestamps {
		node.LastUpdated = iTime
	}

	node, nodeUpdated, scanUpdated, err := b.toUpsert(node)
	if err != nil {
		return err
	}
	if !nodeUpdated && !scanUpdated {
		return nil
	}

	// If the node scan is not updated, skip updating that part in DB, i.e. rewriting components and cves.
	parts := common.Split(node, scanUpdated)

	clusterKey := clusterDackBox.BucketHandler.GetKey(node.GetClusterId())
	keysToUpdate := append(gatherKeysForNodeParts(parts), clusterKey)
	return b.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet(keysToUpdate...), func() error {
		return b.writeNodeParts(parts, clusterKey, iTime, scanUpdated)
	})
}

// toUpsert returns the node to upsert to the store based on the given node.
// The first bool return is true if the node data from Kubernetes is updated from what is currently stored.
// The second bool return is true if the node's scan data is updated from what is currently stored.
func (b *storeImpl) toUpsert(node *storage.Node) (*storage.Node, bool, bool, error) {
	txn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, false, false, err
	}
	defer txn.Discard()

	msg, err := nodeDackBox.Reader.ReadIn(nodeDackBox.BucketHandler.GetKey(node.GetId()), txn)
	if err != nil {
		return nil, false, false, err
	}
	// No node for given ID found, hence mark new node as latest
	if msg == nil {
		return node, true, true, nil
	}

	oldNode := msg.(*storage.Node)

	nodeToUpsert := node.Clone()

	scanUpdated := true
	// We skip rewriting components and CVEs if scan is not newer, hence we do not need to merge.
	if oldNode.GetScan().GetScanTime().Compare(nodeToUpsert.GetScan().GetScanTime()) > 0 {
		scanUpdated = false

		fullOldNode, err := b.readNode(txn, nodeToUpsert.GetId())
		if err != nil {
			return nil, false, false, err
		}
		nodeToUpsert.Scan = fullOldNode.GetScan()

		// The node scan in the DB is latest, then use its risk score and scan stats.
		nodeToUpsert.RiskScore = oldNode.GetRiskScore()
		nodeToUpsert.SetComponents = oldNode.GetSetComponents()
		nodeToUpsert.SetCves = oldNode.GetSetCves()
		nodeToUpsert.SetFixable = oldNode.GetSetFixable()
		nodeToUpsert.SetTopCvss = oldNode.GetSetTopCvss()
	}

	nodeUpdated := true
	// We skip rewriting the node (excluding the components and CVEs) if the node is not newer.
	if oldNode.GetK8SUpdated().Compare(nodeToUpsert.GetK8SUpdated()) > 0 {
		nodeUpdated = false

		lastUpdated := nodeToUpsert.GetLastUpdated()
		scan := nodeToUpsert.GetScan()
		riskScore := nodeToUpsert.GetRiskScore()
		setComponents := nodeToUpsert.GetSetComponents()
		setCVEs := nodeToUpsert.GetSetCves()
		setFixable := nodeToUpsert.GetSetFixable()
		setTopCVSS := nodeToUpsert.GetSetTopCvss()

		nodeToUpsert = oldNode.Clone()
		nodeToUpsert.LastUpdated = lastUpdated
		nodeToUpsert.Scan = scan
		nodeToUpsert.RiskScore = riskScore
		nodeToUpsert.SetComponents = setComponents
		nodeToUpsert.SetCves = setCVEs
		nodeToUpsert.SetFixable = setFixable
		nodeToUpsert.SetTopCvss = setTopCVSS
	}

	return nodeToUpsert, nodeUpdated, scanUpdated, nil
}

// Delete deletes a node and all its data.
func (b *storeImpl) Delete(_ context.Context, id string) error {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Remove, typ)

	keyTxn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return err
	}
	defer keyTxn.Discard()
	keys, err := gatherKeysForNode(keyTxn, id)
	if err != nil {
		return err
	}

	// Lock the set of keys we want to update.
	return b.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet(keys.allKeys...), func() error {
		return b.deleteNodeKeys(keys)
	})
}

func (b *storeImpl) GetTxnCount() (txNum uint64, err error) {
	return 0, nil
}

func (b *storeImpl) IncTxnCount() error {
	return nil
}

// Writing a node to the DB and graph.
//////////////////////////////////////

func gatherKeysForNodeParts(parts *common.NodeParts) [][]byte {
	var allKeys [][]byte
	allKeys = append(allKeys, nodeDackBox.BucketHandler.GetKey(parts.Node.GetId()))
	for _, componentParts := range parts.Children {
		allKeys = append(allKeys, componentDackBox.BucketHandler.GetKey(componentParts.Component.GetId()))
		for _, cveParts := range componentParts.Children {
			allKeys = append(allKeys, cveDackBox.BucketHandler.GetKey(cveParts.CVE.GetId()))
		}
	}
	return allKeys
}

func (b *storeImpl) writeNodeParts(parts *common.NodeParts, clusterKey []byte, iTime *protoTypes.Timestamp, scanUpdated bool) error {
	dackTxn, err := b.dacky.NewTransaction()
	if err != nil {
		return err
	}
	defer dackTxn.Discard()

	var componentKeys [][]byte
	// Update the node components and cves iff the node upsert has updated scan.
	// Note: In such cases, the loops in following block will not be entered anyway since len(parts.Children) is 0.
	// This is more for good readability amidst the complex code.
	if scanUpdated {
		for _, componentData := range parts.Children {
			componentKey, err := b.writeComponentParts(dackTxn, componentData, iTime)
			if err != nil {
				return err
			}
			componentKeys = append(componentKeys, componentKey)
		}
	}

	if err := nodeDackBox.Upserter.UpsertIn(nil, parts.Node, dackTxn); err != nil {
		return err
	}

	nodeKey := nodeDackBox.KeyFunc(parts.Node)

	dackTxn.Graph().AddRefs(clusterKey, nodeKey)

	// Update the downstream node links in the graph iff the node upsert has updated scan.
	if scanUpdated {
		dackTxn.Graph().SetRefs(nodeKey, componentKeys)
	}

	return dackTxn.Commit()
}

func (b *storeImpl) writeComponentParts(txn *dackbox.Transaction, parts *common.ComponentParts, iTime *protoTypes.Timestamp) ([]byte, error) {
	var cveKeys [][]byte
	for _, cveData := range parts.Children {
		cveKey, err := b.writeCVEParts(txn, cveData, iTime)
		if err != nil {
			return nil, err
		}
		cveKeys = append(cveKeys, cveKey)
	}

	componentKey := componentDackBox.KeyFunc(parts.Component)
	if err := nodeComponentEdgeDackBox.Upserter.UpsertIn(nil, parts.Edge, txn); err != nil {
		return nil, err
	}
	if err := componentDackBox.Upserter.UpsertIn(nil, parts.Component, txn); err != nil {
		return nil, err
	}

	txn.Graph().SetRefs(componentKey, cveKeys)
	return componentKey, nil
}

func (b *storeImpl) writeCVEParts(txn *dackbox.Transaction, parts *common.CVEParts, iTime *protoTypes.Timestamp) ([]byte, error) {
	if err := componentCVEEdgeDackBox.Upserter.UpsertIn(nil, parts.Edge, txn); err != nil {
		return nil, err
	}

	currCVEMsg, err := cveDackBox.Reader.ReadIn(cveDackBox.BucketHandler.GetKey(parts.CVE.GetId()), txn)
	if err != nil {
		return nil, err
	}
	if currCVEMsg != nil {
		currCVE := currCVEMsg.(*storage.CVE)
		parts.CVE.Suppressed = currCVE.GetSuppressed()
		parts.CVE.CreatedAt = currCVE.GetCreatedAt()
		parts.CVE.SuppressActivation = currCVE.GetSuppressActivation()
		parts.CVE.SuppressExpiry = currCVE.GetSuppressExpiry()

		parts.CVE.Types = cveUtil.AddCVETypeIfAbsent(currCVE.GetTypes(), storage.CVE_NODE_CVE)

		if parts.CVE.DistroSpecifics == nil {
			parts.CVE.DistroSpecifics = make(map[string]*storage.CVE_DistroSpecific)
		}
		for k, v := range currCVE.GetDistroSpecifics() {
			parts.CVE.DistroSpecifics[k] = v
		}
	} else {
		parts.CVE.CreatedAt = iTime

		// Populate the types slice for the new CVE.
		parts.CVE.Types = []storage.CVE_CVEType{storage.CVE_NODE_CVE}
	}

	parts.CVE.Type = storage.CVE_UNKNOWN_CVE

	if err := cveDackBox.Upserter.UpsertIn(nil, parts.CVE, txn); err != nil {
		return nil, err
	}
	return cveDackBox.KeyFunc(parts.CVE), nil
}

// Deleting a node and it's keys from the graph.
////////////////////////////////////////////////

func (b *storeImpl) deleteNodeKeys(keys *nodeKeySet) error {
	// Delete the keys
	deleteTxn, err := b.dacky.NewTransaction()
	if err != nil {
		return err
	}
	defer deleteTxn.Discard()

	err = nodeDackBox.Deleter.DeleteIn(keys.nodeKey, deleteTxn)
	if err != nil {
		return err
	}
	for _, component := range keys.componentKeys {
		if err := nodeComponentEdgeDackBox.Deleter.DeleteIn(component.nodeComponentEdgeKey, deleteTxn); err != nil {
			return err
		}
		// Only delete component and CVEs if there are no more references to it.
		if deleteTxn.Graph().CountRefsTo(component.componentKey) == 0 {
			if err := componentDackBox.Deleter.DeleteIn(component.componentKey, deleteTxn); err != nil {
				return err
			}
			for _, cve := range component.cveKeys {
				if err := componentCVEEdgeDackBox.Deleter.DeleteIn(cve.componentCVEEdgeKey, deleteTxn); err != nil {
					return err
				}
				if err := cveDackBox.Deleter.DeleteIn(cve.cveKey, deleteTxn); err != nil {
					return err
				}
			}
		}
	}

	// Delete the references from cluster to node.
	deleteTxn.Graph().DeleteRefsTo(keys.nodeKey)

	return deleteTxn.Commit()
}

// Reading a node from the DB.
//////////////////////////////

// readNodeMetadata reads the node without all its components/CVEs from the data store.
func (b *storeImpl) readNodeMetadata(txn *dackbox.Transaction, id string) (*storage.Node, error) {
	msg, err := nodeDackBox.Reader.ReadIn(nodeDackBox.BucketHandler.GetKey(id), txn)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, nil
	}
	return msg.(*storage.Node), nil
}

// readNode reads the node and all its components/CVEs from the data store.
func (b *storeImpl) readNode(txn *dackbox.Transaction, id string) (*storage.Node, error) {
	// Gather the keys for the node we want to read.
	keys, err := gatherKeysForNode(txn, id)
	if err != nil {
		return nil, err
	}

	parts, err := b.readNodeParts(txn, keys)
	if err != nil || parts == nil {
		return nil, err
	}

	return common.Merge(parts), nil
}

type nodeKeySet struct {
	nodeKey       []byte
	clusterKey    []byte
	componentKeys []*componentKeySet
	allKeys       [][]byte
}

type componentKeySet struct {
	nodeComponentEdgeKey []byte
	componentKey         []byte

	cveKeys []*cveKeySet
}

type cveKeySet struct {
	componentCVEEdgeKey []byte
	cveKey              []byte
}

func (b *storeImpl) readNodeParts(txn *dackbox.Transaction, keys *nodeKeySet) (*common.NodeParts, error) {
	// Read the objects for the keys.
	parts := &common.NodeParts{}
	msg, err := nodeDackBox.Reader.ReadIn(keys.nodeKey, txn)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, nil
	}
	parts.Node = msg.(*storage.Node)
	for _, component := range keys.componentKeys {
		componentPart := &common.ComponentParts{}
		compEdgeMsg, err := nodeComponentEdgeDackBox.Reader.ReadIn(component.nodeComponentEdgeKey, txn)
		if err != nil {
			return nil, err
		}
		if compEdgeMsg == nil {
			continue
		}
		compMsg, err := componentDackBox.Reader.ReadIn(component.componentKey, txn)
		if err != nil {
			return nil, err
		}
		if compMsg == nil {
			continue
		}
		componentPart.Edge = compEdgeMsg.(*storage.NodeComponentEdge)
		componentPart.Component = compMsg.(*storage.ImageComponent)
		for _, cve := range component.cveKeys {
			cveEdgeMsg, err := componentCVEEdgeDackBox.Reader.ReadIn(cve.componentCVEEdgeKey, txn)
			if err != nil {
				return nil, err
			}
			if cveEdgeMsg == nil {
				continue
			}
			cveMsg, err := cveDackBox.Reader.ReadIn(cve.cveKey, txn)
			if err != nil {
				return nil, err
			}
			if cveMsg == nil {
				continue
			}
			cve := cveMsg.(*storage.CVE)
			componentPart.Children = append(componentPart.Children, &common.CVEParts{
				Edge: cveEdgeMsg.(*storage.ComponentCVEEdge),
				CVE:  cve,
			})
		}
		parts.Children = append(parts.Children, componentPart)
	}
	return parts, nil
}

// Helper that walks the graph and collects the ids of the parts of a node.
func gatherKeysForNode(txn *dackbox.Transaction, nodeID string) (*nodeKeySet, error) {
	ret := &nodeKeySet{}
	var allKeys [][]byte

	// Get the key for the node.
	ret.nodeKey = nodeDackBox.BucketHandler.GetKey(nodeID)
	allKeys = append(allKeys, ret.nodeKey)

	allCVEsSet := set.NewStringSet()
	// Get the keys of the components.
	for _, componentKey := range componentDackBox.BucketHandler.GetFilteredRefsFrom(txn.Graph(), ret.nodeKey) {
		componentEdgeID := edges.EdgeID{ParentID: nodeID,
			ChildID: componentDackBox.BucketHandler.GetID(componentKey),
		}.ToString()
		component := &componentKeySet{
			componentKey:         componentKey,
			nodeComponentEdgeKey: nodeComponentEdgeDackBox.BucketHandler.GetKey(componentEdgeID),
		}
		for _, cveKey := range cveDackBox.BucketHandler.GetFilteredRefsFrom(txn.Graph(), componentKey) {
			cveID := cveDackBox.BucketHandler.GetID(cveKey)
			cveEdgeID := edges.EdgeID{
				ParentID: componentDackBox.BucketHandler.GetID(componentKey),
				ChildID:  cveID,
			}.ToString()
			cve := &cveKeySet{
				componentCVEEdgeKey: componentCVEEdgeDackBox.BucketHandler.GetKey(cveEdgeID),
				cveKey:              cveKey,
			}
			component.cveKeys = append(component.cveKeys, cve)
			allKeys = append(allKeys, cve.cveKey)
			allKeys = append(allKeys, cve.componentCVEEdgeKey)

			allCVEsSet.Add(cveID)
		}
		ret.componentKeys = append(ret.componentKeys, component)
		allKeys = append(allKeys, component.componentKey)
		allKeys = append(allKeys, component.nodeComponentEdgeKey)
	}
	clusterKeys := clusterDackBox.BucketHandler.GetFilteredRefsFrom(txn.Graph(), ret.nodeKey)
	allKeys = append(allKeys, clusterKeys...)

	// Generate a set of all the keys.
	ret.allKeys = sortedkeys.Sort(allKeys)

	if len(clusterKeys) != 1 {
		return ret, nil
	}

	ret.clusterKey = clusterKeys[0]

	return ret, nil
}
