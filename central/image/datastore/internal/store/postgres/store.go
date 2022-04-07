package postgres

import (
	"context"
	"reflect"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/image/datastore/internal/store/common"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	baseTable = "images"
	// For now hard-code the table names. This is a workaround until schemas can become truly sharable.
	imageComponentRelationTable = "image_component_relations"
	imageComponentTable         = "image_components"
	componentCVERelationTable   = "image_component_cve_relations"
	imageCVETable               = "image_cves"

	countStmt  = "SELECT COUNT(*) FROM images"
	existsStmt = "SELECT EXISTS(SELECT 1 FROM images WHERE Id = $1)"

	getStmt     = "SELECT serialized FROM images WHERE Id = $1"
	deleteStmt  = "DELETE FROM images WHERE Id = $1"
	walkStmt    = "SELECT serialized FROM images"
	getIDsStmt  = "SELECT Id FROM images"
	getManyStmt = "SELECT serialized FROM images WHERE Id = ANY($1::text[])"

	deleteManyStmt = "DELETE FROM images WHERE Id = ANY($1::text[])"

	getImageComponentEdgesStmt = "SELECT serialized FROM image_component_relations WHERE imageid = $1 order by imagecomponentid asc"
	getImageComponentStmt      = "SELECT serialized FROM image_components, image_component_relations WHERE image_component_relations.imageid = $1 AND image_component_relations.imagecomponentid = image_components.id order by image_components.id asc"
	getComponentCVEEdgesStmt   = "SELECT serialized FROM component_cve_relations, image_component_relations WHERE image_component_relations.imageid = $1 AND image_component_relations.imagecomponentid = component_cve_relations.imagecomponentid order by component_cve_relations.imagecomponentid asc"
	getImageCVEStmt            = "SELECT serialized FROM image_cves, component_cve_relations, image_component_relations WHERE image_component_relations.imageid = $1 AND image_component_relations.imagecomponentid = component_cve_relations.imagecomponentid AND component_cve_relations.cveid = image_cves.id group by component_cve_relations.imagecomponentid order by component_cve_relations.imagecomponentid asc"

	batchAfter = 100

	// using copyFrom, we may not even want to batch.  It would probably be simpler
	// to deal with failures if we just sent it all.  Something to think about as we
	// proceed and move into more e2e and larger performance testing
	batchSize = 10000
)

var (
	schema = walker.Walk(reflect.TypeOf((*storage.Image)(nil)), baseTable)
	log    = logging.LoggerForModule()
)

func init() {
	globaldb.RegisterTable(schema)
}

type Store interface {
	Count(ctx context.Context) (int, error)
	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.Image, bool, error)
	Upsert(ctx context.Context, obj *storage.Image) error
	UpsertMany(ctx context.Context, objs []*storage.Image) error
	Delete(ctx context.Context, id string) error
	GetIDs(ctx context.Context) ([]string, error)
	GetMany(ctx context.Context, ids []string) ([]*storage.Image, []int, error)
	DeleteMany(ctx context.Context, ids []string) error

	AckKeysIndexed(ctx context.Context, keys ...string) error
	GetKeysToIndex(ctx context.Context) ([]string, error)

	GetImageMetadata(_ context.Context, id string) (*storage.Image, bool, error)
}

// New returns a new Store instance using the provided sql instance.
func New(ctx context.Context, db *pgxpool.Pool) Store {
	createTableImages(ctx, db)

	return &storeImpl{
		db: db,
	}
}

type storeImpl struct {
	db *pgxpool.Pool
}

func createTableImages(ctx context.Context, db *pgxpool.Pool) {
	table := `
create table if not exists images (
    Id varchar,
    Name_Registry varchar,
    Name_Remote varchar,
    Name_Tag varchar,
    Name_FullName varchar,
    Metadata_V1_Created timestamp,
    Metadata_V1_User varchar,
    Metadata_V1_Command text[],
    Metadata_V1_Entrypoint text[],
    Metadata_V1_Volumes text[],
    Metadata_V1_Labels jsonb,
    Scan_ScanTime timestamp,
    Scan_OperatingSystem varchar,
    Signature_Fetched timestamp,
    Components integer,
    Cves integer,
    FixableCves integer,
    LastUpdated timestamp,
    RiskScore numeric,
    TopCvss numeric,
    serialized bytea,
    PRIMARY KEY(Id)
)
`

	_, err := db.Exec(ctx, table)
	if err != nil {
		log.Panicf("Error creating table %s: %v", table, err)
	}

	indexes := []string{}
	for _, index := range indexes {
		if _, err := db.Exec(ctx, index); err != nil {
			log.Panicf("Error creating index %s: %v", index, err)
		}
	}

	createTableImagesLayers(ctx, db)
}

func createTableImagesLayers(ctx context.Context, db *pgxpool.Pool) {
	table := `
create table if not exists images_Layers (
    images_Id varchar,
    idx integer,
    Instruction varchar,
    Value varchar,
    PRIMARY KEY(images_Id, idx),
    CONSTRAINT fk_parent_table_0 FOREIGN KEY (images_Id) REFERENCES images(Id) ON DELETE CASCADE
)
`

	_, err := db.Exec(ctx, table)
	if err != nil {
		log.Panicf("Error creating table %s: %v", table, err)
	}

	indexes := []string{

		"create index if not exists imagesLayers_idx on images_Layers using btree(idx)",
	}
	for _, index := range indexes {
		if _, err := db.Exec(ctx, index); err != nil {
			log.Panicf("Error creating index %s: %v", index, err)
		}
	}

}

func insertIntoImages(ctx context.Context, tx pgx.Tx, obj *storage.Image, scanUpdated bool) error {

	// TODO: Determine the feasibility of breaking the serialized object. For now, upsert the full image object into image store.

	cloned := obj.Clone()
	cloned.Scan.Components = nil
	serialized, marshalErr := obj.Marshal()
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

	finalStr := "INSERT INTO images (Id, Name_Registry, Name_Remote, Name_Tag, Name_FullName, Metadata_V1_Created, Metadata_V1_User, Metadata_V1_Command, Metadata_V1_Entrypoint, Metadata_V1_Volumes, Metadata_V1_Labels, Scan_ScanTime, Scan_OperatingSystem, Signature_Fetched, Components, Cves, FixableCves, LastUpdated, RiskScore, TopCvss, serialized) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21) ON CONFLICT(Id) DO UPDATE SET Id = EXCLUDED.Id, Name_Registry = EXCLUDED.Name_Registry, Name_Remote = EXCLUDED.Name_Remote, Name_Tag = EXCLUDED.Name_Tag, Name_FullName = EXCLUDED.Name_FullName, Metadata_V1_Created = EXCLUDED.Metadata_V1_Created, Metadata_V1_User = EXCLUDED.Metadata_V1_User, Metadata_V1_Command = EXCLUDED.Metadata_V1_Command, Metadata_V1_Entrypoint = EXCLUDED.Metadata_V1_Entrypoint, Metadata_V1_Volumes = EXCLUDED.Metadata_V1_Volumes, Metadata_V1_Labels = EXCLUDED.Metadata_V1_Labels, Scan_ScanTime = EXCLUDED.Scan_ScanTime, Scan_OperatingSystem = EXCLUDED.Scan_OperatingSystem, Signature_Fetched = EXCLUDED.Signature_Fetched, Components = EXCLUDED.Components, Cves = EXCLUDED.Cves, FixableCves = EXCLUDED.FixableCves, LastUpdated = EXCLUDED.LastUpdated, RiskScore = EXCLUDED.RiskScore, TopCvss = EXCLUDED.TopCvss, serialized = EXCLUDED.serialized"
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

	components, vulns, imageComponentRelations, componentCVERelations := postProcessSplit(common.Split(obj, scanUpdated))
	if err := copyFromImageComponents(ctx, tx, components...); err != nil {
		return err
	}
	if err := copyFromImageComponentRelations(ctx, tx, imageComponentRelations...); err != nil {
		return err
	}
	if err := copyFromImageCves(ctx, tx, vulns...); err != nil {
		return err
	}
	return copyFromImageComponentCVERelations(ctx, tx, componentCVERelations...)
}

func postProcessSplit(parts common.ImageParts) ([]*storage.ImageComponent, []*storage.CVE, []*storage.ImageComponentEdge, []*storage.ComponentCVEEdge) {
	components := make([]*storage.ImageComponent, 0, len(parts.Children))
	imageComponentRelations := make([]*storage.ImageComponentEdge, 0, len(parts.Children))
	vulnMap := make(map[string]*storage.CVE)
	var componentCVERelations []*storage.ComponentCVEEdge
	for _, child := range parts.Children {
		components = append(components, child.Component)
		imageComponentRelations = append(imageComponentRelations, child.Edge)
		for _, gChild := range child.Children {
			componentCVERelations = append(componentCVERelations, gChild.Edge)
			vulnMap[gChild.Cve.GetId()] = gChild.Cve
		}
	}
	vulns := make([]*storage.CVE, 0, len(vulnMap))
	for _, vuln := range vulnMap {
		vulns = append(vulns, vuln)
	}
	return components, vulns, imageComponentRelations, componentCVERelations
}

func insertIntoImagesLayers(ctx context.Context, tx pgx.Tx, obj *storage.ImageLayer, images_Id string, idx int) error {

	values := []interface{}{
		// parent primary keys start
		images_Id,
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

		"operatingsystem",

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

			obj.GetOperatingSystem(),

			serialized,
		})

		// Add the id to be deleted.
		deletes = append(deletes, obj.GetId())

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// Copy does not upsert so have to delete first.
			_, err = tx.Exec(ctx, deleteManyStmt, deletes)
			if err != nil {
				return err
			}

			// clear the inserts for the next batch
			deletes = nil

			_, err = tx.CopyFrom(ctx, pgx.Identifier{imageComponentTable}, copyCols, pgx.CopyFromRows(inputRows))

			if err != nil {
				return err
			}

			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return err
}

func copyFromImageComponentRelations(ctx context.Context, tx pgx.Tx, objs ...*storage.ImageComponentEdge) error {
	inputRows := [][]interface{}{}

	var err error

	var deletes []string

	copyCols := []string{
		"id",
		"haslayerindex",
		"location",
		"imageid",
		"imagecomponentid",
	}

	for idx, obj := range objs {

		serialized, marshalErr := obj.Marshal()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{
			obj.GetId(),
			obj.GetHasLayerIndex(),
			obj.GetLocation(),
			obj.GetImageId(),
			obj.GetImageComponentId(),
			serialized,
		})

		// Add the id to be deleted.
		deletes = append(deletes, obj.GetId())

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// Copy does not upsert so have to delete first.
			_, err = tx.Exec(ctx, deleteManyStmt, deletes)
			if err != nil {
				return err
			}

			// clear the inserts for the next batch
			deletes = nil

			_, err = tx.CopyFrom(ctx, pgx.Identifier{imageComponentRelationTable}, copyCols, pgx.CopyFromRows(inputRows))

			if err != nil {
				return err
			}

			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return err
}

func copyFromImageCves(ctx context.Context, tx pgx.Tx, objs ...*storage.CVE) error {
	inputRows := [][]interface{}{}

	var err error

	// This is a copy so first we must delete the rows and re-add them
	// Which is essentially the desired behaviour of an upsert.
	var deletes []string

	copyCols := []string{

		"id",

		"operatingsystem",

		"cvss",

		"impactscore",

		"publishedon",

		"createdat",

		"suppressed",

		"suppressexpiry",

		"severity",

		"serialized",
	}

	for idx, obj := range objs {
		serialized, marshalErr := obj.Marshal()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{

			obj.GetId(),

			obj.GetOperatingSystem(),

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
			_, err = tx.Exec(ctx, deleteManyStmt, deletes)
			if err != nil {
				return err
			}
			// Clear the inserts for the next batch.
			deletes = nil

			_, err = tx.CopyFrom(ctx, pgx.Identifier{imageCVETable}, copyCols, pgx.CopyFromRows(inputRows))

			if err != nil {
				return err
			}

			// Clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return err
}

func copyFromImageComponentCVERelations(ctx context.Context, tx pgx.Tx, objs ...*storage.ComponentCVEEdge) error {
	inputRows := [][]interface{}{}

	var err error

	var deletes []string

	copyCols := []string{
		"id",
		"isfixable",
		"hasfixedby",
		"imagecomponentid",
		"cveid",
		"cveoperatingsystem",
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
			obj.GetCveId(),
			obj.GetCveOperatingSystem(),
			serialized,
		})

		// Add the id to be deleted.
		deletes = append(deletes, obj.GetId())

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// Copy does not upsert so have to delete first.
			_, err = tx.Exec(ctx, deleteManyStmt, deletes)
			if err != nil {
				return err
			}

			// Clear the inserts for the next batch
			deletes = nil

			_, err = tx.CopyFrom(ctx, pgx.Identifier{componentCVERelationTable}, copyCols, pgx.CopyFromRows(inputRows))

			if err != nil {
				return err
			}

			// Clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return err
}

func (s *storeImpl) copyFromImages(ctx context.Context, tx pgx.Tx, objs ...*storage.Image) error {
	inputRows := [][]interface{}{}

	var err error

	// This is a copy so first we must delete the rows and re-add them
	// Which is essentially the desired behaviour of an upsert.
	var deletes []string

	copyCols := []string{

		"id",

		"name_registry",

		"name_remote",

		"name_tag",

		"name_fullname",

		"metadata_v1_created",

		"metadata_v1_user",

		"metadata_v1_command",

		"metadata_v1_entrypoint",

		"metadata_v1_volumes",

		"metadata_v1_labels",

		"scan_scantime",

		"scan_operatingsystem",

		"signature_fetched",

		"components",

		"cves",

		"fixablecves",

		"lastupdated",

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
		})

		// Add the id to be deleted.
		deletes = append(deletes, obj.GetId())

		// If we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			// Copy does not upsert so have to delete first.
			_, err = tx.Exec(ctx, deleteManyStmt, deletes)
			if err != nil {
				return err
			}

			// Clear the inserts for the next batch.
			deletes = nil

			_, err = tx.CopyFrom(ctx, pgx.Identifier{"images"}, copyCols, pgx.CopyFromRows(inputRows))

			if err != nil {
				return err
			}

			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	for _, obj := range objs {

		if err = s.copyFromImagesLayers(ctx, tx, obj.GetId(), obj.GetMetadata().GetV1().GetLayers()...); err != nil {
			return err
		}
	}

	return err
}

func (s *storeImpl) copyFromImagesLayers(ctx context.Context, tx pgx.Tx, images_Id string, objs ...*storage.ImageLayer) error {
	inputRows := [][]interface{}{}

	var err error

	copyCols := []string{

		"images_id",

		"idx",

		"instruction",

		"value",
	}

	for idx, obj := range objs {
		inputRows = append(inputRows, []interface{}{

			images_Id,

			idx,

			obj.GetInstruction(),

			obj.GetValue(),
		})

		// if we hit our batch size we need to push the data
		if (idx+1)%batchSize == 0 || idx == len(objs)-1 {
			_, err = tx.CopyFrom(ctx, pgx.Identifier{"images_layers"}, copyCols, pgx.CopyFromRows(inputRows))

			if err != nil {
				return err
			}

			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}

	return err
}

func (s *storeImpl) copyFrom(ctx context.Context, objs ...*storage.Image) error {
	conn, release, err := s.acquireConn(ctx, ops.Get, "Image")
	if err != nil {
		return err
	}
	defer release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	if err := s.copyFromImages(ctx, tx, objs...); err != nil {
		if err := tx.Rollback(ctx); err != nil {
			return err
		}
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (s *storeImpl) isUpdated(ctx context.Context, image *storage.Image) (bool, bool, error) {
	oldImage, found, err := s.Get(ctx, image.GetId())
	if err != nil || !found {
		return false, false, err
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

		metadataUpdated, scanUpdated, err := s.isUpdated(ctx, obj)
		if err != nil {
			return err
		}
		if !metadataUpdated && !scanUpdated {
			return nil
		}

		if err := insertIntoImages(ctx, tx, obj, scanUpdated); err != nil {
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

func (s *storeImpl) Upsert(ctx context.Context, obj *storage.Image) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Upsert, "Image")

	return s.upsert(ctx, obj)
}

func (s *storeImpl) UpsertMany(ctx context.Context, objs []*storage.Image) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.UpdateMany, "Image")

	if len(objs) < batchAfter {
		return s.upsert(ctx, objs...)
	} else {
		return s.copyFrom(ctx, objs...)
	}
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

	row := conn.QueryRow(ctx, getStmt, id)
	var data []byte
	if err := row.Scan(&data); err != nil {
		return nil, false, pgutils.ErrNilIfNoRows(err)
	}

	var image storage.Image
	if err := proto.Unmarshal(data, &image); err != nil {
		return nil, false, err
	}

	componentEdgeMap, err := s.getImageComponentEdges(ctx, conn, id)
	if err != nil {
		return nil, false, err
	}
	componentMap, err := s.getImageComponents(ctx, conn, id)
	if err != nil {
		return nil, false, err
	}
	componentCVEEdgeMap, err := s.getComponentCVEEdges(ctx, conn, id)
	if err != nil {
		return nil, false, err
	}
	cveMap, err := s.getCVEs(ctx, conn, id)
	if err != nil {
		return nil, false, err
	}
	if len(componentEdgeMap) != len(componentMap) {
		utils.Should(
			errors.Errorf("Number of component edges (%d) is not equal to number of components (%d) for image %s (id=%s)",
				len(componentEdgeMap), len(componentMap), image.GetName().GetFullName(), image.GetId()),
		)
	}
	if len(componentCVEEdgeMap) != len(cveMap) {
		utils.Should(
			errors.Errorf("Number of component-cve edges (%d) is not equal to number of cves (%d) for image %s (id=%s)",
				len(componentCVEEdgeMap), len(cveMap), image.GetName().GetFullName(), image.GetId()),
		)
	}

	imageParts := common.ImageParts{
		Image:     &image,
		ListImage: types.ConvertImageToListImage(&image),
		Children:  []common.ComponentParts{},
	}
	for componentID, componentEdge := range componentEdgeMap {
		imageParts.Children = append(imageParts.Children, common.ComponentParts{
			Edge:      componentEdge,
			Component: componentMap[componentID],
		})
	}
	for _, child := range imageParts.Children {
		for _, edge := range componentCVEEdgeMap[child.Component.GetId()] {
			child.Children = append(child.Children, common.CVEParts{
				Edge: edge,
				Cve:  cveMap[edge.GetCveId()],
			})
		}
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

func (s *storeImpl) getImageComponentEdges(ctx context.Context, conn *pgxpool.Conn, imageID string) (map[string]*storage.ImageComponentEdge, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "ImageComponentRelations")

	rows, err := conn.Query(ctx, getImageComponentEdgesStmt, imageID)
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

func (s *storeImpl) getImageComponents(ctx context.Context, conn *pgxpool.Conn, componentIDs ...string) (map[string]*storage.ImageComponent, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "ImageComponents")

	rows, err := conn.Query(ctx, getImageComponentStmt, componentIDs)
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

func (s *storeImpl) getComponentCVEEdges(ctx context.Context, conn *pgxpool.Conn, imageID string) (map[string][]*storage.ComponentCVEEdge, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "ImageComponentCVERelations")

	rows, err := conn.Query(ctx, getComponentCVEEdgesStmt, imageID)
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

func (s *storeImpl) getCVEs(ctx context.Context, conn *pgxpool.Conn, imageID string) (map[string]*storage.CVE, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetMany, "ImageCVEs")

	rows, err := conn.Query(ctx, getImageCVEStmt, imageID)
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

// Delete removes the specified ID from the store
func (s *storeImpl) Delete(ctx context.Context, id string) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Remove, "Image")

	conn, release, err := s.acquireConn(ctx, ops.Remove, "Image")
	if err != nil {
		return err
	}
	defer release()

	if _, err := conn.Exec(ctx, deleteStmt, id); err != nil {
		return err
	}
	return nil
}

// GetIDs returns all the IDs for the store
func (s *storeImpl) GetIDs(ctx context.Context) ([]string, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.GetAll, "storage.ImageIDs")

	rows, err := s.db.Query(ctx, getIDsStmt)
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

	rows, err := conn.Query(ctx, getManyStmt, ids)
	if err != nil {
		if err == pgx.ErrNoRows {
			missingIndices := make([]int, 0, len(ids))
			for i := range ids {
				missingIndices = append(missingIndices, i)
			}
			return nil, missingIndices, nil
		}
		return nil, nil, err
	}
	defer rows.Close()
	resultsByID := make(map[string]*storage.Image)
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, nil, err
		}
		msg := &storage.Image{}
		if err := proto.Unmarshal(data, msg); err != nil {
			return nil, nil, err
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

// Delete removes the specified IDs from the store
func (s *storeImpl) DeleteMany(ctx context.Context, ids []string) error {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.RemoveMany, "Image")

	conn, release, err := s.acquireConn(ctx, ops.RemoveMany, "Image")
	if err != nil {
		return err
	}
	defer release()
	if _, err := conn.Exec(ctx, deleteManyStmt, ids); err != nil {
		return err
	}
	return nil
}

//// Used for testing

func dropTableImages(ctx context.Context, db *pgxpool.Pool) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS images CASCADE")
	dropTableImagesLayers(ctx, db)

}

func dropTableImagesLayers(ctx context.Context, db *pgxpool.Pool) {
	_, _ = db.Exec(ctx, "DROP TABLE IF EXISTS images_Layers CASCADE")

}

func Destroy(ctx context.Context, db *pgxpool.Pool) {
	dropTableImages(ctx, db)
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

func (s *storeImpl) GetImageMetadata(ctx context.Context, id string) (*storage.Image, bool, error) {
	defer metrics.SetPostgresOperationDurationTime(time.Now(), ops.Get, "ImageMetadata")

	conn, release, err := s.acquireConn(ctx, ops.Get, "Image")
	if err != nil {
		return nil, false, err
	}
	defer release()

	row := conn.QueryRow(ctx, getStmt, id)
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
