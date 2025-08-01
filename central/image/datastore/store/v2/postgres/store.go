package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	convertutils "github.com/stackrox/rox/central/cve/converter/utils"
	"github.com/stackrox/rox/central/image/datastore/store"
	"github.com/stackrox/rox/central/image/datastore/store/common/v2"
	"github.com/stackrox/rox/central/image/views"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	imagesTable                      = pkgSchema.ImagesTableName
	imageComponentsV2Table           = pkgSchema.ImageComponentV2TableName
	imageComponentsV2CVEsTable       = pkgSchema.ImageCvesV2TableName
	imageCVEsLegacyTable             = pkgSchema.ImageCvesTableName
	imageCVEEdgesLegacyTable         = pkgSchema.ImageCveEdgesTableName
	cveCreatedAtFieldName            = "cveBaseInfo_CVE"
	cveFirstImageOccurrenceFieldName = "FirstImageOccurrence"

	getImageMetaStmt = "SELECT serialized FROM " + imagesTable + " WHERE Id = $1"
)

var (
	log    = logging.LoggerForModule()
	schema = pkgSchema.ImagesSchema
)

type imagePartsAsSlice struct {
	image        *storage.Image
	componentsV2 []*storage.ImageComponentV2
	cvesV2       []*storage.ImageCVEV2
}

type timeFields struct {
	createdAt            time.Time
	firstImageOccurrence time.Time
}

// TODO(ROX-28222): Refactor logic operating on other tables out and up to the datastore layer.

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
	iTime time.Time,
) error {
	// First Image Occurrence and Created At are set based on the CVE itself, not the CVE
	// within the image.  Since a CVE can occur multiple times within an image we can grab
	// those times for the incoming data and set the times appropriately.  We will later go through the
	// existing CVEs to make further adjustments if necessary to make sure we do not overwrite
	// the times of previous occurrences.
	cveTimeMap := make(map[string]*timeFields)
	for _, cve := range parts.cvesV2 {
		if val, ok := cveTimeMap[cve.GetCveBaseInfo().GetCve()]; ok {
			if cve.GetCveBaseInfo().GetCreatedAt() != nil && val.createdAt.After(cve.GetCveBaseInfo().GetCreatedAt().AsTime()) {
				val.createdAt = cve.GetCveBaseInfo().GetCreatedAt().AsTime()
			}
			if cve.GetFirstImageOccurrence() != nil && val.firstImageOccurrence.After(cve.GetFirstImageOccurrence().AsTime()) {
				val.firstImageOccurrence = cve.GetFirstImageOccurrence().AsTime()
			}
		} else {
			if cve.GetFirstImageOccurrence() == nil {
				cve.FirstImageOccurrence = timestamppb.New(iTime)
			}
			if cve.GetCveBaseInfo().GetCreatedAt() == nil {
				cve.GetCveBaseInfo().CreatedAt = timestamppb.New(iTime)
			}
			cveTimeMap[cve.GetCveBaseInfo().GetCve()] = &timeFields{
				createdAt:            cve.GetCveBaseInfo().GetCreatedAt().AsTime(),
				firstImageOccurrence: cve.GetFirstImageOccurrence().AsTime(),
			}
		}
	}

	// Grab all CVEs for the image.
	existingCVEs, err := getImageCVEs(ctx, tx, parts.image.GetId())
	if err != nil {
		return err
	}

	if len(existingCVEs) == 0 {
		// If we did not find any existing CVEs for the image, we may have just upgraded to the version using new CVE data model.
		// So we try to migrate the CVE created and first image occurrence timestamps from the legacy model.
		existingCVEs, err = getLegacyImageCVEs(ctx, tx, parts.image.GetId())
		if err != nil {
			return err
		}
	}

	for _, cve := range existingCVEs {
		// If the existing CVE is not already in the map that implies it no longer exists for this image and
		// the CVE will be removed.
		if val, ok := cveTimeMap[cve.GetCve()]; ok {
			if cve.GetFirstSystemOccurrence() != nil && val.createdAt.After(cve.GetFirstSystemOccurrence().AsTime()) {
				val.createdAt = cve.GetFirstSystemOccurrence().AsTime()
			}
			if cve.GetFirstImageOccurrence() != nil && val.firstImageOccurrence.After(cve.GetFirstImageOccurrence().AsTime()) {
				val.firstImageOccurrence = cve.GetFirstImageOccurrence().AsTime()
			}
		}
	}

	cloned := parts.image
	// Since we are converting the component and CVE data embedded within the Image.Scan, we
	// need to clear that data out so that it is not stored with Image thus greatly duplicating data.
	if cloned.GetScan().GetComponents() != nil {
		cloned = parts.image.CloneVT()
		cloned.Scan.Components = nil
	}
	serialized, marshalErr := cloned.MarshalVT()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		cloned.GetId(),
		cloned.GetName().GetRegistry(),
		cloned.GetName().GetRemote(),
		cloned.GetName().GetTag(),
		cloned.GetName().GetFullName(),
		protocompat.NilOrTime(cloned.GetMetadata().GetV1().GetCreated()),
		cloned.GetMetadata().GetV1().GetUser(),
		cloned.GetMetadata().GetV1().GetCommand(),
		cloned.GetMetadata().GetV1().GetEntrypoint(),
		cloned.GetMetadata().GetV1().GetVolumes(),
		cloned.GetMetadata().GetV1().GetLabels(),
		protocompat.NilOrTime(cloned.GetScan().GetScanTime()),
		cloned.GetScan().GetOperatingSystem(),
		protocompat.NilOrTime(cloned.GetSignature().GetFetched()),
		cloned.GetComponents(),
		cloned.GetCves(),
		cloned.GetFixableCves(),
		protocompat.NilOrTime(cloned.GetLastUpdated()),
		cloned.GetPriority(),
		cloned.GetRiskScore(),
		cloned.GetTopCvss(),
		serialized,
	}

	finalStr := "INSERT INTO " + imagesTable + " (Id, Name_Registry, Name_Remote, Name_Tag, Name_FullName, Metadata_V1_Created, Metadata_V1_User, Metadata_V1_Command, Metadata_V1_Entrypoint, Metadata_V1_Volumes, Metadata_V1_Labels, Scan_ScanTime, Scan_OperatingSystem, Signature_Fetched, Components, Cves, FixableCves, LastUpdated, Priority, RiskScore, TopCvss, serialized) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, Name_Registry = EXCLUDED.Name_Registry, Name_Remote = EXCLUDED.Name_Remote, Name_Tag = EXCLUDED.Name_Tag, Name_FullName = EXCLUDED.Name_FullName, Metadata_V1_Created = EXCLUDED.Metadata_V1_Created, Metadata_V1_User = EXCLUDED.Metadata_V1_User, Metadata_V1_Command = EXCLUDED.Metadata_V1_Command, Metadata_V1_Entrypoint = EXCLUDED.Metadata_V1_Entrypoint, Metadata_V1_Volumes = EXCLUDED.Metadata_V1_Volumes, Metadata_V1_Labels = EXCLUDED.Metadata_V1_Labels, Scan_ScanTime = EXCLUDED.Scan_ScanTime, Scan_OperatingSystem = EXCLUDED.Scan_OperatingSystem, Signature_Fetched = EXCLUDED.Signature_Fetched, Components = EXCLUDED.Components, Cves = EXCLUDED.Cves, FixableCves = EXCLUDED.FixableCves, LastUpdated = EXCLUDED.LastUpdated, Priority = EXCLUDED.Priority, RiskScore = EXCLUDED.RiskScore, TopCvss = EXCLUDED.TopCvss, serialized = EXCLUDED.serialized"
	_, err = tx.Exec(ctx, finalStr, values...)
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

		query = "DELETE FROM images_Layers WHERE images_Id = $1 AND idx >= $2"
		_, err = tx.Exec(ctx, query, cloned.GetId(), len(cloned.GetMetadata().GetV1().GetLayers()))
		if err != nil {
			return err
		}
	}

	// If the scan is not new, we do not need to bother writing the components and CVEs as the latest already
	// exist.
	if !scanUpdated {
		common.SensorEventsDeduperCounter.With(prometheus.Labels{"status": "deduped"}).Inc()
		return nil
	}
	common.SensorEventsDeduperCounter.With(prometheus.Labels{"status": "passed"}).Inc()

	err = s.copyFromImageComponentsV2(ctx, tx, parts.image.GetId(), parts.componentsV2...)
	if err != nil {
		return err
	}

	return copyFromImageComponentV2Cves(ctx, tx, iTime, cveTimeMap, parts.cvesV2...)
}

func getPartsAsSlice(parts common.ImageParts) *imagePartsAsSlice {
	componentsV2 := make([]*storage.ImageComponentV2, 0, len(parts.Children))
	vulns := make([]*storage.ImageCVEV2, 0)
	for _, child := range parts.Children {
		componentsV2 = append(componentsV2, child.ComponentV2)
		for _, gChild := range child.Children {
			vulns = append(vulns, gChild.CVEV2)
		}
	}
	return &imagePartsAsSlice{
		image:        parts.Image,
		componentsV2: componentsV2,
		cvesV2:       vulns,
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

func (s *storeImpl) copyFromImageComponentsV2(ctx context.Context, tx *postgres.Tx, imageID string, objs ...*storage.ImageComponentV2) error {
	// Each scan is complete replacement.  So first thing we do is remove the old components and CVEs for an image.
	if err := s.deleteImageComponents(ctx, tx, imageID); err != nil {
		return err
	}
	batchSize := pgSearch.MaxBatchSize
	if len(objs) < batchSize {
		batchSize = len(objs)
	}
	inputRows := make([][]interface{}, 0, batchSize)

	copyCols := []string{
		"id",
		"name",
		"version",
		"priority",
		"source",
		"riskscore",
		"topcvss",
		"operatingsystem",
		"imageid",
		"location",
		"serialized",
	}

	for idx, obj := range objs {
		serialized, marshalErr := obj.MarshalVT()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{
			obj.GetId(),
			obj.GetName(),
			obj.GetVersion(),
			obj.GetPriority(),
			obj.GetSource(),
			obj.GetRiskScore(),
			obj.GetTopCvss(),
			obj.GetOperatingSystem(),
			obj.GetImageId(),
			obj.GetLocation(),
			serialized,
		})

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			if _, err := tx.CopyFrom(ctx, pgx.Identifier{imageComponentsV2Table}, copyCols, pgx.CopyFromRows(inputRows)); err != nil {
				return err
			}
			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return nil
}

func copyFromImageComponentV2Cves(ctx context.Context, tx *postgres.Tx, iTime time.Time, cveTimeMap map[string]*timeFields, objs ...*storage.ImageCVEV2) error {
	batchSize := pgSearch.MaxBatchSize
	if len(objs) < batchSize {
		batchSize = len(objs)
	}
	inputRows := make([][]interface{}, 0, batchSize)

	copyCols := []string{
		"id",
		"imageid",
		"cvebaseinfo_cve",
		"cvebaseinfo_publishedon",
		"cvebaseinfo_createdat",
		"cvebaseinfo_epss_epssprobability",
		"cvss",
		"severity",
		"impactscore",
		"nvdcvss",
		"firstimageoccurrence",
		"state",
		"isfixable",
		"fixedby",
		"componentid",
		"advisory_name",
		"advisory_link",
		"serialized",
	}

	for idx, obj := range objs {
		// If we have seen this CVE in the image already, set the times consistently.
		if cveTimes := cveTimeMap[obj.GetCveBaseInfo().GetCve()]; cveTimes != nil {
			obj.CveBaseInfo.CreatedAt = protocompat.ConvertTimeToTimestampOrNil(&cveTimes.createdAt)
			obj.FirstImageOccurrence = protocompat.ConvertTimeToTimestampOrNil(&cveTimes.firstImageOccurrence)
		} else {
			if obj.GetCveBaseInfo().GetCreatedAt() == nil {
				obj.CveBaseInfo.CreatedAt = protocompat.ConvertTimeToTimestampOrNil(&iTime)
			}
			if obj.GetFirstImageOccurrence() == nil {
				obj.FirstImageOccurrence = protocompat.ConvertTimeToTimestampOrNil(&iTime)
			}
		}

		serialized, marshalErr := obj.MarshalVT()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{
			obj.GetId(),
			obj.GetImageId(),
			obj.GetCveBaseInfo().GetCve(),
			protocompat.NilOrTime(obj.GetCveBaseInfo().GetPublishedOn()),
			protocompat.NilOrTime(obj.GetCveBaseInfo().GetCreatedAt()),
			obj.GetCveBaseInfo().GetEpss().GetEpssProbability(),
			obj.GetCvss(),
			obj.GetSeverity(),
			obj.GetImpactScore(),
			obj.GetNvdcvss(),
			protocompat.NilOrTime(obj.GetFirstImageOccurrence()),
			obj.GetState(),
			obj.GetIsFixable(),
			obj.GetFixedBy(),
			obj.GetComponentId(),
			obj.GetAdvisory().GetName(),
			obj.GetAdvisory().GetLink(),
			serialized,
		})

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// copy does not upsert so have to delete first.  parent deletion cascades so only need to
			// delete for the top level parent
			if _, err := tx.CopyFrom(ctx, pgx.Identifier{imageComponentsV2CVEsTable}, copyCols, pgx.CopyFromRows(inputRows)); err != nil {
				return err
			}
			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return nil
}

func (s *storeImpl) isUpdated(oldImage, image *storage.Image) (bool, bool, error) {
	if oldImage == nil {
		return true, true, nil
	}
	metadataUpdated := false
	scanUpdated := false

	if protocompat.CompareTimestamps(oldImage.GetMetadata().GetV1().GetCreated(), image.GetMetadata().GetV1().GetCreated()) > 0 {
		image.Metadata = oldImage.GetMetadata()
	} else {
		metadataUpdated = true
	}

	// We skip rewriting components and cves if scan is not newer, hence we do not need to merge.
	if protocompat.CompareTimestamps(oldImage.GetScan().GetScanTime(), image.GetScan().GetScanTime()) > 0 {
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
	iTime := time.Now()

	if !s.noUpdateTimestamps {
		obj.LastUpdated = protocompat.ConvertTimeToTimestampOrNil(&iTime)
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

	// This check ensures that if the components table was empty, we attempt to upsert the related components
	// so that the new data model tables are populated in the event this image has data in the scan.
	componentsEmpty, err := s.isComponentsTableEmpty(ctx, obj.GetId())
	if err != nil {
		return err
	}

	scanUpdated = scanUpdated || componentsEmpty

	splitParts, err := common.SplitV2(obj, scanUpdated)
	if err != nil {
		return err
	}
	imageParts := getPartsAsSlice(splitParts)
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

	return pgutils.Retry(ctx, func() error {
		return s.upsert(ctx, obj)
	})
}

// Count returns the number of objects in the store
func (s *storeImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Count, "Image")

	return pgutils.Retry2(ctx, func() (int, error) {
		return pgSearch.RunCountRequestForSchema(ctx, schema, q, s.db)
	})
}

// Search returns the result matching the query.
func (s *storeImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Search, "Image")

	return pgutils.Retry2(ctx, func() ([]search.Result, error) {
		return pgSearch.RunSearchRequestForSchema(ctx, schema, q, s.db)
	})
}

// Exists returns if the id exists in the store
func (s *storeImpl) Exists(ctx context.Context, id string) (bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Exists, "Image")

	return pgutils.Retry2(ctx, func() (bool, error) {
		return s.retryableExists(ctx, id)
	})
}

func (s *storeImpl) retryableExists(ctx context.Context, id string) (bool, error) {
	q := search.NewQueryBuilder().AddDocIDs(id).ProtoQuery()
	count, err := pgSearch.RunCountRequestForSchema(ctx, schema, q, s.db)
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

// Get returns the object, if it exists from the store.
func (s *storeImpl) Get(ctx context.Context, id string) (*storage.Image, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "Image")

	return pgutils.Retry3(ctx, func() (*storage.Image, bool, error) {
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

func (s *storeImpl) populateImage(ctx context.Context, tx *postgres.Tx, image *storage.Image) error {
	components, err := getImageComponents(ctx, tx, image.Id)
	if err != nil {
		return err
	}

	imageParts := common.ImageParts{
		Image:    image,
		Children: []common.ComponentParts{},
	}
	for _, component := range components {
		cves, err := getImageComponentCVEs(ctx, tx, component.GetId())
		if err != nil {
			return err
		}

		cveParts := make([]common.CVEParts, 0, len(cves))
		for _, cve := range cves {
			cvePart := common.CVEParts{
				CVEV2: cve,
			}
			cveParts = append(cveParts, cvePart)
		}

		child := common.ComponentParts{
			ComponentV2: component,
			Children:    cveParts,
		}
		imageParts.Children = append(imageParts.Children, child)
	}
	common.MergeV2(imageParts)
	return nil
}

func (s *storeImpl) getFullImage(ctx context.Context, tx *postgres.Tx, imageID string) (*storage.Image, bool, error) {
	row := tx.QueryRow(ctx, getImageMetaStmt, imageID)
	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}

	var image storage.Image
	if err := image.UnmarshalVTUnsafe(data); err != nil {
		return nil, false, err
	}

	if err := s.populateImage(ctx, tx, &image); err != nil {
		return nil, false, err
	}
	return &image, true, nil
}

func (s *storeImpl) acquireConn(ctx context.Context, op ops.Op, typ string) (*postgres.Conn, func(), error) {
	defer metrics.SetAcquireDBConnDuration(time.Now(), op, typ)
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	return conn, conn.Release, nil
}

func getImageComponents(ctx context.Context, tx *postgres.Tx, imageID string) ([]*storage.ImageComponentV2, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "ImageComponentsV2")

	// Using this method instead of accessing the component store to ensure the query is in the same transaction as
	// the updates.  That may prove to not matter, but for now doing it this way.
	rows, err := tx.Query(ctx, "SELECT serialized FROM "+imageComponentsV2Table+" WHERE imageid = $1", imageID)
	if err != nil {
		return nil, err
	}
	return pgutils.ScanRows[storage.ImageComponentV2, *storage.ImageComponentV2](rows)
}

func getImageComponentCVEs(ctx context.Context, tx *postgres.Tx, componentID string) ([]*storage.ImageCVEV2, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "ImageCVEsV2")

	// Using this method instead of accessing the component store to ensure the query is in the same transaction as
	// the updates.  That may prove to not matter, but for now doing it this way.
	rows, err := tx.Query(ctx, "SELECT serialized FROM "+imageComponentsV2CVEsTable+" WHERE componentid = $1", componentID)
	if err != nil {
		return nil, err
	}
	return pgutils.ScanRows[storage.ImageCVEV2, *storage.ImageCVEV2](rows)
}

func getImageCVEs(ctx context.Context, tx *postgres.Tx, imageID string) ([]*storage.EmbeddedVulnerability, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "ImageCVEsV2")

	// Using this method instead of accessing the component store to ensure the query is in the same transaction as
	// the updates.  That may prove to not matter, but for now doing it this way.
	rows, err := tx.Query(ctx, "SELECT serialized FROM "+imageComponentsV2CVEsTable+" WHERE imageid = $1", imageID)
	if err != nil {
		return nil, err
	}

	var imageCVEs []*storage.ImageCVEV2
	imageCVEs, err = pgutils.ScanRows[storage.ImageCVEV2, *storage.ImageCVEV2](rows)
	if err != nil {
		return nil, err
	}

	vulns := make([]*storage.EmbeddedVulnerability, 0, len(imageCVEs))
	for _, cve := range imageCVEs {
		vulns = append(vulns, convertutils.ImageCVEV2ToEmbeddedVulnerability(cve))
	}

	return vulns, nil
}

// The purpose of this function is to get legacy CVEs for the given imageID so that we can migrate the CVE created and
// first image occurrence timestamps to the new CVE data model. So we do not populate the fixedBy and vulnerability state
// in the returned vulns as that information is not necessary for migrating the timestamps.
func getLegacyImageCVEs(ctx context.Context, tx *postgres.Tx, imageID string) ([]*storage.EmbeddedVulnerability, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "ImageCVEs")

	// Using this method instead of accessing the legacy image CVE and component stores because the legacy stores
	// would not be initialized when the new data model is enabled
	cveRows, err := tx.Query(ctx, "SELECT "+imageCVEsLegacyTable+".serialized FROM "+imageCVEsLegacyTable+
		" INNER JOIN "+imageCVEEdgesLegacyTable+" ON "+imageCVEsLegacyTable+".Id = "+imageCVEEdgesLegacyTable+".ImageCveId"+
		" WHERE "+imageCVEEdgesLegacyTable+".ImageId = $1", imageID)
	if err != nil {
		return nil, err
	}

	// There should be at most one edge for a given pair of cveID and imageID in the image CVE edges table. And in the above query,
	// we filter the image CVE edges by a single imageID. So there should be only one row per cveID in the query's result.
	var imageCVEs []*storage.ImageCVE
	imageCVEs, err = pgutils.ScanRows[storage.ImageCVE, *storage.ImageCVE](cveRows)
	if err != nil {
		return nil, err
	}

	edgeRows, err := tx.Query(ctx, "SELECT serialized FROM "+imageCVEEdgesLegacyTable+" WHERE ImageId = $1", imageID)
	if err != nil {
		return nil, err
	}

	var imageCVEEdges []*storage.ImageCVEEdge
	imageCVEEdges, err = pgutils.ScanRows[storage.ImageCVEEdge, *storage.ImageCVEEdge](edgeRows)
	if err != nil {
		return nil, err
	}

	// There should be at most one edge for a given pair of cveID and imageID in the image CVE edges table. And in the above query,
	// we filter the image CVE edges by a single imageID. So there should be only one row per cveID in the query's result.
	edgesByCveID := make(map[string]*storage.ImageCVEEdge)
	for _, edge := range imageCVEEdges {
		if _, ok := edgesByCveID[edge.GetImageCveId()]; !ok {
			edgesByCveID[edge.GetImageCveId()] = edge
		}
	}

	vulns := make([]*storage.EmbeddedVulnerability, 0, len(imageCVEs))
	for _, cve := range imageCVEs {
		edge, ok := edgesByCveID[cve.GetId()]
		if !ok {
			continue
		}
		vuln := convertutils.ImageCVEToEmbeddedVulnerability(cve)
		vuln.FirstImageOccurrence = edge.GetFirstImageOccurrence()
		vulns = append(vulns, vuln)
	}

	return vulns, nil
}

// Delete removes the specified ID from the store.
func (s *storeImpl) Delete(ctx context.Context, id string) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "Image")

	return pgutils.Retry(ctx, func() error {
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
	if _, err := tx.Exec(ctx, "DELETE FROM "+imagesTable+" WHERE Id = $1", imageID); err != nil {
		return err
	}

	// We do not need to delete the CVEs because of the FK relationship to components with the cascade action.
	return s.deleteImageComponents(ctx, tx, imageID)
}

func (s *storeImpl) deleteImageComponents(ctx context.Context, tx *postgres.Tx, imageID string) error {
	// Delete image components for this image
	if _, err := tx.Exec(ctx, "DELETE FROM "+imageComponentsV2Table+" WHERE imageid = $1", imageID); err != nil {
		return err
	}

	return nil
}

// GetByIDs returns the objects specified by the IDs or the index in the missing indices slice
func (s *storeImpl) GetByIDs(ctx context.Context, ids []string) ([]*storage.Image, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "Image")

	return pgutils.Retry2(ctx, func() ([]*storage.Image, error) {
		return s.retryableGetByIDs(ctx, ids)
	})
}

func (s *storeImpl) retryableGetByIDs(ctx context.Context, ids []string) ([]*storage.Image, error) {
	conn, release, err := s.acquireConn(ctx, ops.GetMany, "Image")
	if err != nil {
		return nil, err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}

	elems := make([]*storage.Image, 0, len(ids))
	for _, id := range ids {
		msg, found, err := s.getFullImage(ctx, tx, id)
		if err != nil {
			// No changes are made to the database, so COMMIT or ROLLBACK have the same effect.
			if err := tx.Commit(ctx); err != nil {
				return nil, err
			}
			return nil, err
		}
		if !found {
			continue
		}
		elems = append(elems, msg)
	}

	// No changes are made to the database, so COMMIT or ROLLBACK have the same effect.
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return elems, nil
}

// WalkByQuery returns the objects specified by the query
func (s *storeImpl) WalkByQuery(ctx context.Context, q *v1.Query, fn func(image *storage.Image) error) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.WalkByQuery, "Image")

	conn, release, err := s.acquireConn(ctx, ops.WalkByQuery, "Image")
	if err != nil {
		return err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			log.Errorf("error rolling back: %v", err)
		}
	}()

	callback := func(image *storage.Image) error {
		err := s.populateImage(ctx, tx, image)
		if err != nil {
			return errors.Wrap(err, "populate image")
		}
		if err := fn(image); err != nil {
			return errors.Wrap(err, "failed to process image")
		}
		return nil
	}
	err = pgSearch.RunCursorQueryForSchemaFn(ctx, pkgSchema.ImagesSchema, q, s.db, callback)
	if err != nil {
		return errors.Wrap(err, "cursor by query")
	}
	return nil
}

// GetImageMetadata returns the image without scan/component data.
func (s *storeImpl) GetImageMetadata(ctx context.Context, id string) (*storage.Image, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "ImageMetadata")

	return pgutils.Retry3(ctx, func() (*storage.Image, bool, error) {
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
	if err := msg.UnmarshalVTUnsafe(data); err != nil {
		return nil, false, err
	}
	return &msg, true, nil
}

// GetManyImageMetadata returns images without scan/component data.
func (s *storeImpl) GetManyImageMetadata(ctx context.Context, ids []string) ([]*storage.Image, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "Image")

	return pgutils.Retry2(ctx, func() ([]*storage.Image, error) {
		return s.retryableGetManyImageMetadata(ctx, ids)
	})
}

func (s *storeImpl) retryableGetManyImageMetadata(ctx context.Context, ids []string) ([]*storage.Image, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ImageSHA, ids...).ProtoQuery()
	return pgSearch.RunGetManyQueryForSchema[storage.Image](ctx, schema, q, s.db)
}

// GetImagesRiskView retrieves an image id and risk score to initialize rankers
func (s *storeImpl) GetImagesRiskView(ctx context.Context, q *v1.Query) ([]*views.ImageRiskView, error) {
	// The entire image is not needed to initialize the ranker.  We only need the image id and risk score.
	var results []*views.ImageRiskView
	results, err := pgSearch.RunSelectRequestForSchema[views.ImageRiskView](ctx, s.db, pkgSchema.ImagesSchema, q)
	if err != nil {
		log.Errorf("unable to initialize image ranking: %v", err)
	}

	return results, err
}

func (s *storeImpl) UpdateVulnState(ctx context.Context, cve string, imageIDs []string, state storage.VulnerabilityState) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Update, "UpdateVulnState")

	return pgutils.Retry(ctx, func() error {
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

	// Collect stored cves for the image.
	rows, err := tx.Query(ctx, "SELECT serialized FROM "+imageComponentsV2CVEsTable+" "+
		"WHERE "+imageComponentsV2CVEsTable+".imageid = ANY($1::text[]) AND "+imageComponentsV2CVEsTable+".cvebaseinfo_cve = $2", imageIDs, cve)
	if err != nil {
		return err
	}
	imageCVEs, err := pgutils.ScanRows[storage.ImageCVEV2, *storage.ImageCVEV2](rows)
	if err != nil {
		return err
	}

	// Update state.
	cveIDs := make([]string, 0, len(imageCVEs))
	for _, compCVE := range imageCVEs {
		compCVE.State = state
		cveIDs = append(cveIDs, compCVE.GetId())
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
		err = s.updateCVEVulnState(ctx, tx, imageCVEs...)
		if err != nil {
			if err := tx.Rollback(ctx); err != nil {
				return err
			}
			return err
		}
		return tx.Commit(ctx)
	})
}

func (s *storeImpl) updateCVEVulnState(ctx context.Context, tx *postgres.Tx, objs ...*storage.ImageCVEV2) error {
	batch := &pgx.Batch{}
	for _, obj := range objs {
		if err := s.insertIntoImageComponentV2Cves(batch, obj); err != nil {
			return errors.Wrap(err, "error on insertInto")
		}
	}
	batchResults := tx.SendBatch(ctx, batch)
	if err := batchResults.Close(); err != nil {
		return errors.Wrap(err, "closing batch")
	}
	return nil
}

func (s *storeImpl) insertIntoImageComponentV2Cves(batch *pgx.Batch, obj *storage.ImageCVEV2) error {
	serialized, marshalErr := obj.MarshalVT()
	if marshalErr != nil {
		return marshalErr
	}

	values := []interface{}{
		obj.GetId(),
		obj.GetImageId(),
		obj.GetCveBaseInfo().GetCve(),
		protocompat.NilOrTime(obj.GetCveBaseInfo().GetPublishedOn()),
		protocompat.NilOrTime(obj.GetCveBaseInfo().GetCreatedAt()),
		obj.GetCveBaseInfo().GetEpss().GetEpssProbability(),
		obj.GetCvss(),
		obj.GetSeverity(),
		obj.GetImpactScore(),
		obj.GetNvdcvss(),
		protocompat.NilOrTime(obj.GetFirstImageOccurrence()),
		obj.GetState(),
		obj.GetIsFixable(),
		obj.GetFixedBy(),
		obj.GetComponentId(),
		obj.GetAdvisory().GetName(),
		serialized,
	}

	finalStr := "INSERT INTO image_cves_v2 (Id, ImageId, CveBaseInfo_Cve, CveBaseInfo_PublishedOn, CveBaseInfo_CreatedAt, CveBaseInfo_Epss_EpssProbability, Cvss, Severity, ImpactScore, Nvdcvss, FirstImageOccurrence, State, IsFixable, FixedBy, ComponentId, advisory_name, serialized) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,$17) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, ImageId = EXCLUDED.ImageId, CveBaseInfo_Cve = EXCLUDED.CveBaseInfo_Cve, CveBaseInfo_PublishedOn = EXCLUDED.CveBaseInfo_PublishedOn, CveBaseInfo_CreatedAt = EXCLUDED.CveBaseInfo_CreatedAt, CveBaseInfo_Epss_EpssProbability = EXCLUDED.CveBaseInfo_Epss_EpssProbability, Cvss = EXCLUDED.Cvss, Severity = EXCLUDED.Severity, ImpactScore = EXCLUDED.ImpactScore, Nvdcvss = EXCLUDED.Nvdcvss, FirstImageOccurrence = EXCLUDED.FirstImageOccurrence, State = EXCLUDED.State, IsFixable = EXCLUDED.IsFixable, FixedBy = EXCLUDED.FixedBy, ComponentId = EXCLUDED.ComponentId, advisory_name = EXCLUDED.advisory_name, serialized = EXCLUDED.serialized"
	batch.Queue(finalStr, values...)

	return nil
}

func gatherKeys(parts *imagePartsAsSlice) [][]byte {
	// We only need to collect image, component keys because vulns are a child of component and the component
	// datastore does not support upserts and deletes of vulns.
	keys := make([][]byte, 0, len(parts.componentsV2))
	keys = append(keys, []byte(parts.image.GetId()))
	for _, component := range parts.componentsV2 {
		keys = append(keys, []byte(component.GetId()))
	}
	return keys
}

func (s *storeImpl) isComponentsTableEmpty(ctx context.Context, imageID string) (bool, error) {
	q := search.NewQueryBuilder().AddExactMatches(search.ImageSHA, imageID).ProtoQuery()
	count, err := pgSearch.RunCountRequestForSchema(ctx, pkgSchema.ImageComponentV2Schema, q, s.db)
	if err != nil {
		return false, err
	}
	return count < 1, nil
}
