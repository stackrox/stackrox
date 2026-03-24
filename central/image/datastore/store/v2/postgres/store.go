package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/hashstructure"
	convertutils "github.com/stackrox/rox/central/cve/converter/utils"
	cvev2pgstore "github.com/stackrox/rox/central/cve/image/v2/datastore/store/postgres"
	"github.com/stackrox/rox/central/image/datastore/store"
	"github.com/stackrox/rox/central/image/datastore/store/common/v2"
	"github.com/stackrox/rox/central/image/views"
	"github.com/stackrox/rox/central/metrics"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/baseimage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	pgSearch "github.com/stackrox/rox/pkg/search/postgres"
	"github.com/stackrox/rox/pkg/search/sortfields"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	imagesTable            = pkgSchema.ImagesTableName
	imageComponentsV2Table = pkgSchema.ImageComponentV2TableName

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

	// Insert CVEs into the normalized cves and component_cve_edges tables.
	// first_system_occurrence is preserved by the DB's ON CONFLICT clause in component_cve_edges.
	return s.upsertCVEsToNormalizedTables(ctx, parts.cvesV2, iTime)
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
		"layertype",
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
			obj.GetLayerType(),
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

// upsertCVEsToNormalizedTables inserts CVEs into the normalized cves and component_cve_edges tables.
// CVE IDs are derived deterministically from the content hash so ON CONFLICT(id) is idempotent.
// first_system_occurrence is set from the CVE's created_at on first insert and preserved by the DB's
// ON CONFLICT clause on subsequent upserts.
func (s *storeImpl) upsertCVEsToNormalizedTables(
	ctx context.Context,
	cvesV2 []*storage.ImageCVEV2,
	iTime time.Time,
) error {
	cveStore := cvev2pgstore.NewCombined(s.db)

	// Group CVEs by component ID to track which CVEs belong to each component.
	componentCVEMap := make(map[string][]string)
	for _, cveV2 := range cvesV2 {
		// Extract CVSS V3 score.
		var cvssV3 float32
		if v3 := cveV2.GetCveBaseInfo().GetCvssV3(); v3 != nil {
			cvssV3 = v3.GetScore()
		}

		// Determine primary source from CVSS metrics or datasource.
		source := primarySourceFromImageCVE(cveV2)
		severity := convertutils.SeverityToString(cveV2.GetSeverity())

		// Compute content hash and derive deterministic UUID.
		contentHash := convertutils.ComputeCVEContentHash(
			cveV2.GetCveBaseInfo().GetCve(), source, severity, cvssV3, cveV2.GetCveBaseInfo().GetSummary(),
		)
		cveID := convertutils.DeterministicCVEID(contentHash)

		// Build NormalizedCVE proto.
		normalizedCVE := &storage.NormalizedCVE{
			Id:           cveID,
			CveName:      cveV2.GetCveBaseInfo().GetCve(),
			Source:       source,
			Severity:     severity,
			CvssV3:       cvssV3,
			NvdCvssV3:    cveV2.GetNvdcvss(),
			Summary:      cveV2.GetCveBaseInfo().GetSummary(),
			Link:         cveV2.GetCveBaseInfo().GetLink(),
			PublishedOn:  cveV2.GetCveBaseInfo().GetPublishedOn(),
			AdvisoryName: cveV2.GetAdvisory().GetName(),
			AdvisoryLink: cveV2.GetAdvisory().GetLink(),
			ContentHash:  contentHash,
			CreatedAt:    cveV2.GetCveBaseInfo().GetCreatedAt(),
		}
		if v2 := cveV2.GetCveBaseInfo().GetCvssV2(); v2 != nil {
			normalizedCVE.CvssV2 = v2.GetScore()
		}

		// Single-phase upsert: same content_hash → same cveID → idempotent ON CONFLICT(id) DO UPDATE.
		if err := cveStore.Upsert(ctx, normalizedCVE); err != nil {
			return errors.Wrapf(err, "upserting NormalizedCVE %q for component %q", cveV2.GetCveBaseInfo().GetCve(), cveV2.GetComponentId())
		}

		// Track this CVE for the component.
		componentCVEMap[cveV2.GetComponentId()] = append(componentCVEMap[cveV2.GetComponentId()], cveID)

		// Set first_system_occurrence from the CVE's creation timestamp, falling back to scan time.
		// The DB's ON CONFLICT clause preserves the earliest value across subsequent upserts.
		firstSysOccurrenceTS := cveV2.GetCveBaseInfo().GetCreatedAt()
		if firstSysOccurrenceTS == nil {
			firstSysOccurrenceTS = timestamppb.New(iTime)
		}

		// Build NormalizedComponentCVEEdge proto.
		edge := &storage.NormalizedComponentCVEEdge{
			ComponentId:           cveV2.GetComponentId(),
			CveId:                 cveID,
			IsFixable:             cveV2.GetIsFixable(),
			FixedBy:               cveV2.GetFixedBy(),
			State:                 cveV2.GetState().String(),
			FirstSystemOccurrence: firstSysOccurrenceTS,
			FixAvailableAt:        cveV2.GetFixAvailableTimestamp(),
		}

		// Upsert edge.
		if err := cveStore.UpsertEdge(ctx, edge); err != nil {
			return errors.Wrapf(err, "upserting edge for component %q and CVE %q", cveV2.GetComponentId(), cveID)
		}
	}

	// Delete stale edges for each component.
	for componentID, newCVEIDs := range componentCVEMap {
		if err := cveStore.DeleteStaleEdges(ctx, componentID, newCVEIDs); err != nil {
			return errors.Wrapf(err, "deleting stale edges for component %q", componentID)
		}
	}

	return nil
}

// primarySourceFromImageCVE returns the primary source string for an ImageCVEV2
// based on its CVSS metrics, preferring NVD > RED_HAT > OSV > UNKNOWN.
func primarySourceFromImageCVE(cveV2 *storage.ImageCVEV2) string {
	priority := map[storage.Source]int{
		storage.Source_SOURCE_NVD:     3,
		storage.Source_SOURCE_RED_HAT: 2,
		storage.Source_SOURCE_OSV:     1,
	}
	best := storage.Source_SOURCE_UNKNOWN
	bestPrio := -1
	for _, m := range cveV2.GetCveBaseInfo().GetCvssMetrics() {
		if p := priority[m.GetSource()]; p > bestPrio {
			best = m.GetSource()
			bestPrio = p
		}
	}
	return convertutils.SourceToString(best)
}

// stringPtr returns a pointer to a string, or nil if the string is empty.
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// float32Ptr returns a pointer to a float32, or nil if the value is zero.
func float32Ptr(f float32) *float32 {
	if f == 0 {
		return nil
	}
	return &f
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
		image.Scan = oldImage.GetScan()
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
	hash, err := hashstructure.Hash(hashWrapper{scan.GetComponents()}, &hashstructure.HashOptions{ZeroNil: true})
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

	if features.BaseImageDetection.Enabled() {
		// Re-verify base images when base image detection is enabled:
		// 1. Legacy images may lack base image info if the feature was enabled after they were scanned.
		// 2. User-provided base images may change over time.
		scanUpdated = scanUpdated || baseimage.BaseImagesUpdated(oldImage.GetBaseImageInfo(), obj.GetBaseImageInfo())
	}

	splitParts, err := common.SplitV2(obj, scanUpdated)
	if err != nil {
		return err
	}
	imageParts := getPartsAsSlice(splitParts)
	keys := gatherKeys(imageParts)

	return s.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet(keys...), func() error {
		tx, ctx, err := s.begin(ctx)
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

	q = applyDefaultSort(q)

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
	tx, ctx, err := s.begin(ctx)
	if err != nil {
		return nil, false, err
	}
	defer postgres.FinishReadOnlyTransaction(tx)

	image, found, err := s.getFullImage(ctx, tx, id)

	return image, found, err
}

func (s *storeImpl) populateImage(ctx context.Context, tx *postgres.Tx, image *storage.Image) error {
	components, err := getImageComponents(ctx, tx, image.GetId())
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

func (s *storeImpl) begin(ctx context.Context) (*postgres.Tx, context.Context, error) {
	return postgres.GetTransaction(ctx, s.db)
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

// getImageComponentCVEs reads normalized CVEs for a component from the cves and
// component_cve_edges tables and reconstructs ImageCVEV2 objects for the merge path.
func getImageComponentCVEs(ctx context.Context, tx *postgres.Tx, componentID string) ([]*storage.ImageCVEV2, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "ImageCVEsV2")

	const querySQL = `
		SELECT c.serialized, e.is_fixable, e.fixed_by, e.state, e.first_system_occurrence, e.fix_available_at
		FROM cves c
		JOIN component_cve_edges e ON c.id = e.cve_id
		WHERE e.component_id = $1
	`
	rows, err := tx.Query(ctx, querySQL, componentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*storage.ImageCVEV2
	for rows.Next() {
		var serialized []byte
		var isFixable bool
		var fixedBy, stateStr string
		var firstSysOcc, fixAvailAt *time.Time
		if err := rows.Scan(&serialized, &isFixable, &fixedBy, &stateStr, &firstSysOcc, &fixAvailAt); err != nil {
			return nil, err
		}

		n := new(storage.NormalizedCVE)
		if err := n.UnmarshalVTUnsafe(serialized); err != nil {
			return nil, err
		}

		vulnState := storage.VulnerabilityState_OBSERVED
		switch stateStr {
		case "DEFERRED":
			vulnState = storage.VulnerabilityState_DEFERRED
		case "FALSE_POSITIVE":
			vulnState = storage.VulnerabilityState_FALSE_POSITIVE
		}

		cveV2 := &storage.ImageCVEV2{
			ComponentId: componentID,
			CveBaseInfo: &storage.CVEInfo{
				Cve:         n.GetCveName(),
				Summary:     n.GetSummary(),
				Link:        n.GetLink(),
				PublishedOn: n.GetPublishedOn(),
				CreatedAt:   n.GetCreatedAt(),
			},
			Cvss:      n.GetCvssV3(),
			Severity:  convertutils.SeverityFromString(n.GetSeverity()),
			Nvdcvss:   n.GetNvdCvssV3(),
			IsFixable: isFixable,
			State:     vulnState,
		}
		if isFixable && fixedBy != "" {
			cveV2.HasFixedBy = &storage.ImageCVEV2_FixedBy{FixedBy: fixedBy}
		}
		if n.GetAdvisoryName() != "" || n.GetAdvisoryLink() != "" {
			cveV2.Advisory = &storage.Advisory{Name: n.GetAdvisoryName(), Link: n.GetAdvisoryLink()}
		}
		if firstSysOcc != nil {
			cveV2.FirstImageOccurrence = timestamppb.New(*firstSysOcc)
		}
		if fixAvailAt != nil {
			cveV2.FixAvailableTimestamp = timestamppb.New(*fixAvailAt)
		}
		result = append(result, cveV2)
	}
	return result, rows.Err()
}

// Delete removes the specified ID from the store.
func (s *storeImpl) Delete(ctx context.Context, id string) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "Image")

	return pgutils.Retry(ctx, func() error {
		return s.retryableDelete(ctx, id)
	})
}

func (s *storeImpl) retryableDelete(ctx context.Context, id string) error {
	tx, ctx, err := s.begin(ctx)
	if err != nil {
		return err
	}

	if err := s.deleteImageTree(ctx, tx, id); err != nil {
		if errTx := tx.Rollback(ctx); errTx != nil {
			return errors.Wrapf(errTx, "rollbacking transaction due to previous error: %v", err)
		}
		return errors.Wrap(err, "deleting image tree")
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
	tx, ctx, err := s.begin(ctx)
	if err != nil {
		return nil, err
	}
	defer postgres.FinishReadOnlyTransaction(tx)

	elems := make([]*storage.Image, 0, len(ids))
	for _, id := range ids {
		msg, found, err := s.getFullImage(ctx, tx, id)
		if err != nil {
			return nil, err
		}
		if !found {
			continue
		}
		elems = append(elems, msg)
	}

	return elems, nil
}

// WalkByQuery returns the objects specified by the query
func (s *storeImpl) WalkByQuery(ctx context.Context, q *v1.Query, fn func(image *storage.Image) error) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.WalkByQuery, "Image")

	q = applyDefaultSort(q)

	tx, ctx, err := s.begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err := tx.Commit(ctx); err != nil {
			log.Errorf("error comitting transaction: %v", err)
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

func (s *storeImpl) WalkMetadataByQuery(ctx context.Context, q *v1.Query, fn func(img *storage.Image) error) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.WalkMetadataByQuery, "Image")

	q = applyDefaultSort(q)

	err := pgSearch.RunCursorQueryForSchemaFn(ctx, pkgSchema.ImagesSchema, q, s.db, fn)
	if err != nil {
		return errors.Wrap(err, "cursor by query")
	}
	return nil
}

// GetImageMetadata returns the image without scan/component data.
func (s *storeImpl) GetImageMetadata(ctx context.Context, id string) (*storage.Image, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "ImageMetadata")

	imageMetadata, err := pgutils.Retry2(ctx, func() ([]*storage.Image, error) {
		return s.retryableGetManyImageMetadata(ctx, []string{id})
	})
	if err != nil || len(imageMetadata) == 0 {
		return nil, false, err
	}
	return imageMetadata[0], true, nil
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
	results := make([]*views.ImageRiskView, 0, paginated.GetLimit(q.GetPagination().GetLimit(), 100))
	err := pgSearch.RunSelectRequestForSchemaFn[views.ImageRiskView](ctx, s.db, pkgSchema.ImagesSchema, q, func(r *views.ImageRiskView) error {
		results = append(results, r)
		return nil
	})
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

	// Update state in component_cve_edges for all components belonging to the given images that
	// are linked to the CVE with the given name.
	const updateSQL = `
		UPDATE component_cve_edges e
		   SET state = $1
		WHERE e.cve_id IN (SELECT id FROM cves WHERE cve_name = $2)
		  AND e.component_id IN (
		      SELECT ic.id FROM image_component_v2 ic WHERE ic.imageid = ANY($3::text[])
		  )
	`
	keys := make([][]byte, 0, len(imageIDs))
	for _, id := range imageIDs {
		keys = append(keys, []byte(id))
	}

	return s.keyFence.DoStatusWithLock(concurrency.DiscreteKeySet(keys...), func() error {
		_, err := s.db.Exec(ctx, updateSQL, state.String(), cve, imageIDs)
		return err
	})
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

func applyDefaultSort(q *v1.Query) *v1.Query {
	q = sortfields.TransformSortOptions(q, pkgSchema.ImagesSchema.OptionsMap)

	defaultSortOption := &v1.QuerySortOption{
		Field: search.LastUpdatedTime.String(),
	}
	// Add pagination sort order if needed.
	return paginated.FillDefaultSortOption(q, defaultSortOption.CloneVT())
}

// For tesing only
// NewForTest returns a new store instance for testing
func NewForTest(_ testing.TB, db postgres.DB, noUpdateTimestamps bool, keyFence concurrency.KeyFence) store.Store {
	return &storeImpl{
		db:                 db,
		noUpdateTimestamps: noUpdateTimestamps,
		keyFence:           keyFence,
	}
}
