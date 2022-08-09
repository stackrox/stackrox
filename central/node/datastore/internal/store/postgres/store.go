package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	protoTypes "github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/node/datastore/internal/store/common/v2"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/dackbox/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"gorm.io/gorm"
)

const (
	nodesTable              = pkgSchema.NodesTableName
	nodeComponentEdgesTable = pkgSchema.NodeComponentEdgesTableName
	nodeComponentsTable     = pkgSchema.NodeComponentsTableName
	componentCVEEdgesTable  = pkgSchema.NodeComponentsCvesEdgesTableName
	nodeCVEsTable           = pkgSchema.NodeCvesTableName

	getNodeMetaStmt = "SELECT serialized FROM " + nodesTable + " WHERE Id = $1"

	// using copyFrom, we may not even want to batch.  It would probably be simpler
	// to deal with failures if we just sent it all.  Something to think about as we
	// proceed and move into more e2e and larger performance testing
	batchSize = 10000
)

var (
	log            = logging.LoggerForModule()
	schema         = pkgSchema.NodesSchema
	targetResource = resources.Node
)

type nodePartsAsSlice struct {
	node               *storage.Node
	components         []*storage.NodeComponent
	vulns              []*storage.NodeCVE
	nodeComponentEdges []*storage.NodeComponentEdge
	componentCVEEdges  []*storage.NodeComponentCVEEdge
}

// Store provides storage functionality for full nodes.
type Store interface {
	Count(ctx context.Context) (int, error)
	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.Node, bool, error)
	Upsert(ctx context.Context, obj *storage.Node) error
	Delete(ctx context.Context, id string) error
	GetIDs(ctx context.Context) ([]string, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.Node, []int, error)
	// GetNodeMetadata gets the node without scan/component data.
	GetNodeMetadata(ctx context.Context, id string) (*storage.Node, bool, error)

	AckKeysIndexed(ctx context.Context, keys ...string) error
	GetKeysToIndex(ctx context.Context) ([]string, error)
}

type storeImpl struct {
	db                 *pgxpool.Pool
	noUpdateTimestamps bool
	keyFence           concurrency.KeyFence
}

// New returns a new Store instance using the provided sql instance.
func New(db *pgxpool.Pool, noUpdateTimestamps bool, keyFence concurrency.KeyFence) Store {
	return &storeImpl{
		db:                 db,
		noUpdateTimestamps: noUpdateTimestamps,
		keyFence:           keyFence,
	}
}

func (s *storeImpl) insertIntoNodes(
	ctx context.Context,
	tx pgx.Tx,
	parts *nodePartsAsSlice,
	scanUpdated bool,
	iTime *protoTypes.Timestamp,
) error {
	cloned := parts.node
	if cloned.GetScan().GetComponents() != nil {
		cloned = parts.node.Clone()
		cloned.Scan.Components = nil
	}
	serialized, marshalErr := cloned.Marshal()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		cloned.GetId(),
		cloned.GetName(),
		cloned.GetClusterId(),
		cloned.GetClusterName(),
		cloned.GetLabels(),
		cloned.GetAnnotations(),
		pgutils.NilOrTime(cloned.GetJoinedAt()),
		cloned.GetContainerRuntime().GetVersion(),
		cloned.GetOsImage(),
		pgutils.NilOrTime(cloned.GetLastUpdated()),
		pgutils.NilOrTime(cloned.GetScan().GetScanTime()),
		cloned.GetComponents(),
		cloned.GetCves(),
		cloned.GetFixableCves(),
		cloned.GetRiskScore(),
		cloned.GetTopCvss(),
		serialized,
	}

	finalStr := "INSERT INTO nodes (Id, Name, ClusterId, ClusterName, Labels, Annotations, JoinedAt, ContainerRuntime_Version, OsImage, LastUpdated, Scan_ScanTime, Components, Cves, FixableCves, RiskScore, TopCvss, serialized) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, Name = EXCLUDED.Name, ClusterId = EXCLUDED.ClusterId, ClusterName = EXCLUDED.ClusterName, Labels = EXCLUDED.Labels, Annotations = EXCLUDED.Annotations, JoinedAt = EXCLUDED.JoinedAt, ContainerRuntime_Version = EXCLUDED.ContainerRuntime_Version, OsImage = EXCLUDED.OsImage, LastUpdated = EXCLUDED.LastUpdated, Scan_ScanTime = EXCLUDED.Scan_ScanTime, Components = EXCLUDED.Components, Cves = EXCLUDED.Cves, FixableCves = EXCLUDED.FixableCves, RiskScore = EXCLUDED.RiskScore, TopCvss = EXCLUDED.TopCvss, serialized = EXCLUDED.serialized"
	_, err := tx.Exec(ctx, finalStr, values...)
	if err != nil {
		return err
	}

	var query string

	for childIdx, child := range cloned.GetTaints() {
		if err := insertIntoNodesTaints(ctx, tx, child, cloned.GetId(), childIdx); err != nil {
			return err
		}
	}

	query = "delete from nodes_taints where nodes_Id = $1 AND idx >= $2"
	_, err = tx.Exec(ctx, query, cloned.GetId(), len(cloned.GetTaints()))
	if err != nil {
		return err
	}
	if !scanUpdated {
		return nil
	}

	// DO NOT CHANGE THE ORDER.
	if err := copyFromNodeComponentEdges(ctx, tx, cloned.GetId(), parts.nodeComponentEdges...); err != nil {
		return err
	}
	if err := copyFromNodeComponents(ctx, tx, parts.components...); err != nil {
		return err
	}

	if err := copyFromNodeComponentCVEEdges(ctx, tx, parts.componentCVEEdges...); err != nil {
		return err
	}
	return copyFromNodeCves(ctx, tx, iTime, parts.vulns...)
}

func getPartsAsSlice(parts *common.NodeParts) *nodePartsAsSlice {
	components := make([]*storage.NodeComponent, 0, len(parts.Children))
	nodeComponentEdges := make([]*storage.NodeComponentEdge, 0, len(parts.Children))
	vulnMap := make(map[string]*storage.NodeCVE)
	var componentCVEEdges []*storage.NodeComponentCVEEdge
	for _, child := range parts.Children {
		components = append(components, child.Component)
		nodeComponentEdges = append(nodeComponentEdges, child.Edge)
		for _, gChild := range child.Children {
			componentCVEEdges = append(componentCVEEdges, gChild.Edge)
			vulnMap[gChild.CVE.GetId()] = gChild.CVE
		}
	}
	vulns := make([]*storage.NodeCVE, 0, len(vulnMap))
	for _, vuln := range vulnMap {
		vulns = append(vulns, vuln)
	}
	return &nodePartsAsSlice{
		node:               parts.Node,
		components:         components,
		vulns:              vulns,
		nodeComponentEdges: nodeComponentEdges,
		componentCVEEdges:  componentCVEEdges,
	}
}

func insertIntoNodesTaints(ctx context.Context, tx pgx.Tx, obj *storage.Taint, nodeID string, idx int) error {

	values := []interface{}{
		// parent primary keys start
		nodeID,
		idx,
		obj.GetKey(),
		obj.GetValue(),
		obj.GetTaintEffect(),
	}

	finalStr := "INSERT INTO nodes_taints (nodes_Id, idx, Key, Value, TaintEffect) VALUES($1, $2, $3, $4, $5) ON CONFLICT(nodes_Id, idx) DO UPDATE SET nodes_Id = EXCLUDED.nodes_Id, idx = EXCLUDED.idx, Key = EXCLUDED.Key, Value = EXCLUDED.Value, TaintEffect = EXCLUDED.TaintEffect"
	_, err := tx.Exec(ctx, finalStr, values...)
	if err != nil {
		return err
	}

	return nil
}

func copyFromNodeComponents(ctx context.Context, tx pgx.Tx, objs ...*storage.NodeComponent) error {
	inputRows := [][]interface{}{}
	var err error
	var deletes []string
	copyCols := []string{
		"id",
		"name",
		"version",
		"priority",
		"riskscore",
		"topcvss",
		"serialized",
	}

	for idx, obj := range objs {
		serialized, marshalErr := obj.Marshal()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{
			obj.GetId(),
			obj.GetName(),
			obj.GetVersion(),
			obj.GetPriority(),
			obj.GetRiskScore(),
			obj.GetTopCvss(),
			serialized,
		})

		// Add the id to be deleted.
		deletes = append(deletes, obj.GetId())

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// Copy does not upsert so have to delete first.
			_, err = tx.Exec(ctx, "DELETE FROM "+nodeComponentsTable+" WHERE id = ANY($1::text[])", deletes)
			if err != nil {
				return err
			}

			// clear the inserts for the next batch
			deletes = nil

			_, err = tx.CopyFrom(ctx, pgx.Identifier{nodeComponentsTable}, copyCols, pgx.CopyFromRows(inputRows))
			if err != nil {
				return err
			}

			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}
	return removeOrphanedNodeComponent(ctx, tx)
}

func copyFromNodeComponentEdges(ctx context.Context, tx pgx.Tx, nodeID string, objs ...*storage.NodeComponentEdge) error {
	inputRows := [][]interface{}{}
	var err error
	copyCols := []string{
		"id",
		"nodeid",
		"nodecomponentid",
		"serialized",
	}

	// Copy does not upsert so have to delete first. This also ensures newly orphaned component edges are removed.
	_, err = tx.Exec(ctx, "DELETE FROM "+nodeComponentEdgesTable+" WHERE nodeid = $1", nodeID)
	if err != nil {
		return err
	}

	for idx, obj := range objs {
		serialized, marshalErr := obj.Marshal()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{
			obj.GetId(),
			obj.GetNodeId(),
			obj.GetNodeComponentId(),
			serialized,
		})

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			_, err = tx.CopyFrom(ctx, pgx.Identifier{nodeComponentEdgesTable}, copyCols, pgx.CopyFromRows(inputRows))
			if err != nil {
				return err
			}

			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}
	return nil
}

func copyFromNodeCves(ctx context.Context, tx pgx.Tx, iTime *protoTypes.Timestamp, objs ...*storage.NodeCVE) error {
	inputRows := [][]interface{}{}

	var err error

	// This is a copy so first we must delete the rows and re-add them
	var deletes []string

	copyCols := []string{
		"id",
		"cvebaseinfo_cve",
		"cvebaseinfo_publishedon",
		"cvebaseinfo_createdat",
		"cvss",
		"severity",
		"impactscore",
		"snoozed",
		"snoozeexpiry",
		"serialized",
	}

	ids := set.NewStringSet()
	for _, obj := range objs {
		ids.Add(obj.GetId())
	}
	existingCVEs, err := getCVEs(ctx, tx, ids.AsSlice())
	if err != nil {
		return err
	}

	for idx, obj := range objs {
		if storedCVE := existingCVEs[obj.GetId()]; storedCVE != nil {
			obj.Snoozed = storedCVE.GetSnoozed()
			obj.CveBaseInfo.CreatedAt = storedCVE.GetCveBaseInfo().GetCreatedAt()
			obj.SnoozeStart = storedCVE.GetSnoozeStart()
			obj.SnoozeExpiry = storedCVE.GetSnoozeExpiry()
		} else {
			obj.CveBaseInfo.CreatedAt = iTime
		}

		serialized, marshalErr := obj.Marshal()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{
			obj.GetId(),
			obj.GetCveBaseInfo().GetCve(),
			pgutils.NilOrTime(obj.GetCveBaseInfo().GetPublishedOn()),
			pgutils.NilOrTime(obj.GetCveBaseInfo().GetCreatedAt()),
			obj.GetCvss(),
			obj.GetSeverity(),
			obj.GetImpactScore(),
			obj.GetSnoozed(),
			pgutils.NilOrTime(obj.GetSnoozeExpiry()),
			serialized,
		})

		// Add the id to be deleted.
		deletes = append(deletes, obj.GetId())

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// Copy does not upsert so have to delete first.
			_, err = tx.Exec(ctx, "DELETE FROM "+nodeCVEsTable+" WHERE id = ANY($1::text[])", deletes)
			if err != nil {
				return err
			}
			// Clear the inserts for the next batch.
			deletes = nil

			_, err = tx.CopyFrom(ctx, pgx.Identifier{nodeCVEsTable}, copyCols, pgx.CopyFromRows(inputRows))
			if err != nil {
				return err
			}

			// Clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}
	return removeOrphanedNodeCVEs(ctx, tx)
}

func copyFromNodeComponentCVEEdges(ctx context.Context, tx pgx.Tx, objs ...*storage.NodeComponentCVEEdge) error {
	inputRows := [][]interface{}{}
	var err error
	deletes := set.NewStringSet()
	copyCols := []string{
		"id",
		"isfixable",
		"fixedby",
		"nodecomponentid",
		"nodecveid",
		"serialized",
	}

	for idx, obj := range objs {
		serialized, marshalErr := obj.Marshal()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{
			obj.GetId(),
			obj.GetIsFixable(),
			obj.GetFixedBy(),
			obj.GetNodeComponentId(),
			obj.GetNodeCveId(),
			serialized,
		})

		// Add the id to be deleted.
		deletes.Add(obj.GetId())

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// Copy does not upsert so have to delete first.
			_, err = tx.Exec(ctx, "DELETE FROM "+componentCVEEdgesTable+" WHERE id = ANY($1::text[])", deletes.AsSlice())
			if err != nil {
				return err
			}

			// Clear the inserts for the next batch
			deletes = nil

			_, err = tx.CopyFrom(ctx, pgx.Identifier{componentCVEEdgesTable}, copyCols, pgx.CopyFromRows(inputRows))
			if err != nil {
				return err
			}

			// Clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}
	// Due to referential constraint orphaned component-cve edges are removed when orphaned image components are removed.
	return nil
}

func removeOrphanedNodeComponent(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, "DELETE FROM "+nodeComponentsTable+" WHERE not exists (select "+nodeComponentEdgesTable+".nodecomponentid from "+nodeComponentEdgesTable+" where "+nodeComponentsTable+".id = "+nodeComponentEdgesTable+".nodecomponentid)")
	if err != nil {
		return err
	}
	return nil
}

func removeOrphanedNodeCVEs(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, "DELETE FROM "+nodeCVEsTable+" WHERE not exists (select "+componentCVEEdgesTable+".nodecveid from "+componentCVEEdgesTable+" where "+nodeCVEsTable+".id = "+componentCVEEdgesTable+".nodecveid)")
	if err != nil {
		return err
	}
	return nil
}

func (s *storeImpl) isUpdated(ctx context.Context, node *storage.Node) (bool, bool, error) {
	oldNode, found, err := s.GetNodeMetadata(ctx, node.GetId())
	if err != nil {
		return false, false, err
	}
	if !found {
		return true, true, nil
	}

	scanUpdated := false
	// We skip rewriting components and cves if scan is not newer, hence we do not need to merge.
	if oldNode.GetScan().GetScanTime().Compare(node.GetScan().GetScanTime()) > 0 {
		node.Scan = oldNode.Scan
	} else {
		scanUpdated = true
	}

	// If the node in the DB is latest, then use its risk score and scan stats
	if !scanUpdated {
		node.RiskScore = oldNode.GetRiskScore()
		node.SetComponents = oldNode.GetSetComponents()
		node.SetCves = oldNode.GetSetCves()
		node.SetFixable = oldNode.GetSetFixable()
		node.SetTopCvss = oldNode.GetSetTopCvss()
	}
	return true, scanUpdated, nil
}

func (s *storeImpl) upsert(ctx context.Context, obj *storage.Node) error {
	iTime := protoTypes.TimestampNow()

	if !s.noUpdateTimestamps {
		obj.LastUpdated = iTime
	}
	metadataUpdated, scanUpdated, err := s.isUpdated(ctx, obj)
	if err != nil {
		return err
	}
	if !metadataUpdated && !scanUpdated {
		return nil
	}

	nodeParts := getPartsAsSlice(common.Split(obj, scanUpdated))
	keys := gatherKeys(nodeParts)

	conn, release, err := s.acquireConn(ctx, ops.Upsert, "Node")
	if err != nil {
		return err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	return s.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet(keys...), func() error {
		if err := s.insertIntoNodes(ctx, tx, nodeParts, scanUpdated, iTime); err != nil {
			if err := tx.Rollback(ctx); err != nil {
				return err
			}
			return err
		}
		return tx.Commit(ctx)
	})
}

// Upsert upserts node into the store.
func (s *storeImpl) Upsert(ctx context.Context, obj *storage.Node) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Upsert, "Node")

	return s.upsert(ctx, obj)
}

func (s *storeImpl) copyFromNodesTaints(ctx context.Context, tx pgx.Tx, nodeID string, objs ...*storage.Taint) error {
	inputRows := [][]interface{}{}
	var err error
	copyCols := []string{
		"nodes_id",
		"idx",
		"key",
		"value",
		"tainteffect",
	}

	for idx, obj := range objs {
		// Todo: ROX-9499 Figure out how to more cleanly template around this issue.
		log.Debugf("This is here for now because there is an issue with pods_TerminatedInstances where the obj in the loop is not used as it only consists of the parent id and the idx.  Putting this here as a stop gap to simply use the object.  %s", obj)

		inputRows = append(inputRows, []interface{}{
			nodeID,
			idx,
			obj.GetKey(),
			obj.GetValue(),
			obj.GetTaintEffect(),
		})

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// copy does not upsert so have to delete first.  parent deletion cascades so only need to
			// delete for the top level parent

			_, err = tx.CopyFrom(ctx, pgx.Identifier{"nodes_taints"}, copyCols, pgx.CopyFromRows(inputRows))
			if err != nil {
				return err
			}

			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}
	return err
}

// Count returns the number of objects in the store
func (s *storeImpl) Count(ctx context.Context) (int, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Count, "Node")

	var sacQueryFilter *v1.Query

	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_ACCESS).Resource(targetResource)
	scopeTree, err := scopeChecker.EffectiveAccessScope(permissions.View(targetResource))
	if err != nil {
		return 0, err
	}
	sacQueryFilter, err = sac.BuildClusterLevelSACQueryFilter(scopeTree)

	if err != nil {
		return 0, err
	}

	return postgres.RunCountRequestForSchema(schema, sacQueryFilter, s.db)
}

// Exists returns if the id exists in the store
func (s *storeImpl) Exists(ctx context.Context, id string) (bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Exists, "Node")

	var sacQueryFilter *v1.Query
	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_ACCESS).Resource(targetResource)
	scopeTree, err := scopeChecker.EffectiveAccessScope(permissions.View(targetResource))
	if err != nil {
		return false, err
	}
	sacQueryFilter, err = sac.BuildClusterLevelSACQueryFilter(scopeTree)
	if err != nil {
		return false, err
	}

	q := search.ConjunctionQuery(
		sacQueryFilter,
		search.NewQueryBuilder().AddDocIDs(id).ProtoQuery(),
	)

	count, err := postgres.RunCountRequestForSchema(schema, q, s.db)
	return count == 1, err
}

// Get returns the object, if it exists from the store
func (s *storeImpl) Get(ctx context.Context, id string) (*storage.Node, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "Node")

	conn, release, err := s.acquireConn(ctx, ops.Get, "Node")
	if err != nil {
		return nil, false, err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, false, err
	}
	return s.getFullNode(ctx, tx, id)
}

func (s *storeImpl) getFullNode(ctx context.Context, tx pgx.Tx, nodeID string) (*storage.Node, bool, error) {
	row := tx.QueryRow(ctx, getNodeMetaStmt, nodeID)
	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}

	var node storage.Node
	if err := proto.Unmarshal(data, &node); err != nil {
		return nil, false, err
	}

	componentEdgeMap, err := getNodeComponentEdges(ctx, tx, nodeID)
	if err != nil {
		return nil, false, err
	}
	componentIDs := make([]string, 0, len(componentEdgeMap))
	for _, val := range componentEdgeMap {
		componentIDs = append(componentIDs, val.GetNodeComponentId())
	}

	componentMap, err := getNodeComponents(ctx, tx, componentIDs)
	if err != nil {
		return nil, false, err
	}

	if len(componentEdgeMap) != len(componentMap) {
		utils.Should(
			errors.Errorf("Number of node component from edges (%d) is unexpected (%d) for node %s (id=%s)",
				len(componentEdgeMap), len(componentMap), node.GetName(), node.GetId()),
		)
	}
	componentCVEEdgeMap, err := getComponentCVEEdges(ctx, tx, componentIDs)
	if err != nil {
		return nil, false, err
	}

	cveIDs := set.NewStringSet()
	for _, edges := range componentCVEEdgeMap {
		for _, edge := range edges {
			cveIDs.Add(edge.GetNodeCveId())
		}
	}

	cveMap, err := getCVEs(ctx, tx, cveIDs.AsSlice())
	if err != nil {
		return nil, false, err
	}

	nodeParts := &common.NodeParts{
		Node:     &node,
		Children: []*common.ComponentParts{},
	}
	for componentID, component := range componentMap {
		child := &common.ComponentParts{
			Edge:      componentEdgeMap[componentID],
			Component: component,
			Children:  []*common.CVEParts{},
		}

		for _, edge := range componentCVEEdgeMap[componentID] {
			child.Children = append(child.Children, &common.CVEParts{
				Edge: edge,
				CVE:  cveMap[edge.GetNodeCveId()],
			})
		}
		nodeParts.Children = append(nodeParts.Children, child)
	}
	return common.Merge(nodeParts), true, nil
}

func getNodeComponentEdges(ctx context.Context, tx pgx.Tx, nodeID string) (map[string]*storage.NodeComponentEdge, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "NodeComponentEdge")

	rows, err := tx.Query(ctx, "SELECT serialized FROM "+nodeComponentEdgesTable+" WHERE nodeid = $1", nodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	componentIDToEdgeMap := make(map[string]*storage.NodeComponentEdge)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		msg := &storage.NodeComponentEdge{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		componentIDToEdgeMap[msg.GetNodeComponentId()] = msg
	}
	return componentIDToEdgeMap, nil
}

func getNodeComponents(ctx context.Context, tx pgx.Tx, componentIDs []string) (map[string]*storage.NodeComponent, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "NodeComponent")

	rows, err := tx.Query(ctx, "SELECT serialized FROM "+nodeComponentsTable+" WHERE id = ANY($1::text[])", componentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	idToComponentMap := make(map[string]*storage.NodeComponent)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		msg := &storage.NodeComponent{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		idToComponentMap[msg.GetId()] = msg
	}
	return idToComponentMap, nil
}

func getComponentCVEEdges(ctx context.Context, tx pgx.Tx, componentIDs []string) (map[string][]*storage.NodeComponentCVEEdge, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "NodeComponentCVERelations")

	rows, err := tx.Query(ctx, "SELECT serialized FROM "+componentCVEEdgesTable+" WHERE nodecomponentid = ANY($1::text[])", componentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	componentIDToEdgesMap := make(map[string][]*storage.NodeComponentCVEEdge)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		msg := &storage.NodeComponentCVEEdge{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		componentIDToEdgesMap[msg.GetNodeComponentId()] = append(componentIDToEdgesMap[msg.GetNodeComponentId()], msg)
	}
	return componentIDToEdgesMap, nil
}

func (s *storeImpl) acquireConn(ctx context.Context, op ops.Op, typ string) (*pgxpool.Conn, func(), error) {
	defer metrics.SetAcquireDBConnDuration(time.Now(), op, typ)
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	return conn, conn.Release, nil
}

// Delete removes the specified ID from the store
func (s *storeImpl) Delete(ctx context.Context, id string) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "Node")

	conn, release, err := s.acquireConn(ctx, ops.Remove, "Node")
	if err != nil {
		return err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil
	}
	return s.deleteNodeTree(ctx, tx, id)
}

func (s *storeImpl) deleteNodeTree(ctx context.Context, tx pgx.Tx, nodeID string) error {
	// Delete from node table.
	if _, err := tx.Exec(ctx, "delete from "+nodesTable+" where Id = $1", nodeID); err != nil {
		return err
	}

	// Delete orphaned node components.
	if _, err := tx.Exec(ctx, "delete from "+nodeComponentsTable+" where not exists (select "+nodeComponentEdgesTable+".nodecomponentid FROM "+nodeComponentEdgesTable+" where "+nodeComponentEdgesTable+".nodecomponentid = "+nodeComponentsTable+".id)"); err != nil {
		return err
	}

	// Delete orphaned cves.
	if _, err := tx.Exec(ctx, "delete from "+nodeCVEsTable+" where not exists (select "+componentCVEEdgesTable+".nodecveid FROM "+componentCVEEdgesTable+" where "+componentCVEEdgesTable+".nodecveid = "+nodeCVEsTable+".id)"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// GetIDs returns all the IDs for the store
func (s *storeImpl) GetIDs(ctx context.Context) ([]string, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetAll, "NodeIDs")
	var sacQueryFilter *v1.Query

	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_ACCESS).Resource(targetResource)
	scopeTree, err := scopeChecker.EffectiveAccessScope(permissions.View(targetResource))
	if err != nil {
		return nil, err
	}
	sacQueryFilter, err = sac.BuildClusterLevelSACQueryFilter(scopeTree)
	if err != nil {
		return nil, err
	}
	result, err := postgres.RunSearchRequestForSchema(schema, sacQueryFilter, s.db)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(result))
	for _, entry := range result {
		ids = append(ids, entry.ID)
	}

	return ids, nil
}

// GetMany returns the objects specified by the IDs or the index in the missing indices slice
func (s *storeImpl) GetMany(ctx context.Context, ids []string) ([]*storage.Node, []int, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "Node")

	conn, release, err := s.acquireConn(ctx, ops.GetMany, "Node")
	if err != nil {
		return nil, nil, err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}

	resultsByID := make(map[string]*storage.Node)
	for _, id := range ids {
		msg, found, err := s.getFullNode(ctx, tx, id)
		if err != nil {
			return nil, nil, err
		}
		if !found {
			continue
		}
		resultsByID[msg.GetId()] = msg
	}

	missingIndices := make([]int, 0, len(ids)-len(resultsByID))
	// It is important that the elems are populated in the same order as the input ids
	// slice, since some calling code relies on that to maintain order.
	elems := make([]*storage.Node, 0, len(resultsByID))
	for i, id := range ids {
		if result, ok := resultsByID[id]; !ok {
			missingIndices = append(missingIndices, i)
		} else {
			elems = append(elems, result)
		}
	}
	return elems, missingIndices, nil
}

// GetNodeMetadata gets the node without scan/component data.
func (s *storeImpl) GetNodeMetadata(ctx context.Context, id string) (*storage.Node, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "NodeMetadata")

	conn, release, err := s.acquireConn(ctx, ops.Get, "NodeMetadata")
	if err != nil {
		return nil, false, err
	}
	defer release()

	row := conn.QueryRow(ctx, getNodeMetaStmt, id)
	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}

	var msg storage.Node
	if err := proto.Unmarshal(data, &msg); err != nil {
		return nil, false, err
	}
	return &msg, true, nil
}

//// Used for testing

// CreateTableAndNewStore returns a new Store instance for testing
func CreateTableAndNewStore(ctx context.Context, t *testing.T, db *pgxpool.Pool, gormDB *gorm.DB, noUpdateTimestamps bool) Store {
	pgutils.CreateTableFromModel(ctx, gormDB, pkgSchema.CreateTableClustersStmt)
	pgutils.CreateTableFromModel(ctx, gormDB, pkgSchema.CreateTableNodesStmt)
	pgutils.CreateTableFromModel(ctx, gormDB, pkgSchema.CreateTableNodeComponentsStmt)
	pgutils.CreateTableFromModel(ctx, gormDB, pkgSchema.CreateTableNodeCvesStmt)
	pgutils.CreateTableFromModel(ctx, gormDB, pkgSchema.CreateTableNodeComponentEdgesStmt)
	pgutils.CreateTableFromModel(ctx, gormDB, pkgSchema.CreateTableNodeComponentsCvesEdgesStmt)
	return New(db, noUpdateTimestamps, concurrency.NewKeyFence())
}

func dropTableNodes(ctx context.Context, db *pgxpool.Pool) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS nodes CASCADE")
	dropTableNodesTaints(ctx, db)
	dropTableNodesComponents(ctx, db)
	dropTableNodeCVEs(ctx, db)
	dropTableNodeComponentEdges(ctx, db)
	dropTableComponentCVEEdges(ctx, db)
}

func dropTableNodesTaints(ctx context.Context, db *pgxpool.Pool) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS nodes_taints CASCADE")
}

func dropTableNodesComponents(ctx context.Context, db *pgxpool.Pool) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS "+nodeComponentsTable+" CASCADE")
}

func dropTableNodeCVEs(ctx context.Context, db *pgxpool.Pool) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS "+nodeCVEsTable+" CASCADE")
}

func dropTableComponentCVEEdges(ctx context.Context, db *pgxpool.Pool) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS "+componentCVEEdgesTable+" CASCADE")
}

func dropTableNodeComponentEdges(ctx context.Context, db *pgxpool.Pool) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS "+nodeComponentEdgesTable+" CASCADE")
}

// Destroy drops all node tree tables.
func Destroy(ctx context.Context, db *pgxpool.Pool) {
	dropTableNodes(ctx, db)
}

//// Stubs for satisfying legacy interfaces

// AckKeysIndexed acknowledges the passed keys were indexed
func (s *storeImpl) AckKeysIndexed(ctx context.Context, keys ...string) error {
	return nil
}

// GetKeysToIndex returns the keys that need to be indexed
func (s *storeImpl) GetKeysToIndex(ctx context.Context) ([]string, error) {
	return nil, nil
}

func getCVEs(ctx context.Context, tx pgx.Tx, cveIDs []string) (map[string]*storage.NodeCVE, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "NodeCVEs")

	rows, err := tx.Query(ctx, "SELECT serialized FROM "+nodeCVEsTable+" WHERE id = ANY($1::text[])", cveIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	idToCVEMap := make(map[string]*storage.NodeCVE)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		msg := &storage.NodeCVE{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		idToCVEMap[msg.GetId()] = msg
	}
	return idToCVEMap, nil
}

func gatherKeys(parts *nodePartsAsSlice) [][]byte {
	// We only need to collect node, component, and vuln keys because edges are derived from those resources and edge
	// datastores are do not support upserts and deletes.
	keys := make([][]byte, 0, len(parts.components)+len(parts.vulns)+1)
	keys = append(keys, []byte(parts.node.GetId()))
	for _, component := range parts.components {
		keys = append(keys, []byte(component.GetId()))
	}
	for _, vuln := range parts.vulns {
		keys = append(keys, []byte(vuln.GetId()))
	}
	return keys
}

func scanIDs(rows pgx.Rows) ([]string, error) {
	defer rows.Close()
	var ids []string

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
