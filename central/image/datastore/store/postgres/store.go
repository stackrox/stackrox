package postgres

import (
	"context"
	"time"

	protoTypes "github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v5"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/central/image/datastore/store"
	"github.com/stackrox/rox/central/image/datastore/store/common/v2"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/set"
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
	batchSize = 500
)

var (
	log            = logging.LoggerForModule()
	schema         = pkgSchema.ImagesSchema
	targetResource = resources.Image
)

type imagePartsAsSlice struct {
	image               *storage.Image
	components          []*storage.ImageComponent
	vulns               []*storage.ImageCVE
	imageComponentEdges []*storage.ImageComponentEdge
	componentCVEEdges   []*storage.ComponentCVEEdge
	imageCVEEdges       []*storage.ImageCVEEdge
}

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB, noUpdateTimestamps bool, keyFence concurrency.KeyFence) store.Store {
	return &storeImpl{
		db:                 db,
		noUpdateTimestamps: noUpdateTimestamps,
		keyFence:           keyFence,
	}
}

type storeImpl struct {
	db                 postgres.DB
	noUpdateTimestamps bool
	keyFence           concurrency.KeyFence
}

func (s *storeImpl) insertIntoImages(
	ctx context.Context,
	tx *postgres.Tx, parts *imagePartsAsSlice,
	metadataUpdated, scanUpdated bool,
	iTime *protoTypes.Timestamp,
) error {
	cloned := parts.image
	if cloned.GetScan().GetComponents() != nil {
		cloned = parts.image.Clone()
		cloned.Scan.Components = nil
	}
	serialized, marshalErr := cloned.Marshal()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		// parent primary keys start
		cloned.GetId(),
		cloned.GetName().GetRegistry(),
		cloned.GetName().GetRemote(),
		cloned.GetName().GetTag(),
		cloned.GetName().GetFullName(),
		pgutils.NilOrTime(cloned.GetMetadata().GetV1().GetCreated()),
		cloned.GetMetadata().GetV1().GetUser(),
		cloned.GetMetadata().GetV1().GetCommand(),
		cloned.GetMetadata().GetV1().GetEntrypoint(),
		cloned.GetMetadata().GetV1().GetVolumes(),
		cloned.GetMetadata().GetV1().GetLabels(),
		pgutils.NilOrTime(cloned.GetScan().GetScanTime()),
		cloned.GetScan().GetOperatingSystem(),
		pgutils.NilOrTime(cloned.GetSignature().GetFetched()),
		cloned.GetComponents(),
		cloned.GetCves(),
		cloned.GetFixableCves(),
		pgutils.NilOrTime(cloned.GetLastUpdated()),
		cloned.GetPriority(),
		cloned.GetRiskScore(),
		cloned.GetTopCvss(),
		serialized,
	}

	finalStr := "INSERT INTO " + imagesTable + " (Id, Name_Registry, Name_Remote, Name_Tag, Name_FullName, Metadata_V1_Created, Metadata_V1_User, Metadata_V1_Command, Metadata_V1_Entrypoint, Metadata_V1_Volumes, Metadata_V1_Labels, Scan_ScanTime, Scan_OperatingSystem, Signature_Fetched, Components, Cves, FixableCves, LastUpdated, Priority, RiskScore, TopCvss, serialized) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, Name_Registry = EXCLUDED.Name_Registry, Name_Remote = EXCLUDED.Name_Remote, Name_Tag = EXCLUDED.Name_Tag, Name_FullName = EXCLUDED.Name_FullName, Metadata_V1_Created = EXCLUDED.Metadata_V1_Created, Metadata_V1_User = EXCLUDED.Metadata_V1_User, Metadata_V1_Command = EXCLUDED.Metadata_V1_Command, Metadata_V1_Entrypoint = EXCLUDED.Metadata_V1_Entrypoint, Metadata_V1_Volumes = EXCLUDED.Metadata_V1_Volumes, Metadata_V1_Labels = EXCLUDED.Metadata_V1_Labels, Scan_ScanTime = EXCLUDED.Scan_ScanTime, Scan_OperatingSystem = EXCLUDED.Scan_OperatingSystem, Signature_Fetched = EXCLUDED.Signature_Fetched, Components = EXCLUDED.Components, Cves = EXCLUDED.Cves, FixableCves = EXCLUDED.FixableCves, LastUpdated = EXCLUDED.LastUpdated, Priority = EXCLUDED.Priority, RiskScore = EXCLUDED.RiskScore, TopCvss = EXCLUDED.TopCvss, serialized = EXCLUDED.serialized"
	_, err := tx.Exec(ctx, finalStr, values...)
	if err != nil {
		return err
	}

	var query string
	if metadataUpdated {
		for childIdx, child := range cloned.GetMetadata().GetV1().GetLayers() {
			if err := insertIntoImagesLayers(ctx, tx, child, cloned.GetId(), childIdx); err != nil {
				return err
			}
		}

		query = "delete from images_Layers where images_Id = $1 AND idx >= $2"
		_, err = tx.Exec(ctx, query, cloned.GetId(), len(cloned.GetMetadata().GetV1().GetLayers()))
		if err != nil {
			return err
		}
	}

	if !scanUpdated {
		sensorEventsDeduperCounter.With(prometheus.Labels{"status": "deduped"}).Inc()
		return nil
	}
	sensorEventsDeduperCounter.With(prometheus.Labels{"status": "passed"}).Inc()

	// DO NOT CHANGE THE ORDER.
	if err := copyFromImageComponentEdges(ctx, tx, cloned.GetId(), parts.imageComponentEdges...); err != nil {
		return err
	}
	if err := copyFromImageComponents(ctx, tx, parts.components...); err != nil {
		return err
	}
	if err := copyFromImageComponentCVEEdges(ctx, tx, parts.componentCVEEdges...); err != nil {
		return err
	}
	if err := copyFromImageCves(ctx, tx, iTime, parts.vulns...); err != nil {
		return err
	}
	return copyFromImageCVEEdges(ctx, tx, iTime, false, parts.imageCVEEdges...)
}

func getPartsAsSlice(parts common.ImageParts) *imagePartsAsSlice {
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
	return &imagePartsAsSlice{
		image:               parts.Image,
		components:          components,
		vulns:               vulns,
		imageComponentEdges: imageComponentEdges,
		componentCVEEdges:   componentCVEEdges,
		imageCVEEdges:       imageCVEEdges,
	}
}

func insertIntoImagesLayers(ctx context.Context, tx *postgres.Tx, obj *storage.ImageLayer, imageID string, idx int) error {

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

func copyFromImageComponents(ctx context.Context, tx *postgres.Tx, objs ...*storage.ImageComponent) error {
	inputRows := [][]interface{}{}

	var err error

	var deletes []string

	copyCols := []string{
		"id",
		"name",
		"version",
		"operatingsystem",
		"priority",
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
			obj.GetOperatingSystem(),
			obj.GetPriority(),
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
	return removeOrphanedImageComponent(ctx, tx)
}

func copyFromImageComponentEdges(ctx context.Context, tx *postgres.Tx, imageID string, objs ...*storage.ImageComponentEdge) error {
	inputRows := [][]interface{}{}
	var err error

	copyCols := []string{
		"id",
		"location",
		"imageid",
		"imagecomponentid",
		"serialized",
	}

	// Copy does not upsert so have to delete first. This also cleans up orphaned edges.
	_, err = tx.Exec(ctx, "DELETE FROM "+imageComponentEdgesTable+" WHERE imageid = $1", imageID)
	if err != nil {
		return err
	}

	if len(objs) == 0 {
		return nil
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

func copyFromImageCves(ctx context.Context, tx *postgres.Tx, iTime *protoTypes.Timestamp, objs ...*storage.ImageCVE) error {
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
	if err != nil {
		return err
	}

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

	return removeOrphanedImageCVEs(ctx, tx)
}

func copyFromImageComponentCVEEdges(ctx context.Context, tx *postgres.Tx, objs ...*storage.ComponentCVEEdge) error {
	inputRows := [][]interface{}{}
	var err error
	deletes := set.NewStringSet()

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

	// Due to referential constraints, orphaned component-cve edges are removed when orphaned image components are removed.
	return nil
}

func copyFromImageCVEEdges(ctx context.Context, tx *postgres.Tx, iTime *protoTypes.Timestamp, vulnStateUpdate bool,
	objs ...*storage.ImageCVEEdge) error {

	if vulnStateUpdate {
		return copyFromImageCVEEdgesWithVulnStateUpdates(ctx, tx, objs)
	}

	var err error
	var oldEdgeIDs set.Set[string]

	var imageIDs []string
	for _, obj := range objs {
		imageIDs = append(imageIDs, obj.GetImageId())
	}

	// Collect the existing edges for the images to skip re-inserting existing edges.
	oldEdgeIDs, err = getImageCVEEdgeIDs(ctx, tx, imageIDs...)
	if err != nil {
		return err
	}

	inputRows := [][]interface{}{}
	for _, edge := range objs {
		// Since the edge only maintains states enriched by ACS, if the edge already exists, then it should skip copy from.
		if oldEdgeIDs.Remove(edge.GetId()) {
			continue
		}
		edge.FirstImageOccurrence = iTime

		inputRow, err := getImageCVEEdgeRowToInsert(edge)
		if err != nil {
			return err
		}
		inputRows = append(inputRows, inputRow)

		// if we hit our batch size or end of slice, push the edges
		if len(inputRows) > 0 && len(inputRows)%batchSize == 0 {
			err = execCopyFromImageCVEEdges(ctx, tx, inputRows)
			if err != nil {
				return err
			}

			// Clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	if len(inputRows) > 0 {
		err = execCopyFromImageCVEEdges(ctx, tx, inputRows)
		if err != nil {
			return err
		}
	}

	// Remove orphaned edges.
	return removeOrphanedImageCVEEdges(ctx, tx, oldEdgeIDs.AsSlice())
}

func copyFromImageCVEEdgesWithVulnStateUpdates(ctx context.Context, tx *postgres.Tx, edges []*storage.ImageCVEEdge) error {
	inputRows := [][]interface{}{}
	deletes := set.NewStringSet()

	for _, edge := range edges {
		// Add the id to be deleted.
		deletes.Add(edge.GetId())

		inputRow, err := getImageCVEEdgeRowToInsert(edge)
		if err != nil {
			return err
		}
		inputRows = append(inputRows, inputRow)

		// if we hit our batch size or end of slice, delete old edges and push new ones
		if len(inputRows) > 0 && len(inputRows)%batchSize == 0 {
			// Copy does not upsert so have to delete first.
			err = execDeleteFromImageCVEEdges(ctx, tx, deletes)
			if err != nil {
				return err
			}

			// Clear the inserts for the next batch
			deletes = nil

			err = execCopyFromImageCVEEdges(ctx, tx, inputRows)
			if err != nil {
				return err
			}

			// Clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	if len(inputRows) > 0 {
		err := execDeleteFromImageCVEEdges(ctx, tx, deletes)
		if err != nil {
			return err
		}

		err = execCopyFromImageCVEEdges(ctx, tx, inputRows)
		if err != nil {
			return err
		}
	}
	return nil
}

func getImageCVEEdgeRowToInsert(edge *storage.ImageCVEEdge) ([]interface{}, error) {
	serialized, marshalErr := edge.Marshal()
	if marshalErr != nil {
		return nil, marshalErr
	}

	return []interface{}{
		edge.GetId(),
		pgutils.NilOrTime(edge.GetFirstImageOccurrence()),
		edge.GetState(),
		edge.GetImageId(),
		edge.GetImageCveId(),
		serialized,
	}, nil
}

func execCopyFromImageCVEEdges(ctx context.Context, tx *postgres.Tx, inputRows [][]interface{}) error {
	copyCols := []string{
		"id",
		"firstimageoccurrence",
		"state",
		"imageid",
		"imagecveid",
		"serialized",
	}

	_, err := tx.CopyFrom(ctx, pgx.Identifier{imageCVEEdgesTable}, copyCols, pgx.CopyFromRows(inputRows))
	if err != nil {
		return err
	}
	return nil
}

func execDeleteFromImageCVEEdges(ctx context.Context, tx *postgres.Tx, deletes set.Set[string]) error {
	_, err := tx.Exec(ctx, "DELETE FROM "+imageCVEEdgesTable+" WHERE id = ANY($1::text[])", deletes.AsSlice())
	if err != nil {
		return err
	}
	return nil
}

func removeOrphanedImageComponent(ctx context.Context, tx *postgres.Tx) error {
	_, err := tx.Exec(ctx, "DELETE FROM "+imageComponentsTable+" WHERE not exists (select "+imageComponentEdgesTable+".imagecomponentid from "+imageComponentEdgesTable+" where "+imageComponentsTable+".id = "+imageComponentEdgesTable+".imagecomponentid)")
	if err != nil {
		return err
	}
	return nil
}

func removeOrphanedImageCVEs(ctx context.Context, tx *postgres.Tx) error {
	_, err := tx.Exec(ctx, "DELETE FROM "+imageCVEsTable+" WHERE not exists (select "+componentCVEEdgesTable+".imagecveid from "+componentCVEEdgesTable+" where "+imageCVEsTable+".id = "+componentCVEEdgesTable+".imagecveid)")
	if err != nil {
		return err
	}
	return nil
}

func removeOrphanedImageCVEEdges(ctx context.Context, tx *postgres.Tx, orphanedEdgeIDs []string) error {
	if len(orphanedEdgeIDs) == 0 {
		return nil
	}

	_, err := tx.Exec(ctx, "DELETE FROM "+imageCVEEdgesTable+" WHERE id = ANY($1::text[])", orphanedEdgeIDs)
	if err != nil {
		return err
	}
	return nil
}

func (s *storeImpl) isUpdated(oldImage, image *storage.Image) (bool, bool, error) {
	if oldImage == nil {
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

type hashWrapper struct {
	Components []*storage.EmbeddedImageScanComponent `hash:"set"`
}

func populateImageScanHash(scan *storage.ImageScan) error {
	hash, err := hashstructure.Hash(hashWrapper{scan.GetComponents()}, hashstructure.FormatV2, &hashstructure.HashOptions{ZeroNil: true})
	if err != nil {
		return errors.Wrap(err, "calculating hash for image scan")
	}
	scan.Hashoneof = &storage.ImageScan_Hash{
		Hash: hash,
	}
	return nil
}

func (s *storeImpl) upsert(ctx context.Context, obj *storage.Image) error {
	iTime := protoTypes.TimestampNow()

	if !s.noUpdateTimestamps {
		obj.LastUpdated = iTime
	}

	oldImage, _, err := s.GetImageMetadata(ctx, obj.GetId())
	if err != nil {
		return errors.Wrapf(err, "retrieving existing image: %q", obj.GetId())
	}

	metadataUpdated, scanUpdated, err := s.isUpdated(oldImage, obj)
	if err != nil {
		return err
	}
	if !metadataUpdated && !scanUpdated {
		return nil
	}

	if obj.GetScan() != nil {
		if err := populateImageScanHash(obj.GetScan()); err != nil {
			log.Errorf("unable to populate image scan hash for %q", obj.GetId())
		} else if oldImage.GetScan().GetHashoneof() != nil && obj.GetScan().GetHash() == oldImage.GetScan().GetHash() {
			scanUpdated = false
		}
	}

	imageParts := getPartsAsSlice(common.Split(obj, scanUpdated))
	keys := gatherKeys(imageParts)

	return s.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet(keys...), func() error {
		conn, release, err := s.acquireConn(ctx, ops.Get, "Image")
		if err != nil {
			return err
		}
		defer release()

		tx, err := conn.Begin(ctx)
		if err != nil {
			return err
		}

		if err := s.insertIntoImages(ctx, tx, imageParts, metadataUpdated, scanUpdated, iTime); err != nil {
			if err := tx.Rollback(ctx); err != nil {
				return err
			}
			return err
		}
		return tx.Commit(ctx)
	})
}

// Upsert upserts image into the store.
func (s *storeImpl) Upsert(ctx context.Context, obj *storage.Image) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Upsert, "Image")

	return pgutils.Retry(func() error {
		return s.upsert(ctx, obj)
	})
}

// Count returns the number of objects in the store
func (s *storeImpl) Count(ctx context.Context) (int, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Count, "Image")

	return pgutils.Retry2(func() (int, error) {
		row := s.db.QueryRow(ctx, countStmt)
		var count int
		if err := row.Scan(&count); err != nil {
			return 0, err
		}
		return count, nil
	})
}

// Exists returns if the id exists in the store
func (s *storeImpl) Exists(ctx context.Context, id string) (bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Exists, "Image")

	return pgutils.Retry2(func() (bool, error) {
		row := s.db.QueryRow(ctx, existsStmt, id)
		var exists bool
		if err := row.Scan(&exists); err != nil {
			return false, pgutils.ErrNilIfNoRows(err)
		}
		return exists, nil
	})
}

// Get returns the object, if it exists from the store.
func (s *storeImpl) Get(ctx context.Context, id string) (*storage.Image, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "Image")

	return pgutils.Retry3(func() (*storage.Image, bool, error) {
		return s.retryableGet(ctx, id)
	})
}

func (s *storeImpl) retryableGet(ctx context.Context, id string) (*storage.Image, bool, error) {
	conn, release, err := s.acquireConn(ctx, ops.Get, "Image")
	if err != nil {
		return nil, false, err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, false, err
	}
	image, found, err := s.getFullImage(ctx, tx, id)
	// No changes are made to the database, so COMMIT or ROLLBACK have same effect.
	if err := tx.Commit(ctx); err != nil {
		return nil, false, err
	}
	return image, found, err
}

func (s *storeImpl) getFullImage(ctx context.Context, tx *postgres.Tx, imageID string) (*storage.Image, bool, error) {
	row := tx.QueryRow(ctx, getImageMetaStmt, imageID)
	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}

	var image storage.Image
	if err := image.Unmarshal(data); err != nil {
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
		log.Errorf("Number of component (%d) in image-component edges is not equal to number of stored components (%d) for image %s (imageID=%s)",
			len(componentEdgeMap), len(componentMap), image.GetName().GetFullName(), image.GetId())
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

func (s *storeImpl) acquireConn(ctx context.Context, op ops.Op, typ string) (*postgres.Conn, func(), error) {
	defer metrics.SetAcquireDBConnDuration(time.Now(), op, typ)
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	return conn, conn.Release, nil
}

func getImageComponentEdges(ctx context.Context, tx *postgres.Tx, imageID string) (map[string]*storage.ImageComponentEdge, error) {
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
		if err := msg.Unmarshal(data); err != nil {
			return nil, err
		}
		componentIDToEdgeMap[msg.GetImageComponentId()] = msg
	}
	return componentIDToEdgeMap, rows.Err()
}

func getImageCVEEdgeIDs(ctx context.Context, tx *postgres.Tx, imageIDs ...string) (set.StringSet, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "ImageCVEEdges")

	rows, err := tx.Query(ctx, "SELECT id FROM "+imageCVEEdgesTable+" WHERE imageid = ANY($1::text[])", imageIDs)
	if err != nil {
		return nil, err
	}
	ids, err := scanIDs(rows)
	if err != nil {
		return nil, err
	}
	return set.NewStringSet(ids...), nil
}

func getImageCVEEdges(ctx context.Context, tx *postgres.Tx, imageID string) (map[string]*storage.ImageCVEEdge, error) {
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
		if err := msg.Unmarshal(data); err != nil {
			return nil, err
		}
		cveIDToEdgeMap[msg.GetImageCveId()] = msg
	}
	return cveIDToEdgeMap, rows.Err()
}

func getImageComponents(ctx context.Context, tx *postgres.Tx, componentIDs []string) (map[string]*storage.ImageComponent, error) {
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
		if err := msg.Unmarshal(data); err != nil {
			return nil, err
		}
		idToComponentMap[msg.GetId()] = msg
	}
	return idToComponentMap, rows.Err()
}

func getComponentCVEEdges(ctx context.Context, tx *postgres.Tx, componentIDs []string) (map[string][]*storage.ComponentCVEEdge, error) {
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
		if err := msg.Unmarshal(data); err != nil {
			return nil, err
		}
		componentIDToEdgeMap[msg.GetImageComponentId()] = append(componentIDToEdgeMap[msg.GetImageComponentId()], msg)
	}
	return componentIDToEdgeMap, rows.Err()
}

func getCVEs(ctx context.Context, tx *postgres.Tx, cveIDs []string) (map[string]*storage.ImageCVE, error) {
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
		if err := msg.Unmarshal(data); err != nil {
			return nil, err
		}
		idToCVEMap[msg.GetId()] = msg
	}
	return idToCVEMap, rows.Err()
}

// Delete removes the specified ID from the store.
func (s *storeImpl) Delete(ctx context.Context, id string) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "Image")

	return pgutils.Retry(func() error {
		return s.retryableDelete(ctx, id)
	})
}

func (s *storeImpl) retryableDelete(ctx context.Context, id string) error {
	conn, release, err := s.acquireConn(ctx, ops.Remove, "Image")
	if err != nil {
		return err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	if err := s.deleteImageTree(ctx, tx, id); err != nil {
		if err := tx.Rollback(ctx); err != nil {
			return err
		}
		return err
	}
	return tx.Commit(ctx)
}

func (s *storeImpl) deleteImageTree(ctx context.Context, tx *postgres.Tx, imageID string) error {
	// Delete from image table.
	if _, err := tx.Exec(ctx, "delete from "+imagesTable+" where Id = $1", imageID); err != nil {
		return err
	}

	// Delete orphaned image components.
	if _, err := tx.Exec(ctx, "delete from "+imageComponentsTable+" where not exists (select "+imageComponentEdgesTable+".imagecomponentid FROM "+imageComponentEdgesTable+" where "+imageComponentsTable+".id = "+imageComponentEdgesTable+".imagecomponentid)"); err != nil {
		return err
	}

	// Delete orphaned cves.
	if _, err := tx.Exec(ctx, "delete from "+imageCVEsTable+" where not exists (select "+componentCVEEdgesTable+".imagecveid FROM "+componentCVEEdgesTable+" where "+imageCVEsTable+".id = "+componentCVEEdgesTable+".imagecveid)"); err != nil {
		return err
	}
	return nil
}

// GetIDs returns all the IDs for the store
func (s *storeImpl) GetIDs(ctx context.Context) ([]string, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetAll, "ImageIDs")

	return pgutils.Retry2(func() ([]string, error) {
		return s.retryableGetIDs(ctx)
	})
}

func (s *storeImpl) retryableGetIDs(ctx context.Context) ([]string, error) {
	rows, err := s.db.Query(ctx, getImageIDsStmt)
	if err != nil {
		return nil, pgutils.ErrNilIfNoRows(err)
	}
	ids, err := scanIDs(rows)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// GetMany returns the objects specified by the IDs or the index in the missing indices slice
func (s *storeImpl) GetMany(ctx context.Context, ids []string) ([]*storage.Image, []int, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "Image")

	return pgutils.Retry3(func() ([]*storage.Image, []int, error) {
		return s.retryableGetMany(ctx, ids)
	})
}

func (s *storeImpl) retryableGetMany(ctx context.Context, ids []string) ([]*storage.Image, []int, error) {
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
			// No changes are made to the database, so COMMIT or ROLLBACK have the same effect.
			if err := tx.Commit(ctx); err != nil {
				return nil, nil, err
			}
			return nil, nil, err
		}
		if !found {
			continue
		}
		resultsByID[msg.GetId()] = msg
	}

	// No changes are made to the database, so COMMIT or ROLLBACK have the same effect.
	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
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

func dropAllTablesInImageTree(ctx context.Context, db postgres.DB) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS images CASCADE")
	dropTableImagesLayers(ctx, db)
	dropTableImageComponents(ctx, db)
	dropTableImageCVEs(ctx, db)
	dropTableImageCVEEdges(ctx, db)
	dropTableComponentCVEEdges(ctx, db)
	dropTableImageComponentEdges(ctx, db)
}

func dropTableImagesLayers(ctx context.Context, db postgres.DB) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS images_Layers CASCADE")
}

func dropTableImageComponents(ctx context.Context, db postgres.DB) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS "+imageComponentsTable+" CASCADE")
}

func dropTableImageCVEs(ctx context.Context, db postgres.DB) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS "+imageCVEsTable+" CASCADE")
}

func dropTableImageCVEEdges(ctx context.Context, db postgres.DB) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS "+imageCVEEdgesTable+" CASCADE")
}

func dropTableComponentCVEEdges(ctx context.Context, db postgres.DB) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS "+componentCVEEdgesTable+" CASCADE")
}

func dropTableImageComponentEdges(ctx context.Context, db postgres.DB) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS "+imageComponentEdgesTable+" CASCADE")
}

// Destroy drops image table.
func Destroy(ctx context.Context, db postgres.DB) {
	dropAllTablesInImageTree(ctx, db)
}

// CreateTableAndNewStore returns a new Store instance for testing
func CreateTableAndNewStore(ctx context.Context, db postgres.DB, gormDB *gorm.DB, noUpdateTimestamps bool) store.Store {
	pgutils.CreateTableFromModel(ctx, gormDB, pkgSchema.CreateTableImagesStmt)
	pgutils.CreateTableFromModel(ctx, gormDB, pkgSchema.CreateTableImageComponentsStmt)
	pgutils.CreateTableFromModel(ctx, gormDB, pkgSchema.CreateTableImageCvesStmt)
	pgutils.CreateTableFromModel(ctx, gormDB, pkgSchema.CreateTableImageComponentEdgesStmt)
	pgutils.CreateTableFromModel(ctx, gormDB, pkgSchema.CreateTableImageComponentCveEdgesStmt)
	pgutils.CreateTableFromModel(ctx, gormDB, pkgSchema.CreateTableImageCveEdgesStmt)
	return New(db, noUpdateTimestamps, concurrency.NewKeyFence())
}

// GetImageMetadata returns the image without scan/component data.
func (s *storeImpl) GetImageMetadata(ctx context.Context, id string) (*storage.Image, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "ImageMetadata")

	return pgutils.Retry3(func() (*storage.Image, bool, error) {
		return s.retryableGetImageMetadata(ctx, id)
	})
}

func (s *storeImpl) retryableGetImageMetadata(ctx context.Context, id string) (*storage.Image, bool, error) {
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
	if err := msg.Unmarshal(data); err != nil {
		return nil, false, err
	}
	return &msg, true, nil
}

// GetManyImageMetadata returns images without scan/component data.
func (s *storeImpl) GetManyImageMetadata(ctx context.Context, ids []string) ([]*storage.Image, []int, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "Image")

	return pgutils.Retry3(func() ([]*storage.Image, []int, error) {
		return s.retryableGetManyImageMetadata(ctx, ids)
	})
}

func (s *storeImpl) retryableGetManyImageMetadata(ctx context.Context, ids []string) ([]*storage.Image, []int, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}

	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(storage.Access_READ_ACCESS).Resource(targetResource)
	scopeTree, err := scopeChecker.EffectiveAccessScope(permissions.ResourceWithAccess{
		Resource: targetResource,
		Access:   storage.Access_READ_ACCESS,
	})
	if err != nil {
		return nil, nil, err
	}
	sacQueryFilter, err := sac.BuildNonVerboseClusterNamespaceLevelSACQueryFilter(scopeTree)
	if err != nil {
		return nil, nil, err
	}
	q := search.ConjunctionQuery(
		sacQueryFilter,
		search.NewQueryBuilder().AddExactMatches(search.ImageSHA, ids...).ProtoQuery(),
	)

	rows, err := pgSearch.RunGetManyQueryForSchema[storage.Image](ctx, schema, q, s.db)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			missingIndices := make([]int, 0, len(ids))
			for i := range ids {
				missingIndices = append(missingIndices, i)
			}
			return nil, missingIndices, nil
		}
		return nil, nil, err
	}
	resultsByID := make(map[string]*storage.Image, len(rows))
	for _, msg := range rows {
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

func (s *storeImpl) UpdateVulnState(ctx context.Context, cve string, imageIDs []string, state storage.VulnerabilityState) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Update, "UpdateVulnState")

	return pgutils.Retry(func() error {
		return s.retryableUpdateVulnState(ctx, cve, imageIDs, state)
	})
}

func (s *storeImpl) retryableUpdateVulnState(ctx context.Context, cve string, imageIDs []string, state storage.VulnerabilityState) error {
	if len(imageIDs) == 0 {
		return nil
	}

	conn, release, err := s.acquireConn(ctx, ops.Update, "UpdateVulnState")
	if err != nil {
		return err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	// Collect stored edges.
	rows, err := tx.Query(ctx, "SELECT "+imageCVEEdgesTable+".serialized FROM "+imageCVEEdgesTable+" "+
		"inner join "+imageCVEsTable+" on "+imageCVEEdgesTable+".imagecveid = "+imageCVEsTable+".id "+
		"WHERE "+imageCVEEdgesTable+".imageid = ANY($1::text[]) AND "+imageCVEsTable+".cvebaseinfo_cve = $2", imageIDs, cve)
	if err != nil {
		return err
	}
	defer rows.Close()
	var imageCVEEdges []*storage.ImageCVEEdge
	imageCVEEdges, err = pgutils.ScanRows[storage.ImageCVEEdge](rows)
	if err != nil || len(imageCVEEdges) == 0 {
		return err
	}

	// Update state.
	cveIDs := make([]string, 0, len(imageCVEEdges))
	for _, edge := range imageCVEEdges {
		edge.State = state
		cveIDs = append(cveIDs, edge.GetImageCveId())
	}

	// Construct keys to lock.
	keys := make([][]byte, 0, len(cveIDs)+len(imageIDs))
	for _, id := range imageIDs {
		keys = append(keys, []byte(id))
	}
	for _, id := range cveIDs {
		keys = append(keys, []byte(id))
	}

	return s.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet(keys...), func() error {
		err = copyFromImageCVEEdges(ctx, tx, protoTypes.TimestampNow(), true, imageCVEEdges...)
		if err != nil {
			if err := tx.Rollback(ctx); err != nil {
				return err
			}
			return err
		}
		return tx.Commit(ctx)
	})
}

func gatherKeys(parts *imagePartsAsSlice) [][]byte {
	// We only need to collect image, component, and vuln keys because edges are derived from those resources and edge
	// datastores are do not support upserts and deletes.
	keys := make([][]byte, 0, len(parts.components)+len(parts.vulns)+1)
	keys = append(keys, []byte(parts.image.GetId()))
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
	return ids, rows.Err()
}
