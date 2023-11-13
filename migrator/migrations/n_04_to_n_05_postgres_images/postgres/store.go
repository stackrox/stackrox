// This file was originally generated with
// //go:generate cp ../../../../central/image/datastore/store/postgres/store.go store.go

package postgres

import (
	"context"

	"github.com/gogo/protobuf/proto"
	protoTypes "github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	pkgSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v73"
	"github.com/stackrox/rox/migrator/migrations/n_04_to_n_05_postgres_images/common/v2"
	"github.com/stackrox/rox/migrator/migrations/n_04_to_n_05_postgres_images/store"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	imagesTable              = pkgSchema.ImagesTableName
	imageComponentEdgesTable = pkgSchema.ImageComponentEdgesTableName
	imageComponentsTable     = pkgSchema.ImageComponentsTableName
	componentCVEEdgesTable   = pkgSchema.ImageComponentCveEdgesTableName
	imageCVEsTable           = pkgSchema.ImageCvesTableName
	imageCVEEdgesTable       = pkgSchema.ImageCveEdgesTableName

	countStmt = "SELECT COUNT(*) FROM " + imagesTable

	getImageMetaStmt = "SELECT serialized FROM " + imagesTable + " WHERE Id = $1"
	getImageIDsStmt  = "SELECT Id FROM " + imagesTable

	// using copyFrom, we may not even want to batch.  It would probably be simpler
	// to deal with failures if we just sent it all.  Something to think about as we
	// proceed and move into more e2e and larger performance testing
	batchSize = 500
)

var (
	storedComponents         = set.NewStringSet()
	storedComponentsCVEEdges = set.NewStringSet()
	storedCVEs               = set.NewStringSet()
)

// New returns a new Store instance using the provided sql instance.
func New(db postgres.DB, noUpdateTimestamps bool) store.Store {
	return &storeImpl{
		db:                 db,
		noUpdateTimestamps: noUpdateTimestamps,
	}
}

type storeImpl struct {
	db                 postgres.DB
	noUpdateTimestamps bool
}

func insertIntoImages(ctx context.Context, tx *postgres.Tx, obj *storage.Image, scanUpdated bool, iTime *protoTypes.Timestamp) error {
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

	for childIdx, child := range obj.GetMetadata().GetV1().GetLayers() {
		if err := insertIntoImagesLayers(ctx, tx, child, obj.GetId(), childIdx); err != nil {
			return err
		}
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

	copyCols := []string{
		"id",
		"name",
		"version",
		"operatingsystem",
		"source",
		"riskscore",
		"topcvss",
		"serialized",
	}

	for idx, obj := range objs {
		if !storedComponents.Add(obj.GetId()) {
			continue
		}

		serialized, marshalErr := obj.Marshal()
		if marshalErr != nil {
			return marshalErr
		}

		inputRows = append(inputRows, []interface{}{
			obj.GetId(),
			obj.GetName(),
			obj.GetVersion(),
			obj.GetOperatingSystem(),
			obj.GetSource(),
			obj.GetRiskScore(),
			obj.GetTopCvss(),
			serialized,
		})

		// if we hit our batch size we need to push the data
		if len(inputRows)%batchSize == 0 || idx == len(objs)-1 {
			_, err = tx.CopyFrom(ctx, pgx.Identifier{imageComponentsTable}, copyCols, pgx.CopyFromRows(inputRows))
			if err != nil {
				return err
			}

			// clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}
	if len(inputRows) != 0 {
		_, err = tx.CopyFrom(ctx, pgx.Identifier{imageComponentsTable}, copyCols, pgx.CopyFromRows(inputRows))
		if err != nil {
			return err
		}
	}

	return err
}

func copyFromImageComponentEdges(ctx context.Context, tx *postgres.Tx, objs ...*storage.ImageComponentEdge) error {
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

func copyFromImageCves(ctx context.Context, tx *postgres.Tx, _ *protoTypes.Timestamp, objs ...*storage.ImageCVE) error {
	inputRows := [][]interface{}{}

	var err error

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

	for idx, obj := range objs {
		if !storedCVEs.Add(obj.GetId()) {
			continue
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

		// if we hit our batch size we need to push the data
		if len(inputRows)%batchSize == 0 || idx == len(objs)-1 {
			_, err = tx.CopyFrom(ctx, pgx.Identifier{imageCVEsTable}, copyCols, pgx.CopyFromRows(inputRows))

			if err != nil {
				return err
			}

			// Clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}
	if len(inputRows) != 0 {
		_, err = tx.CopyFrom(ctx, pgx.Identifier{imageCVEsTable}, copyCols, pgx.CopyFromRows(inputRows))
		if err != nil {
			return err
		}
	}

	return err
}

func copyFromImageComponentCVEEdges(ctx context.Context, tx *postgres.Tx, _ string, objs ...*storage.ComponentCVEEdge) error {
	inputRows := [][]interface{}{}

	var err error

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

		if !storedComponentsCVEEdges.Add(obj.GetId()) {
			continue
		}

		inputRows = append(inputRows, []interface{}{
			obj.GetId(),
			obj.GetIsFixable(),
			obj.GetFixedBy(),
			obj.GetImageComponentId(),
			obj.GetImageCveId(),
			serialized,
		})

		// if we hit our batch size we need to push the data
		if len(inputRows)%batchSize == 0 || idx == len(objs)-1 {
			_, err = tx.CopyFrom(ctx, pgx.Identifier{componentCVEEdgesTable}, copyCols, pgx.CopyFromRows(inputRows))

			if err != nil {
				return err
			}

			// Clear the input rows for the next batch
			inputRows = inputRows[:0]
		}
	}
	if len(inputRows) != 0 {
		_, err = tx.CopyFrom(ctx, pgx.Identifier{componentCVEEdgesTable}, copyCols, pgx.CopyFromRows(inputRows))

		if err != nil {
			return err
		}
	}

	return err
}

func copyFromImageCVEEdges(ctx context.Context, tx *postgres.Tx, iTime *protoTypes.Timestamp, objs ...*storage.ImageCVEEdge) error {
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

	for idx, obj := range objs {
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
	return pgutils.Retry(func() error {
		return s.upsert(ctx, obj)
	})
}

// Count returns the number of objects in the store
func (s *storeImpl) Count(ctx context.Context) (int, error) {
	return pgutils.Retry2(func() (int, error) {
		row := s.db.QueryRow(ctx, countStmt)
		var count int
		if err := row.Scan(&count); err != nil {
			return 0, err
		}
		return count, nil
	})
}

// Get returns the object, if it exists from the store.
func (s *storeImpl) Get(ctx context.Context, id string) (*storage.Image, bool, error) {
	return pgutils.Retry3(func() (*storage.Image, bool, error) {
		return s.retryableGet(ctx, id)
	})
}

func (s *storeImpl) retryableGet(ctx context.Context, id string) (*storage.Image, bool, error) {
	conn, release, err := s.acquireConn(ctx)
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

func (s *storeImpl) getFullImage(ctx context.Context, tx *postgres.Tx, imageID string) (*storage.Image, bool, error) {
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

func (s *storeImpl) acquireConn(ctx context.Context) (*postgres.Conn, func(), error) {
	conn, err := s.db.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	return conn, conn.Release, nil
}

func getImageComponentEdges(ctx context.Context, tx *postgres.Tx, imageID string) (map[string]*storage.ImageComponentEdge, error) {
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
	return componentIDToEdgeMap, rows.Err()
}

func getImageCVEEdgeIDs(ctx context.Context, tx *postgres.Tx, imageID string) (set.StringSet, error) {
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
	return ids, rows.Err()
}

func getImageCVEEdges(ctx context.Context, tx *postgres.Tx, imageID string) (map[string]*storage.ImageCVEEdge, error) {
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
	return cveIDToEdgeMap, rows.Err()
}

func getImageComponents(ctx context.Context, tx *postgres.Tx, componentIDs []string) (map[string]*storage.ImageComponent, error) {
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
	return idToComponentMap, rows.Err()
}

func getComponentCVEEdges(ctx context.Context, tx *postgres.Tx, componentIDs []string) (map[string][]*storage.ComponentCVEEdge, error) {
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
	return componentIDToEdgeMap, rows.Err()
}

func getCVEs(ctx context.Context, tx *postgres.Tx, cveIDs []string) (map[string]*storage.ImageCVE, error) {
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
	return idToCVEMap, rows.Err()
}

// GetIDs returns all the IDs for the store
func (s *storeImpl) GetIDs(ctx context.Context) ([]string, error) {
	return pgutils.Retry2(func() ([]string, error) {
		return s.retryableGetIDs(ctx)
	})
}

func (s *storeImpl) retryableGetIDs(ctx context.Context) ([]string, error) {
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
	return ids, rows.Err()
}

// GetMany returns the objects specified by the IDs or the index in the missing indices slice
func (s *storeImpl) GetMany(ctx context.Context, ids []string) ([]*storage.Image, []int, error) {
	return pgutils.Retry3(func() ([]*storage.Image, []int, error) {
		return s.retryableGetMany(ctx, ids)
	})
}

func (s *storeImpl) retryableGetMany(ctx context.Context, ids []string) ([]*storage.Image, []int, error) {
	conn, release, err := s.acquireConn(ctx)
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
