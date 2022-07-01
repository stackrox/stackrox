package postgres

import (
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	protoTypes "github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/image/datastore/store"
	"github.com/stackrox/rox/central/image/datastore/store/common/v2"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"gorm.io/gorm"
)

const (
	imagesTable              = pkgSchema.ImagesTableName
	imageComponentEdgesTable = pkgSchema.ImageComponentEdgesTableName
	imageComponentsTable     = pkgSchema.ImageComponentsTableName
	componentCVEEdgesTable   = pkgSchema.ImageComponentCveEdgesTableName
	imageCVEsTable           = pkgSchema.ImageCvesTableName
	imageCVEEdgesTable       = pkgSchema.ImageCveEdgesTableName

	countStmt  = "SELECT COUNT(*) FROM " + imagesTable
	existsStmt = "SELECT EXISTS(SELECT 1 FROM " + imagesTable + " WHERE Id = $1)"

	getImageMetaStmt = "SELECT serialized FROM " + imagesTable + " WHERE Id = $1"
	getImageIDsStmt  = "SELECT Id FROM " + imagesTable

	// using copyFrom, we may not even want to batch.  It would probably be simpler
	// to deal with failures if we just sent it all.  Something to think about as we
	// proceed and move into more e2e and larger performance testing
	batchSize = 10000
)

var (
	log    = logging.LoggerForModule()
	schema = pkgSchema.ImagesSchema
)

// New returns a new Store instance using the provided sql instance.
func New(db *pgxpool.Pool, noUpdateTimestamps bool) store.Store {
	return &storeImpl{
		db:                 db,
		noUpdateTimestamps: noUpdateTimestamps,
	}
}

type storeImpl struct {
	db                 *pgxpool.Pool
	noUpdateTimestamps bool
}

func insertIntoImages(ctx context.Context, tx pgx.Tx, obj *storage.Image, scanUpdated bool, iTime *protoTypes.Timestamp) error {
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
		obj.GetName().GetRegistry(),
		obj.GetName().GetRemote(),
		obj.GetName().GetTag(),
		obj.GetName().GetFullName(),
		pgutils.NilOrTime(obj.GetMetadata().GetV1().GetCreated()),
		obj.GetMetadata().GetV1().GetUser(),
		obj.GetMetadata().GetV1().GetCommand(),
		obj.GetMetadata().GetV1().GetEntrypoint(),
		obj.GetMetadata().GetV1().GetVolumes(),
		obj.GetMetadata().GetV1().GetLabels(),
		pgutils.NilOrTime(obj.GetScan().GetScanTime()),
		obj.GetScan().GetOperatingSystem(),
		pgutils.NilOrTime(obj.GetSignature().GetFetched()),
		obj.GetComponents(),
		obj.GetCves(),
		obj.GetFixableCves(),
		pgutils.NilOrTime(obj.GetLastUpdated()),
		obj.GetRiskScore(),
		obj.GetTopCvss(),
		serialized,
	}

	finalStr := "INSERT INTO " + imagesTable + " (Id, Name_Registry, Name_Remote, Name_Tag, Name_FullName, Metadata_V1_Created, Metadata_V1_User, Metadata_V1_Command, Metadata_V1_Entrypoint, Metadata_V1_Volumes, Metadata_V1_Labels, Scan_ScanTime, Scan_OperatingSystem, Signature_Fetched, Components, Cves, FixableCves, LastUpdated, RiskScore, TopCvss, serialized) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, Name_Registry = EXCLUDED.Name_Registry, Name_Remote = EXCLUDED.Name_Remote, Name_Tag = EXCLUDED.Name_Tag, Name_FullName = EXCLUDED.Name_FullName, Metadata_V1_Created = EXCLUDED.Metadata_V1_Created, Metadata_V1_User = EXCLUDED.Metadata_V1_User, Metadata_V1_Command = EXCLUDED.Metadata_V1_Command, Metadata_V1_Entrypoint = EXCLUDED.Metadata_V1_Entrypoint, Metadata_V1_Volumes = EXCLUDED.Metadata_V1_Volumes, Metadata_V1_Labels = EXCLUDED.Metadata_V1_Labels, Scan_ScanTime = EXCLUDED.Scan_ScanTime, Scan_OperatingSystem = EXCLUDED.Scan_OperatingSystem, Signature_Fetched = EXCLUDED.Signature_Fetched, Components = EXCLUDED.Components, Cves = EXCLUDED.Cves, FixableCves = EXCLUDED.FixableCves, LastUpdated = EXCLUDED.LastUpdated, RiskScore = EXCLUDED.RiskScore, TopCvss = EXCLUDED.TopCvss, serialized = EXCLUDED.serialized"
	_, err := tx.Exec(ctx, finalStr, values...)
	if err != nil {
		return err
	}

	var query string

	for childIdx, child := range obj.GetMetadata().GetV1().GetLayers() {
		if err := insertIntoImagesLayers(ctx, tx, child, obj.GetId(), childIdx); err != nil {
			return err
		}
	}

	query = "delete from images_Layers where images_Id = $1 AND idx >= $2"
	_, err = tx.Exec(ctx, query, obj.GetId(), len(obj.GetMetadata().GetV1().GetLayers()))
	if err != nil {
		return err
	}

	if !scanUpdated {
		return nil
	}

	components, vulns, imageComponentEdges, componentCVEEdges, imageCVEEdges := getPartsAsSlice(common.Split(obj, scanUpdated))
	if err := copyFromImageComponents(ctx, tx, components...); err != nil {
		return err
	}
	if err := copyFromImageComponentEdges(ctx, tx, imageComponentEdges...); err != nil {
		return err
	}
	if err := copyFromImageCves(ctx, tx, iTime, vulns...); err != nil {
		return err
	}
	if err := copyFromImageComponentCVEEdges(ctx, tx, obj.GetScan().GetOperatingSystem(), componentCVEEdges...); err != nil {
		return err
	}
	return copyFromImageCVEEdges(ctx, tx, iTime, imageCVEEdges...)
}

func getPartsAsSlice(parts common.ImageParts) ([]*storage.ImageComponent, []*storage.ImageCVE, []*storage.ImageComponentEdge, []*storage.ComponentCVEEdge, []*storage.ImageCVEEdge) {
	components := make([]*storage.ImageComponent, 0, len(parts.Children))
	imageComponentEdges := make([]*storage.ImageComponentEdge, 0, len(parts.Children))
	vulnMap := make(map[string]*storage.ImageCVE)
	var componentCVEEdges []*storage.ComponentCVEEdge
	for _, child := range parts.Children {
		components = append(components, child.Component)
		imageComponentEdges = append(imageComponentEdges, child.Edge)
		for _, gChild := range child.Children {
			componentCVEEdges = append(componentCVEEdges, gChild.Edge)
			vulnMap[gChild.CVE.GetId()] = gChild.CVE
		}
	}
	vulns := make([]*storage.ImageCVE, 0, len(vulnMap))
	for _, vuln := range vulnMap {
		vulns = append(vulns, vuln)
	}
	imageCVEEdges := make([]*storage.ImageCVEEdge, 0, len(parts.ImageCVEEdges))
	for _, imageCVEEdge := range parts.ImageCVEEdges {
		imageCVEEdges = append(imageCVEEdges, imageCVEEdge)
	}
	return components, vulns, imageComponentEdges, componentCVEEdges, imageCVEEdges
}

func insertIntoImagesLayers(ctx context.Context, tx pgx.Tx, obj *storage.ImageLayer, imageID string, idx int) error {

	values := []interface{}{
		// parent primary keys start
		imageID,
		idx,
		obj.GetInstruction(),
		obj.GetValue(),
	}

	finalStr := "INSERT INTO images_Layers (images_Id, idx, Instruction, Value) VALUES($1, $2, $3, $4) ON CONFLICT(images_Id, idx) DO UPDATE SET images_Id = EXCLUDED.images_Id, idx = EXCLUDED.idx, Instruction = EXCLUDED.Instruction, Value = EXCLUDED.Value"
	_, err := tx.Exec(ctx, finalStr, values...)
	if err != nil {
		return err
	}

	return nil
}

func copyFromImageComponents(ctx context.Context, tx pgx.Tx, objs ...*storage.ImageComponent) error {
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
			_, err = tx.Exec(ctx, "DELETE FROM "+imageComponentsTable+" WHERE id = ANY($1::text[])", deletes)
			if err != nil {
				return err
			}

			// clear the inserts for the next batch
			deletes = nil

			_, err = tx.CopyFrom(ctx, pgx.Identifier{imageComponentsTable}, copyCols, pgx.CopyFromRows(inputRows))

			if err != nil {
				return err
			}

			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return err
}

func copyFromImageComponentEdges(ctx context.Context, tx pgx.Tx, objs ...*storage.ImageComponentEdge) error {
	inputRows := [][]interface{}{}

	var err error

	copyCols := []string{
		"id",
		"location",
		"imageid",
		"imagecomponentid",
		"serialized",
	}

	if len(objs) == 0 {
		return nil
	}

	// Copy does not upsert so have to delete first.
	_, err = tx.Exec(ctx, "DELETE FROM "+imageComponentEdgesTable+" WHERE imageid = $1", objs[0].GetImageId())
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
			obj.GetLocation(),
			obj.GetImageId(),
			obj.GetImageComponentId(),
			serialized,
		})

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			_, err = tx.CopyFrom(ctx, pgx.Identifier{imageComponentEdgesTable}, copyCols, pgx.CopyFromRows(inputRows))

			if err != nil {
				return err
			}

			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return err
}

func copyFromImageCves(ctx context.Context, tx pgx.Tx, iTime *protoTypes.Timestamp, objs ...*storage.ImageCVE) error {
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
			obj.CveBaseInfo.CreatedAt = storedCVE.GetCveBaseInfo().GetCreatedAt()
			obj.Snoozed = storedCVE.GetSnoozed()
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
			obj.GetSnoozed(),
			pgutils.NilOrTime(obj.GetSnoozeExpiry()),
			serialized,
		})

		// Add the id to be deleted.
		deletes = append(deletes, obj.GetId())

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// Copy does not upsert so have to delete first.
			_, err = tx.Exec(ctx, "DELETE FROM "+imageCVEsTable+" WHERE id = ANY($1::text[])", deletes)
			if err != nil {
				return err
			}
			// Clear the inserts for the next batch.
			deletes = nil

			_, err = tx.CopyFrom(ctx, pgx.Identifier{imageCVEsTable}, copyCols, pgx.CopyFromRows(inputRows))

			if err != nil {
				return err
			}

			// Clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return err
}

func copyFromImageComponentCVEEdges(ctx context.Context, tx pgx.Tx, os string, objs ...*storage.ComponentCVEEdge) error {
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

func copyFromImageCVEEdges(ctx context.Context, tx pgx.Tx, iTime *protoTypes.Timestamp, objs ...*storage.ImageCVEEdge) error {
	inputRows := [][]interface{}{}

	var err error

	copyCols := []string{
		"id",
		"firstimageoccurrence",
		"state",
		"imageid",
		"imagecveid",
		"serialized",
	}

	if len(objs) == 0 {
		return nil
	}

	ids := set.NewStringSet()
	for _, obj := range objs {
		ids.Add(obj.GetId())
	}

	// Remove orphaned edges.
	if err := removeOrphanedImageCVEEdges(ctx, tx, objs[0].GetImageId(), ids.AsSlice()); err != nil {
		return err
	}

	exisitingEdgeIDs, err := getImageCVEEdgeIDs(ctx, tx, objs[0].GetImageId())
	if err != nil {
		return err
	}

	for idx, obj := range objs {
		if exisitingEdgeIDs.Contains(obj.GetId()) {
			continue
		}

		obj.FirstImageOccurrence = iTime
		serialized, marshalErr := obj.Marshal()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{
			obj.GetId(),
			pgutils.NilOrTime(obj.GetFirstImageOccurrence()),
			obj.GetState(),
			obj.GetImageId(),
			obj.GetImageCveId(),
			serialized,
		})

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			_, err = tx.CopyFrom(ctx, pgx.Identifier{imageCVEEdgesTable}, copyCols, pgx.CopyFromRows(inputRows))
			if err != nil {
				return err
			}

			// Clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}
	return err
}

func removeOrphanedImageCVEEdges(ctx context.Context, tx pgx.Tx, imageID string, ids []string) error {
	_, err := tx.Exec(ctx, "DELETE FROM "+imageCVEEdgesTable+" WHERE id in (select id from "+imageCVEEdgesTable+" where imageid = $1 and id != ANY($2::text[]))", imageID, ids)
	if err != nil {
		return err
	}
	return nil
}

func (s *storeImpl) isUpdated(ctx context.Context, image *storage.Image) (bool, bool, error) {
	oldImage, found, err := s.Get(ctx, image.GetId())
	if err != nil {
		return false, false, err
	}
	if !found {
		return true, true, nil
	}

	metadataUpdated := false
	scanUpdated := false
	if oldImage.GetMetadata().GetV1().GetCreated().Compare(image.GetMetadata().GetV1().GetCreated()) > 0 {
		image.Metadata = oldImage.GetMetadata()
	} else {
		metadataUpdated = true
	}

	// We skip rewriting components and cves if scan is not newer, hence we do not need to merge.
	if oldImage.GetScan().GetScanTime().Compare(image.GetScan().GetScanTime()) > 0 {
		image.Scan = oldImage.Scan
	} else {
		scanUpdated = true
	}

	// If the image in the DB is latest, then use its risk score and scan stats
	if !scanUpdated {
		image.RiskScore = oldImage.GetRiskScore()
		image.SetComponents = oldImage.GetSetComponents()
		image.SetCves = oldImage.GetSetCves()
		image.SetFixable = oldImage.GetSetFixable()
		image.SetTopCvss = oldImage.GetSetTopCvss()
	}
	return metadataUpdated, scanUpdated, nil
}

func (s *storeImpl) upsert(ctx context.Context, objs ...*storage.Image) error {
	iTime := protoTypes.TimestampNow()
	conn, release, err := s.acquireConn(ctx, ops.Get, "Image")
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

		if err := insertIntoImages(ctx, tx, obj, scanUpdated, iTime); err != nil {
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
func (s *storeImpl) Upsert(ctx context.Context, obj *storage.Image) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Upsert, "Image")

	return s.upsert(ctx, obj)
}

// Count returns the number of objects in the store
func (s *storeImpl) Count(ctx context.Context) (int, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Count, "Image")

	row := s.db.QueryRow(ctx, countStmt)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// Exists returns if the id exists in the store
func (s *storeImpl) Exists(ctx context.Context, id string) (bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Exists, "Image")

	row := s.db.QueryRow(ctx, existsStmt, id)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, pgutils.ErrNilIfNoRows(err)
	}
	return exists, nil
}

// Get returns the object, if it exists from the store.
func (s *storeImpl) Get(ctx context.Context, id string) (*storage.Image, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "Image")

	conn, release, err := s.acquireConn(ctx, ops.Get, "Image")
	if err != nil {
		return nil, false, err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, false, err
	}
	return s.getFullImage(ctx, tx, id)
}

func (s *storeImpl) getFullImage(ctx context.Context, tx pgx.Tx, imageID string) (*storage.Image, bool, error) {
	row := tx.QueryRow(ctx, getImageMetaStmt, imageID)
	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}

	var image storage.Image
	if err := proto.Unmarshal(data, &image); err != nil {
		return nil, false, err
	}

	imageCVEEdgeMap, err := getImageCVEEdges(ctx, tx, imageID)
	if err != nil {
		return nil, false, err
	}
	cveIDs := make([]string, 0, len(imageCVEEdgeMap))
	for _, val := range imageCVEEdgeMap {
		cveIDs = append(cveIDs, val.GetImageCveId())
	}

	componentEdgeMap, err := getImageComponentEdges(ctx, tx, imageID)
	if err != nil {
		return nil, false, err
	}
	componentIDs := make([]string, 0, len(componentEdgeMap))
	for _, val := range componentEdgeMap {
		componentIDs = append(componentIDs, val.GetImageComponentId())
	}

	componentMap, err := getImageComponents(ctx, tx, componentIDs)
	if err != nil {
		return nil, false, err
	}

	componentCVEEdgeMap, err := getComponentCVEEdges(ctx, tx, componentIDs)
	if err != nil {
		return nil, false, err
	}

	cveMap, err := getCVEs(ctx, tx, cveIDs)
	if err != nil {
		return nil, false, err
	}

	if len(componentEdgeMap) != len(componentMap) {
		utils.Should(
			errors.Errorf("Number of component (%d) in image-component edges is not equal to number of stored components (%d) for image %s (imageID=%s)",
				len(componentEdgeMap), len(componentMap), image.GetName().GetFullName(), image.GetId()),
		)
	}

	imageParts := common.ImageParts{
		Image:         &image,
		Children:      []common.ComponentParts{},
		ImageCVEEdges: imageCVEEdgeMap,
	}
	for componentID, component := range componentMap {
		child := common.ComponentParts{
			Edge:      componentEdgeMap[componentID],
			Component: component,
			Children:  []common.CVEParts{},
		}

		for _, edge := range componentCVEEdgeMap[componentID] {
			child.Children = append(child.Children, common.CVEParts{
				Edge: edge,
				CVE:  cveMap[edge.GetImageCveId()],
			})
		}
		imageParts.Children = append(imageParts.Children, child)
	}
	return common.Merge(imageParts), true, nil
}

func (s *storeImpl) acquireConn(ctx context.Context, op ops.Op, typ string) (*pgxpool.Conn, func(), error) {
	defer metrics.SetAcquireDBConnDuration(time.Now(), op, typ)
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	return conn, conn.Release, nil
}

func getImageComponentEdges(ctx context.Context, tx pgx.Tx, imageID string) (map[string]*storage.ImageComponentEdge, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "ImageComponentEdges")

	rows, err := tx.Query(ctx, "SELECT serialized FROM "+imageComponentEdgesTable+" WHERE imageid = $1", imageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	componentIDToEdgeMap := make(map[string]*storage.ImageComponentEdge)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		msg := &storage.ImageComponentEdge{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		componentIDToEdgeMap[msg.GetImageComponentId()] = msg
	}
	return componentIDToEdgeMap, nil
}

func getImageCVEEdgeIDs(ctx context.Context, tx pgx.Tx, imageID string) (set.StringSet, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "ImageCVEEdges")

	rows, err := tx.Query(ctx, "SELECT id FROM "+imageCVEEdgesTable+" WHERE imageid = $1", imageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := set.NewStringSet()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids.Add(id)
	}
	return ids, nil
}

func getImageCVEEdges(ctx context.Context, tx pgx.Tx, imageID string) (map[string]*storage.ImageCVEEdge, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "ImageCVEEdges")

	rows, err := tx.Query(ctx, "SELECT serialized FROM "+imageCVEEdgesTable+" WHERE imageid = $1", imageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cveIDToEdgeMap := make(map[string]*storage.ImageCVEEdge)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		msg := &storage.ImageCVEEdge{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		cveIDToEdgeMap[msg.GetImageCveId()] = msg
	}
	return cveIDToEdgeMap, nil
}

func getImageComponents(ctx context.Context, tx pgx.Tx, componentIDs []string) (map[string]*storage.ImageComponent, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "ImageComponents")

	rows, err := tx.Query(ctx, "SELECT serialized FROM "+imageComponentsTable+" WHERE id = ANY($1::text[])", componentIDs)
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
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "ImageComponentCVEEdges")

	rows, err := tx.Query(ctx, "SELECT serialized FROM "+componentCVEEdgesTable+" WHERE imagecomponentid = ANY($1::text[])", componentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	componentIDToEdgeMap := make(map[string][]*storage.ComponentCVEEdge)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		msg := &storage.ComponentCVEEdge{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		componentIDToEdgeMap[msg.GetImageComponentId()] = append(componentIDToEdgeMap[msg.GetImageComponentId()], msg)
	}
	return componentIDToEdgeMap, nil
}

func getCVEs(ctx context.Context, tx pgx.Tx, cveIDs []string) (map[string]*storage.ImageCVE, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "ImageCVEs")

	rows, err := tx.Query(ctx, "SELECT serialized FROM "+imageCVEsTable+" WHERE id = ANY($1::text[])", cveIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	idToCVEMap := make(map[string]*storage.ImageCVE)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		msg := &storage.ImageCVE{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, err
		}
		idToCVEMap[msg.GetId()] = msg
	}
	return idToCVEMap, nil
}

// Delete removes the specified ID from the store.
func (s *storeImpl) Delete(ctx context.Context, id string) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "Image")

	conn, release, err := s.acquireConn(ctx, ops.Remove, "Image")
	if err != nil {
		return err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	return s.deleteImageTree(ctx, tx, id)
}

func (s *storeImpl) deleteImageTree(ctx context.Context, tx pgx.Tx, imageID string) error {
	// Delete from image table.
	if _, err := tx.Exec(ctx, "delete from "+imagesTable+" where Id = $1", imageID); err != nil {
		return err
	}

	// Delete orphaned image components.
	if _, err := tx.Exec(ctx, "delete from "+imageComponentsTable+" where not exists (select "+imageComponentEdgesTable+".imagecomponentid FROM "+imageComponentEdgesTable+")"); err != nil {
		return err
	}

	// Delete orphaned cves.
	if _, err := tx.Exec(ctx, "delete from "+imageCVEsTable+" where not exists (select "+componentCVEEdgesTable+".imagecveid FROM "+componentCVEEdgesTable+")"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// GetIDs returns all the IDs for the store
func (s *storeImpl) GetIDs(ctx context.Context) ([]string, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetAll, "ImageIDs")

	rows, err := s.db.Query(ctx, getImageIDsStmt)
	if err != nil {
		return nil, pgutils.ErrNilIfNoRows(err)
	}
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

// GetMany returns the objects specified by the IDs or the index in the missing indices slice
func (s *storeImpl) GetMany(ctx context.Context, ids []string) ([]*storage.Image, []int, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "Image")

	conn, release, err := s.acquireConn(ctx, ops.GetMany, "Image")
	if err != nil {
		return nil, nil, err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}

	resultsByID := make(map[string]*storage.Image)
	for _, id := range ids {
		msg, found, err := s.getFullImage(ctx, tx, id)
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
	elems := make([]*storage.Image, 0, len(resultsByID))
	for i, id := range ids {
		if result, ok := resultsByID[id]; !ok {
			missingIndices = append(missingIndices, i)
		} else {
			elems = append(elems, result)
		}
	}
	return elems, missingIndices, nil
}

//// Used for testing

func dropTableImages(ctx context.Context, db *pgxpool.Pool) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS images CASCADE")
	dropTableImagesLayers(ctx, db)
	dropTableImageComponents(ctx, db)
	dropTableImageCVEs(ctx, db)
	dropTableImageCVEEdges(ctx, db)
	dropTableComponentCVEEdges(ctx, db)
	dropTableImageComponentEdges(ctx, db)
}

func dropTableImagesLayers(ctx context.Context, db *pgxpool.Pool) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS images_Layers CASCADE")
}

func dropTableImageComponents(ctx context.Context, db *pgxpool.Pool) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS "+imageComponentsTable+" CASCADE")
}

func dropTableImageCVEs(ctx context.Context, db *pgxpool.Pool) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS "+imageCVEsTable+" CASCADE")
}

func dropTableImageCVEEdges(ctx context.Context, db *pgxpool.Pool) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS "+imageCVEEdgesTable+" CASCADE")
}

func dropTableComponentCVEEdges(ctx context.Context, db *pgxpool.Pool) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS "+componentCVEEdgesTable+" CASCADE")
}

func dropTableImageComponentEdges(ctx context.Context, db *pgxpool.Pool) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS "+imageComponentEdgesTable+" CASCADE")
}

// Destroy drops image table.
func Destroy(ctx context.Context, db *pgxpool.Pool) {
	dropTableImages(ctx, db)
}

// CreateTableAndNewStore returns a new Store instance for testing
func CreateTableAndNewStore(ctx context.Context, db *pgxpool.Pool, gormDB *gorm.DB, noUpdateTimestamps bool) store.Store {
	pgutils.CreateTableFromModel(ctx, gormDB, pkgSchema.CreateTableImagesStmt)
	pgutils.CreateTableFromModel(ctx, gormDB, pkgSchema.CreateTableImageComponentsStmt)
	pgutils.CreateTableFromModel(ctx, gormDB, pkgSchema.CreateTableImageCvesStmt)
	pgutils.CreateTableFromModel(ctx, gormDB, pkgSchema.CreateTableImageComponentEdgesStmt)
	pgutils.CreateTableFromModel(ctx, gormDB, pkgSchema.CreateTableImageComponentCveEdgesStmt)
	pgutils.CreateTableFromModel(ctx, gormDB, pkgSchema.CreateTableImageCveEdgesStmt)
	return New(db, noUpdateTimestamps)
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

// GetImageMetadata gets the image without scan/component data.
func (s *storeImpl) GetImageMetadata(ctx context.Context, id string) (*storage.Image, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "ImageMetadata")

	conn, release, err := s.acquireConn(ctx, ops.Get, "Image")
	if err != nil {
		return nil, false, err
	}
	defer release()

	row := conn.QueryRow(ctx, getImageMetaStmt, id)
	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}

	var msg storage.Image
	if err := proto.Unmarshal(data, &msg); err != nil {
		return nil, false, err
	}
	return &msg, true, nil
}

func (s *storeImpl) UpdateVulnState(ctx context.Context, cve string, imageIDs []string, state storage.VulnerabilityState) error {
	conn, release, err := s.acquireConn(ctx, ops.Get, "UpdateVulnState")
	if err != nil {
		return err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	cveIDs, err := func() ([]string, error) {
		rows, err := s.db.Query(ctx, "select id from "+imageCVEsTable+" where cvebaseinfo_cve = $1", cve)
		if err != nil {
			return nil, pgutils.ErrNilIfNoRows(err)
		}

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
	}()
	if err != nil {
		return err
	}

	if len(imageIDs) == 0 || len(cveIDs) == 0 {
		return nil
	}

	query := "update " + imageCVEEdgesTable + " set state = $1 where imagecveid = ANY($2::text[]) AND imageid = ANY($3::text[])"
	_, err = tx.Exec(ctx, query, state, cveIDs, imageIDs)
	if err != nil {
		if err := tx.Rollback(ctx); err != nil {
			return err
		}
		return err
	}
	return tx.Commit(ctx)
}
