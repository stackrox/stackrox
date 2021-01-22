package dackbox

import (
	"time"

	protoTypes "github.com/gogo/protobuf/types"
	clusterDackBox "github.com/stackrox/rox/central/cluster/dackbox"
	componentCVEEdgeDackBox "github.com/stackrox/rox/central/componentcveedge/dackbox"
	cveDackBox "github.com/stackrox/rox/central/cve/dackbox"
	componentDackBox "github.com/stackrox/rox/central/imagecomponent/dackbox"
	"github.com/stackrox/rox/central/metrics"
	namespaceDackBox "github.com/stackrox/rox/central/namespace/dackbox"
	nodeDackBox "github.com/stackrox/rox/central/node/dackbox"
	"github.com/stackrox/rox/central/node/datastore/internal/store"
	nodeComponentEdgeDackBox "github.com/stackrox/rox/central/nodecomponentedge/dackbox"
	nodeCVEEdgeDackBox "github.com/stackrox/rox/central/nodecveedge/dackbox"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/dackbox/edges"
	"github.com/stackrox/rox/pkg/dackbox/sortedkeys"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/set"
)

const (
	typ = "Node"
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
func (b *storeImpl) Exists(id string) (bool, error) {
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

// GetNodes returns all nodes regardless of request
func (b *storeImpl) GetNodes() ([]*storage.Node, error) {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.GetAll, typ)

	branch, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return nil, err
	}
	defer branch.Discard()

	keys, err := nodeDackBox.Reader.ReadKeysIn(nodeDackBox.Bucket, branch)
	if err != nil {
		return nil, err
	}

	nodes := make([]*storage.Node, 0, len(keys))
	for _, key := range keys {
		node, err := b.readNode(branch, nodeDackBox.BucketHandler.GetID(key))
		if err != nil {
			return nil, err
		}
		if node != nil {
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

// CountNodes returns the number of nodes currently stored in the DB.
func (b *storeImpl) CountNodes() (int, error) {
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

// GetNode returns node with given id.
func (b *storeImpl) GetNode(id string) (*storage.Node, bool, error) {
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

// GetNodesBatch returns nodes with given ids.
func (b *storeImpl) GetNodesBatch(ids []string) ([]*storage.Node, []int, error) {
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
func (b *storeImpl) Upsert(node *storage.Node) error {
	defer metrics.SetDackboxOperationDurationTime(time.Now(), ops.Upsert, typ)

	iTime := protoTypes.TimestampNow()
	if !b.noUpdateTimestamps {
		node.LastUpdated = iTime
	}

	scanUpdated, err := b.isUpdated(node)
	if err != nil {
		return err
	}

	// Unlike images, nodes are not static, so we must continue upsert even if the scan is not updated.

	// If the node scan is not updated, skip updating that part in DB, i.e. rewriting components and cves.
	parts := Split(node, scanUpdated)

	clusterKey := clusterDackBox.BucketHandler.GetKey(node.GetClusterId())
	keysToUpdate := append(gatherKeysForNodeParts(parts), clusterKey)
	return b.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet(keysToUpdate...), func() error {
		return b.writeNodeParts(parts, clusterKey, iTime, scanUpdated)
	})
}

func (b *storeImpl) isUpdated(node *storage.Node) (bool, error) {
	txn, err := b.dacky.NewReadOnlyTransaction()
	if err != nil {
		return false, err
	}
	defer txn.Discard()

	msg, err := nodeDackBox.Reader.ReadIn(nodeDackBox.BucketHandler.GetKey(node.GetId()), txn)
	if err != nil {
		return false, err
	}
	// No node for given ID found, hence mark new node as latest
	if msg == nil {
		return true, nil
	}

	oldNode := msg.(*storage.Node)

	scanUpdated := false
	// We skip rewriting components and cves if scan is not newer, hence we do not need to merge.
	if oldNode.GetScan().GetScanTime().Compare(node.GetScan().GetScanTime()) > 0 {
		fullOldNode, err := b.readNode(txn, node.GetId())
		if err != nil {
			return false, err
		}
		node.Scan = fullOldNode.Scan
	} else {
		scanUpdated = true
	}

	// If the node in the DB is latest, then use its risk score and scan stats.
	if !scanUpdated {
		node.RiskScore = oldNode.GetRiskScore()
		node.SetComponents = oldNode.GetSetComponents()
		node.SetCves = oldNode.GetSetCves()
		node.SetFixable = oldNode.GetSetFixable()
		node.SetTopCvss = oldNode.GetSetTopCvss()
	}

	return scanUpdated, nil
}

// DeleteNode deletes an node and all its data.
func (b *storeImpl) Delete(id string) error {
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

func gatherKeysForNodeParts(parts *NodeParts) [][]byte {
	var allKeys [][]byte
	allKeys = append(allKeys, nodeDackBox.BucketHandler.GetKey(parts.node.GetId()))
	for _, componentParts := range parts.children {
		allKeys = append(allKeys, componentDackBox.BucketHandler.GetKey(componentParts.component.GetId()))
		for _, cveParts := range componentParts.children {
			allKeys = append(allKeys, cveDackBox.BucketHandler.GetKey(cveParts.cve.GetId()))
		}
	}
	return allKeys
}

func (b *storeImpl) writeNodeParts(parts *NodeParts, clusterKey []byte, iTime *protoTypes.Timestamp, scanUpdated bool) error {
	dackTxn, err := b.dacky.NewTransaction()
	if err != nil {
		return err
	}
	defer dackTxn.Discard()

	var componentKeys [][]byte
	// Update the node components and cves iff the node upsert has updated scan.
	// Note: In such cases, the loops in following block will not be entered anyways since len(parts.children) and len(parts.nodeCVEEdges) is 0.
	// This is more for good readability amidst the complex code.
	if scanUpdated {
		for _, componentData := range parts.children {
			componentKey, err := b.writeComponentParts(dackTxn, componentData, iTime)
			if err != nil {
				return err
			}
			componentKeys = append(componentKeys, componentKey)
		}

		if err := b.writeNodeCVEEdges(dackTxn, parts.nodeCVEEdges, iTime); err != nil {
			return err
		}
	}

	if err := nodeDackBox.Upserter.UpsertIn(nil, parts.node, dackTxn); err != nil {
		return err
	}

	nodeKey := nodeDackBox.KeyFunc(parts.node)

	err = dackTxn.Graph().AddRefs(clusterKey, nodeKey)
	if err != nil {
		return err
	}

	// Update the downstream node links in the graph iff the node upsert has updated scan.
	if scanUpdated {
		if err := dackTxn.Graph().SetRefs(nodeKey, componentKeys); err != nil {
			return err
		}
	}

	return dackTxn.Commit()
}

func (b *storeImpl) writeNodeCVEEdges(txn *dackbox.Transaction, edges map[string]*storage.NodeCVEEdge, iTime *protoTypes.Timestamp) error {
	for _, edge := range edges {
		// If node-cve edge exists, it means we have already determined and stored its first node occurrence.
		// If not, this is the first node occurrence.
		if exists, err := nodeCVEEdgeDackBox.Reader.ExistsIn(nodeCVEEdgeDackBox.BucketHandler.GetKey(edge.GetId()), txn); err != nil {
			return err
		} else if exists {
			continue
		}

		edge.FirstNodeOccurrence = iTime

		if err := nodeCVEEdgeDackBox.Upserter.UpsertIn(nil, edge, txn); err != nil {
			return err
		}
	}

	return nil
}

func (b *storeImpl) writeComponentParts(txn *dackbox.Transaction, parts *ComponentParts, iTime *protoTypes.Timestamp) ([]byte, error) {
	var cveKeys [][]byte
	for _, cveData := range parts.children {
		cveKey, err := b.writeCVEParts(txn, cveData, iTime)
		if err != nil {
			return nil, err
		}
		cveKeys = append(cveKeys, cveKey)
	}

	componentKey := componentDackBox.KeyFunc(parts.component)
	if err := nodeComponentEdgeDackBox.Upserter.UpsertIn(nil, parts.edge, txn); err != nil {
		return nil, err
	}
	if err := componentDackBox.Upserter.UpsertIn(nil, parts.component, txn); err != nil {
		return nil, err
	}

	if err := txn.Graph().SetRefs(componentKey, cveKeys); err != nil {
		return nil, err
	}
	return componentKey, nil
}

func (b *storeImpl) writeCVEParts(txn *dackbox.Transaction, parts *CVEParts, iTime *protoTypes.Timestamp) ([]byte, error) {
	if err := componentCVEEdgeDackBox.Upserter.UpsertIn(nil, parts.edge, txn); err != nil {
		return nil, err
	}

	currCVEMsg, err := cveDackBox.Reader.ReadIn(cveDackBox.BucketHandler.GetKey(parts.cve.GetId()), txn)
	if err != nil {
		return nil, err
	}
	if currCVEMsg != nil {
		currCVE := currCVEMsg.(*storage.CVE)
		parts.cve.Suppressed = currCVE.GetSuppressed()
		parts.cve.CreatedAt = currCVE.GetCreatedAt()
		parts.cve.SuppressActivation = currCVE.GetSuppressActivation()
		parts.cve.SuppressExpiry = currCVE.GetSuppressExpiry()
	} else {
		parts.cve.CreatedAt = iTime
	}
	if err := cveDackBox.Upserter.UpsertIn(nil, parts.cve, txn); err != nil {
		return nil, err
	}
	return cveDackBox.KeyFunc(parts.cve), nil
}

// Deleting a node and it's keys from the graph.
////////////////////////////////////////////////

func (b *storeImpl) deleteNodeKeys(keys *nodeKeySet) error {
	// Delete the keys
	upsertTxn, err := b.dacky.NewTransaction()
	if err != nil {
		return err
	}
	defer upsertTxn.Discard()

	// Cluster deletion is handled by the deployment datastore instead of here.
	// It deletes a cluster once all namespaces are deleted.

	err = nodeDackBox.Deleter.DeleteIn(keys.nodeKey, upsertTxn)
	if err != nil {
		return err
	}
	for _, component := range keys.componentKeys {
		if err := nodeComponentEdgeDackBox.Deleter.DeleteIn(component.nodeComponentEdgeKey, upsertTxn); err != nil {
			return err
		}
		if err := componentDackBox.Deleter.DeleteIn(component.componentKey, upsertTxn); err != nil {
			return err
		}
		for _, cve := range component.cveKeys {
			if err := componentCVEEdgeDackBox.Deleter.DeleteIn(cve.componentCVEEdgeKey, upsertTxn); err != nil {
				return err
			}
			if err := cveDackBox.Deleter.DeleteIn(cve.cveKey, upsertTxn); err != nil {
				return err
			}
		}
	}

	for _, nodeCVEEdgeKey := range keys.nodeCVEEdgeKeys {
		if err := nodeCVEEdgeDackBox.Deleter.DeleteIn(nodeCVEEdgeKey, upsertTxn); err != nil {
			return err
		}
	}

	// If the cluster has no more namespaces nor nodes, remove its refs. (Clusters only have forward refs)
	if keys.clusterKey != nil && len(namespaceDackBox.BucketHandler.FilterKeys(upsertTxn.Graph().GetRefsFrom(keys.clusterKey))) == 0 &&
		len(nodeDackBox.BucketHandler.FilterKeys(upsertTxn.Graph().GetRefsFrom(keys.clusterKey))) == 0 {
		if err := upsertTxn.Graph().DeleteRefsFrom(keys.clusterKey); err != nil {
			return err
		}
	}

	return upsertTxn.Commit()
}

// Reading a node from the DB.
//////////////////////////////

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

	return Merge(parts), nil
}

type nodeKeySet struct {
	nodeKey         []byte
	clusterKey      []byte
	componentKeys   []*componentKeySet
	nodeCVEEdgeKeys [][]byte
	allKeys         [][]byte
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

func (b *storeImpl) readNodeParts(txn *dackbox.Transaction, keys *nodeKeySet) (*NodeParts, error) {
	// Read the objects for the keys.
	parts := &NodeParts{nodeCVEEdges: make(map[string]*storage.NodeCVEEdge)}
	msg, err := nodeDackBox.Reader.ReadIn(keys.nodeKey, txn)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, nil
	}
	parts.node = msg.(*storage.Node)
	for _, component := range keys.componentKeys {
		componentPart := &ComponentParts{}
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
		componentPart.edge = compEdgeMsg.(*storage.NodeComponentEdge)
		componentPart.component = compMsg.(*storage.ImageComponent)
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
			componentPart.children = append(componentPart.children, &CVEParts{
				edge: cveEdgeMsg.(*storage.ComponentCVEEdge),
				cve:  cve,
			})
		}
		parts.children = append(parts.children, componentPart)
	}

	// Gather all the edges from node to cves and store it as a map from CVE IDs to *storage.NodeCVEEdge object.
	for _, nodeCVEEdgeKey := range keys.nodeCVEEdgeKeys {
		nodeCVEEdgeMsg, err := nodeCVEEdgeDackBox.Reader.ReadIn(nodeCVEEdgeKey, txn)
		if err != nil {
			return nil, err
		}

		if nodeCVEEdgeMsg == nil {
			continue
		}

		nodeCVEEdge := nodeCVEEdgeMsg.(*storage.NodeCVEEdge)
		edgeID, err := edges.FromString(nodeCVEEdge.GetId())
		if err != nil {
			return nil, err
		}
		parts.nodeCVEEdges[edgeID.ChildID] = nodeCVEEdge
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
	for _, componentKey := range componentDackBox.BucketHandler.FilterKeys(txn.Graph().GetRefsFrom(ret.nodeKey)) {
		componentEdgeID := edges.EdgeID{ParentID: nodeID,
			ChildID: componentDackBox.BucketHandler.GetID(componentKey),
		}.ToString()
		component := &componentKeySet{
			componentKey:         componentKey,
			nodeComponentEdgeKey: nodeComponentEdgeDackBox.BucketHandler.GetKey(componentEdgeID),
		}
		for _, cveKey := range cveDackBox.BucketHandler.FilterKeys(txn.Graph().GetRefsFrom(componentKey)) {
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

	for cveID := range allCVEsSet {
		nodeCVEEdgeID := edges.EdgeID{
			ParentID: nodeID,
			ChildID:  cveID,
		}.ToString()
		nodeCVEEdgeKey := nodeCVEEdgeDackBox.BucketHandler.GetKey(nodeCVEEdgeID)
		ret.nodeCVEEdgeKeys = append(ret.nodeCVEEdgeKeys, nodeCVEEdgeKey)
		allKeys = append(allKeys, nodeCVEEdgeKey)
	}

	clusterKeys := clusterDackBox.BucketHandler.FilterKeys(txn.Graph().GetRefsFrom(ret.nodeKey))
	allKeys = append(allKeys, clusterKeys...)

	// Generate a set of all the keys.
	ret.allKeys = sortedkeys.Sort(allKeys)

	if len(clusterKeys) != 1 {
		return ret, nil
	}

	ret.clusterKey = clusterKeys[0]

	return ret, nil
}
