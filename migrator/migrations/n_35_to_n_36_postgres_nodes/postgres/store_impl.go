// This file was originally generated with
// //go:generate cp ../../../../central/node/datastore/internal/store/postgres/store.go store_impl.go

package postgres

import (
	"context"

	"github.com/gogo/protobuf/proto"
	protoTypes "github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v73"
	"github.com/stackrox/rox/migrator/migrations/loghelper"
	"github.com/stackrox/rox/migrator/migrations/n_35_to_n_36_postgres_nodes/common/v2"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
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
	batchSize = 500
)

var (
	log    = loghelper.LogWrapper{}
	schema = pkgSchema.NodesSchema
)

// Store provides storage functionality for full nodes.
type Store interface {
	Count(ctx context.Context) (int, error)
	Get(ctx context.Context, id string) (*storage.Node, bool, error)
	Upsert(ctx context.Context, obj *storage.Node) error
}

type storeImpl struct {
	db                 postgres.DB
	noUpdateTimestamps bool
}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB, noUpdateTimestamps bool) Store {
	return &storeImpl{
		db:                 db,
		noUpdateTimestamps: noUpdateTimestamps,
	}
}

func insertIntoNodes(ctx context.Context, tx *postgres.Tx, obj *storage.Node, scanUpdated bool, iTime *protoTypes.Timestamp) error {
	cloned := obj
	if cloned.GetScan().GetComponents() != nil {
		cloned = obj.Clone()
		cloned.Scan.Components = nil
	}
	serialized, marshalErr := cloned.Marshal()
	if marshalErr != nil {
		return marshalErr
	}

	if pgutils.NilOrUUID(cloned.GetId()) == nil {
		log.WriteToStderrf("id is not a valid uuid -- %q", cloned.GetId())
		return nil
	}

	values := []interface{}{
		// parent primary keys start
		pgutils.NilOrUUID(cloned.GetId()),
		cloned.GetName(),
		pgutils.NilOrUUID(cloned.GetClusterId()),
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

	for childIdx, child := range obj.GetTaints() {
		if err := insertIntoNodesTaints(ctx, tx, child, obj.GetId(), childIdx); err != nil {
			return err
		}
	}

	query = "delete from nodes_taints where nodes_Id = $1 AND idx >= $2"
	_, err = tx.Exec(ctx, query, pgutils.NilOrUUID(cloned.GetId()), len(obj.GetTaints()))
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

func getPartsAsSlice(parts *common.NodeParts) ([]*storage.NodeComponent, []*storage.NodeCVE, []*storage.NodeComponentEdge, []*storage.NodeComponentCVEEdge) {
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
	return components, vulns, nodeComponentEdges, componentCVEEdges
}

func insertIntoNodesTaints(ctx context.Context, tx *postgres.Tx, obj *storage.Taint, nodeID string, idx int) error {
	if pgutils.NilOrUUID(nodeID) == nil {
		log.WriteToStderrf("id is not a valid uuid -- %q", nodeID)
		return nil
	}

	values := []interface{}{
		// parent primary keys start
		pgutils.NilOrUUID(nodeID),
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

func copyFromNodeComponents(ctx context.Context, tx *postgres.Tx, objs ...*storage.NodeComponent) error {
	inputRows := [][]interface{}{}
	var err error
	var deletes []string
	copyCols := []string{
		"id",
		"name",
		"version",
		"operatingsystem",
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
			obj.GetOperatingSystem(),
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
	return err
}

func copyFromNodeComponentEdges(ctx context.Context, tx *postgres.Tx, objs ...*storage.NodeComponentEdge) error {
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
	_, err = tx.Exec(ctx, "DELETE FROM "+nodeComponentEdgesTable+" WHERE nodeid = $1", pgutils.NilOrUUID(objs[0].GetNodeId()))
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
			pgutils.NilOrUUID(obj.GetNodeId()),
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

func copyFromNodeCves(ctx context.Context, tx *postgres.Tx, iTime *protoTypes.Timestamp, objs ...*storage.NodeCVE) error {
	inputRows := [][]interface{}{}

	var err error

	// This is a copy so first we must delete the rows and re-add them
	var deletes []string

	copyCols := []string{
		"id",
		"cvebaseinfo_cve",
		"cvebaseinfo_publishedon",
		"cvebaseinfo_createdat",
		"operatingsystem",
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
			obj.GetOperatingSystem(),
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
	return err
}

func copyFromNodeComponentCVEEdges(ctx context.Context, tx *postgres.Tx, objs ...*storage.NodeComponentCVEEdge) error {
	inputRows := [][]interface{}{}
	var err error
	componentIDsToDelete := set.NewStringSet()
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
		componentIDsToDelete.Add(obj.GetNodeComponentId())

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// Copy does not upsert so have to delete first.
			_, err = tx.Exec(ctx, "DELETE FROM "+componentCVEEdgesTable+" WHERE nodecomponentid = ANY($1::text[])", componentIDsToDelete.AsSlice())
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

func (s *storeImpl) upsert(ctx context.Context, objs ...*storage.Node) error {
	iTime := protoTypes.TimestampNow()
	conn, release, err := s.acquireConn(ctx)
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
		} else {
			iTime = obj.LastUpdated
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

// Upsert upserts node into the store.
func (s *storeImpl) Upsert(ctx context.Context, obj *storage.Node) error {
	return pgutils.Retry(func() error {
		return s.upsert(ctx, obj)
	})
}

func (s *storeImpl) copyFromNodesTaints(ctx context.Context, tx *postgres.Tx, nodeID string, objs ...*storage.Taint) error {
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

		if pgutils.NilOrUUID(nodeID) == nil {
			log.WriteToStderrf("id is not a valid uuid -- %q", nodeID)
			continue
		}

		inputRows = append(inputRows, []interface{}{
			pgutils.NilOrUUID(nodeID),
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
	return pgutils.Retry2(func() (int, error) {
		return s.retryableCount(ctx)
	})
}

func (s *storeImpl) retryableCount(ctx context.Context) (int, error) {
	var sacQueryFilter *v1.Query
	return pgSearch.RunCountRequestForSchema(ctx, schema, sacQueryFilter, s.db)
}

// Get returns the object, if it exists from the store
func (s *storeImpl) Get(ctx context.Context, id string) (*storage.Node, bool, error) {
	return pgutils.Retry3(func() (*storage.Node, bool, error) {
		return s.retryableGet(ctx, id)
	})
}

func (s *storeImpl) retryableGet(ctx context.Context, id string) (*storage.Node, bool, error) {
	conn, release, err := s.acquireConn(ctx)
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

func (s *storeImpl) getFullNode(ctx context.Context, tx *postgres.Tx, nodeID string) (*storage.Node, bool, error) {
	row := tx.QueryRow(ctx, getNodeMetaStmt, pgutils.NilOrUUID(nodeID))
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

func getNodeComponentEdges(ctx context.Context, tx *postgres.Tx, nodeID string) (map[string]*storage.NodeComponentEdge, error) {
	rows, err := tx.Query(ctx, "SELECT serialized FROM "+nodeComponentEdgesTable+" WHERE nodeid = $1", pgutils.NilOrUUID(nodeID))
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
	return componentIDToEdgeMap, rows.Err()
}

func getNodeComponents(ctx context.Context, tx *postgres.Tx, componentIDs []string) (map[string]*storage.NodeComponent, error) {
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
	return idToComponentMap, rows.Err()
}

func getComponentCVEEdges(ctx context.Context, tx *postgres.Tx, componentIDs []string) (map[string][]*storage.NodeComponentCVEEdge, error) {
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
	return componentIDToEdgesMap, rows.Err()
}

func (s *storeImpl) acquireConn(ctx context.Context) (*postgres.Conn, func(), error) {
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	return conn, conn.Release, nil
}

// GetNodeMetadata gets the node without scan/component data.
func (s *storeImpl) GetNodeMetadata(ctx context.Context, id string) (*storage.Node, bool, error) {
	conn, release, err := s.acquireConn(ctx)
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

func getCVEs(ctx context.Context, tx *postgres.Tx, cveIDs []string) (map[string]*storage.NodeCVE, error) {
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
	return idToCVEMap, rows.Err()
}
