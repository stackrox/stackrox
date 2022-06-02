package postgres

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	protoTypes "github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/node/datastore/internal/store/common"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
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
	DeleteMany(ctx context.Context, ids []string) error

	AckKeysIndexed(ctx context.Context, keys ...string) error
	GetKeysToIndex(ctx context.Context) ([]string, error)
}

type storeImpl struct {
	db                 *pgxpool.Pool
	noUpdateTimestamps bool
}

// New returns a new Store instance using the provided sql instance.
func New(ctx context.Context, db *pgxpool.Pool, noUpdateTimestamps bool) Store {
	pgutils.CreateTable(ctx, db, pkgSchema.CreateTableClustersStmt)
	pgutils.CreateTable(ctx, db, pkgSchema.CreateTableNodesStmt)
	pgutils.CreateTable(ctx, db, pkgSchema.CreateTableNodeComponentsStmt)
	pgutils.CreateTable(ctx, db, pkgSchema.CreateTableNodeCvesStmt)
	pgutils.CreateTable(ctx, db, pkgSchema.CreateTableNodeComponentEdgesStmt)
	pgutils.CreateTable(ctx, db, pkgSchema.CreateTableNodeComponentsCvesEdgesStmt)

	return &storeImpl{
		db:                 db,
		noUpdateTimestamps: noUpdateTimestamps,
	}
}

func insertIntoNodes(ctx context.Context, tx pgx.Tx, obj *storage.Node, scanUpdated bool, iTime *protoTypes.Timestamp) error {
	cloned := obj
	if cloned.GetScan().GetComponents() != nil {
		cloned = obj.Clone()
		cloned.Scan.Components = nil
	}
	serialized, marshalErr := cloned.Marshal()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		obj.GetId(),
		obj.GetName(),
		obj.GetClusterId(),
		obj.GetClusterName(),
		obj.GetLabels(),
		obj.GetAnnotations(),
		pgutils.NilOrTime(obj.GetJoinedAt()),
		obj.GetContainerRuntime().GetVersion(),
		obj.GetOsImage(),
		pgutils.NilOrTime(obj.GetLastUpdated()),
		pgutils.NilOrTime(obj.GetScan().GetScanTime()),
		obj.GetComponents(),
		obj.GetCves(),
		obj.GetFixableCves(),
		obj.GetRiskScore(),
		obj.GetTopCvss(),
		serialized,
	}

	finalStr := "INSERT INTO nodes (Id, Name, ClusterId, ClusterName, Labels, Annotations, JoinedAt, ContainerRuntime_Version, OsImage, LastUpdated, Scan_ScanTime, Components, Cves, FixableCves, RiskScore, TopCvss, serialized) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, Name = EXCLUDED.Name, ClusterId = EXCLUDED.ClusterId, ClusterName = EXCLUDED.ClusterName, Labels = EXCLUDED.Labels, Annotations = EXCLUDED.Annotations, JoinedAt = EXCLUDED.JoinedAt, ContainerRuntime_Version = EXCLUDED.ContainerRuntime_Version, OsImage = EXCLUDED.OsImage, LastUpdated = EXCLUDED.LastUpdated, Scan_ScanTime = EXCLUDED.Scan_ScanTime, Components = EXCLUDED.Components, Cves = EXCLUDED.Cves, FixableCves = EXCLUDED.FixableCves, RiskScore = EXCLUDED.RiskScore, TopCvss = EXCLUDED.TopCvss, serialized = EXCLUDED.serialized"
	_, err := tx.Exec(ctx, finalStr, values...)
	if err != nil {
		return err
	}

	var query string

	for childIdx, child := range obj.GetTaints() {
		if err := insertIntoNodesTaints(ctx, tx, child, obj.GetId(), childIdx); err != nil {
			return err
		}
	}

	query = "delete from nodes_taints where nodes_Id = $1 AND idx >= $2"
	_, err = tx.Exec(ctx, query, obj.GetId(), len(obj.GetTaints()))
	if err != nil {
		return err
	}
	if !scanUpdated {
		return nil
	}

	components, vulns, nodeComponentEdges, componentCVEEdges := getPartsAsSlice(common.Split(obj, scanUpdated))
	if err := copyFromNodeComponents(ctx, tx, components...); err != nil {
		return err
	}
	if err := copyFromNodeComponentEdges(ctx, tx, nodeComponentEdges...); err != nil {
		return err
	}
	if err := copyFromNodeCves(ctx, tx, iTime, vulns...); err != nil {
		return err
	}
	return copyFromNodeComponentCVEEdges(ctx, tx, componentCVEEdges...)
}

func getPartsAsSlice(parts *common.NodeParts) ([]*storage.ImageComponent, []*storage.CVE, []*storage.NodeComponentEdge, []*storage.ComponentCVEEdge) {
	components := make([]*storage.ImageComponent, 0, len(parts.Children))
	nodeComponentEdges := make([]*storage.NodeComponentEdge, 0, len(parts.Children))
	vulnMap := make(map[string]*storage.CVE)
	var componentCVEEdges []*storage.ComponentCVEEdge
	for _, child := range parts.Children {
		components = append(components, child.Component)
		nodeComponentEdges = append(nodeComponentEdges, child.Edge)
		for _, gChild := range child.Children {
			componentCVEEdges = append(componentCVEEdges, gChild.Edge)
			vulnMap[gChild.CVE.GetId()] = gChild.CVE
		}
	}
	vulns := make([]*storage.CVE, 0, len(vulnMap))
	for _, vuln := range vulnMap {
		vulns = append(vulns, vuln)
	}
	return components, vulns, nodeComponentEdges, componentCVEEdges
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

func copyFromNodeComponents(ctx context.Context, tx pgx.Tx, objs ...*storage.ImageComponent) error {
	inputRows := [][]interface{}{}
	var err error
	var deletes []string
	copyCols := []string{
		"id",
		"name",
		"version",
		"source",
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
			obj.GetSource(),
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
	return err
}

func copyFromNodeComponentEdges(ctx context.Context, tx pgx.Tx, objs ...*storage.NodeComponentEdge) error {
	inputRows := [][]interface{}{}
	var err error
	copyCols := []string{
		"id",
		"nodeid",
		"nodecomponentid",
		"serialized",
	}

	if len(objs) == 0 {
		return nil
	}

	// Copy does not upsert so have to delete first.
	_, err = tx.Exec(ctx, "DELETE FROM "+nodeComponentEdgesTable+" WHERE nodeid = $1", objs[0].GetNodeId())
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
	return err
}

func copyFromNodeCves(ctx context.Context, tx pgx.Tx, iTime *protoTypes.Timestamp, objs ...*storage.CVE) error {
	inputRows := [][]interface{}{}

	var err error

	// This is a copy so first we must delete the rows and re-add them
	var deletes []string

	copyCols := []string{
		"id",
		"cve",
		"cvss",
		"impactscore",
		"publishedon",
		"createdat",
		"suppressed",
		"suppressexpiry",
		"severity",
		"serialized",
	}

	ids := set.NewStringSet()
	for _, obj := range objs {
		ids.Add(obj.GetId())
	}
	existingCVEs, err := getCVEs(ctx, tx, ids.AsSlice())

	for idx, obj := range objs {
		obj.Type = storage.CVE_NODE_CVE
		obj.Types = []storage.CVE_CVEType{storage.CVE_NODE_CVE}
		if storedCVE := existingCVEs[obj.GetId()]; storedCVE != nil {
			obj.Suppressed = storedCVE.GetSuppressed()
			obj.CreatedAt = storedCVE.GetCreatedAt()
			obj.SuppressActivation = storedCVE.GetSuppressActivation()
			obj.SuppressExpiry = storedCVE.GetSuppressExpiry()
		} else {
			obj.CreatedAt = iTime
		}

		serialized, marshalErr := obj.Marshal()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{
			obj.GetId(),
			obj.GetCve(),
			obj.GetCvss(),
			obj.GetImpactScore(),
			pgutils.NilOrTime(obj.GetPublishedOn()),
			pgutils.NilOrTime(obj.GetCreatedAt()),
			obj.GetSuppressed(),
			pgutils.NilOrTime(obj.GetSuppressExpiry()),
			obj.GetSeverity(),
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
	return err
}

func copyFromNodeComponentCVEEdges(ctx context.Context, tx pgx.Tx, objs ...*storage.ComponentCVEEdge) error {
	inputRows := [][]interface{}{}
	var err error
	componentIDsToDelete := set.NewStringSet()
	copyCols := []string{
		"id",
		"isfixable",
		"fixedby",
		"imagecomponentid",
		"imagecveid",
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
			obj.GetImageComponentId(),
			obj.GetImageCveId(),
			serialized,
		})

		// Add the id to be deleted.
		componentIDsToDelete.Add(obj.GetImageComponentId())

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// Copy does not upsert so have to delete first.
			_, err = tx.Exec(ctx, "DELETE FROM "+componentCVEEdgesTable+" WHERE imagecomponentid = ANY($1::text[])", componentIDsToDelete.AsSlice())
			if err != nil {
				return err
			}

			// Clear the inserts for the next batch
			componentIDsToDelete = nil

			_, err = tx.CopyFrom(ctx, pgx.Identifier{componentCVEEdgesTable}, copyCols, pgx.CopyFromRows(inputRows))
			if err != nil {
				return err
			}

			// Clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}
	return err
}

func (s *storeImpl) isUpdated(ctx context.Context, node *storage.Node) (bool, bool, error) {
	oldNode, found, err := s.Get(ctx, node.GetId())
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

func (s *storeImpl) upsert(ctx context.Context, objs ...*storage.Node) error {
	iTime := protoTypes.TimestampNow()
	conn, release, err := s.acquireConn(ctx, ops.Get, "Node")
	if err != nil {
		return err
	}
	defer release()

	for _, obj := range objs {
		tx, err := conn.Begin(ctx)
		if err != nil {
			return err
		}

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

		if err := insertIntoNodes(ctx, tx, obj, scanUpdated, iTime); err != nil {
			if err := tx.Rollback(ctx); err != nil {
				return err
			}
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Upsert upserts image into the store.
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

	conn, release, err := s.acquireConn(ctx, ops.Get, "Image")
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
			cveIDs.Add(edge.GetImageCveId())
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
				CVE:  cveMap[edge.GetImageCveId()],
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

func getNodeComponents(ctx context.Context, tx pgx.Tx, componentIDs []string) (map[string]*storage.ImageComponent, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "NodeComponent")

	rows, err := tx.Query(ctx, "SELECT serialized FROM "+nodeComponentsTable+" WHERE id = ANY($1::text[])", componentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	idToComponentMap := make(map[string]*storage.ImageComponent)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		msg := &storage.ImageComponent{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		idToComponentMap[msg.GetId()] = msg
	}
	return idToComponentMap, nil
}

func getComponentCVEEdges(ctx context.Context, tx pgx.Tx, componentIDs []string) (map[string][]*storage.ComponentCVEEdge, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "NodeComponentCVERelations")

	rows, err := tx.Query(ctx, "SELECT serialized FROM "+componentCVEEdgesTable+" WHERE imagecomponentid = ANY($1::text[])", componentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	componentIDToEdgesMap := make(map[string][]*storage.ComponentCVEEdge)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		msg := &storage.ComponentCVEEdge{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		componentIDToEdgesMap[msg.GetImageComponentId()] = append(componentIDToEdgesMap[msg.GetImageComponentId()], msg)
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

	return s.deleteNodeTree(ctx, conn, id)
}

func (s *storeImpl) deleteNodeTree(ctx context.Context, conn *pgxpool.Conn, nodeIDs ...string) error {
	// Delete nodes.
	if _, err := conn.Exec(ctx, "delete from "+nodesTable+" where Id = ANY($1::text[])", nodeIDs); err != nil {
		return err
	}
	// Node-components edges have ON DELETE CASCADE referential constraint on nodeid, therefore, no need to explicitly trigger deletion.

	// Get orphaned node components.
	rows, err := s.db.Query(ctx, "select id from "+nodeComponentsTable+" where not exists (select "+nodeComponentsTable+".id FROM "+nodeComponentsTable+", "+nodeComponentEdgesTable+" WHERE "+nodeComponentsTable+".id = "+nodeComponentEdgesTable+".nodecomponentid)")
	if err != nil {
		return pgutils.ErrNilIfNoRows(err)
	}
	defer rows.Close()
	var componentIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		componentIDs = append(componentIDs, id)
	}

	// Delete orphaned node components.
	if _, err := conn.Exec(ctx, "delete from "+nodeComponentsTable+" where id = ANY($1::text[])", componentIDs); err != nil {
		return err
	}

	// Component-CVE edges have ON DELETE CASCADE referential constraint on component id, therefore, no need to explicitly trigger deletion.

	// Delete orphaned cves.
	if _, err := conn.Exec(ctx, "delete from "+nodeCVEsTable+" where not exists (select "+nodeCVEsTable+".id FROM "+nodeCVEsTable+", "+componentCVEEdgesTable+" WHERE "+nodeCVEsTable+".id = "+componentCVEEdgesTable+".imagecveid)"); err != nil {
		return err
	}
	return nil
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

// GetNodeMetadata gets the image without scan/component data.
func (s *storeImpl) GetNodeMetadata(ctx context.Context, id string) (*storage.Node, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "NodeMetadata")

	conn, release, err := s.acquireConn(ctx, ops.Get, "Node")
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

// Delete removes the specified IDs from the store
func (s *storeImpl) DeleteMany(ctx context.Context, ids []string) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.RemoveMany, "Node")

	conn, release, err := s.acquireConn(ctx, ops.RemoveMany, "Node")
	if err != nil {
		return err
	}
	defer release()

	return s.deleteNodeTree(ctx, conn, ids...)
}

//// Used for testing

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

func getCVEs(ctx context.Context, tx pgx.Tx, cveIDs []string) (map[string]*storage.CVE, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "NodeCVEs")

	rows, err := tx.Query(ctx, "SELECT serialized FROM "+nodeCVEsTable+" WHERE id = ANY($1::text[])", cveIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	idToCVEMap := make(map[string]*storage.CVE)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		msg := &storage.CVE{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		idToCVEMap[msg.GetId()] = msg
	}
	return idToCVEMap, nil
}
