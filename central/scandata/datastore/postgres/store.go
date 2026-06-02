package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/scandata/datastore"
	"github.com/stackrox/rox/central/scandata/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	pkgSchema "github.com/stackrox/rox/pkg/postgres/schema"
)

const (
	imageScanV2Table      = pkgSchema.ImageScanV2TableName
	componentsTable       = pkgSchema.ScanComponentsTableName
	findingsTable         = pkgSchema.ScanFindingsTableName
	deploymentsTable      = pkgSchema.DeploymentsTableName
	deploymentsContainers = pkgSchema.DeploymentsContainersTableName
	clustersTable         = pkgSchema.ClustersTableName
	batchSize             = 500
)

var (
	log = logging.LoggerForModule()
)

type storeImpl struct {
	db postgres.DB
}

// New creates a new ScanData store
func New(db postgres.DB) datastore.DataStore {
	return &storeImpl{
		db: db,
	}
}

// UpsertScanData atomically replaces all scan data for an image
func (s *storeImpl) UpsertScanData(ctx context.Context, data *types.ScanData) error {
	if data.Scan == nil || data.Scan.GetImageId() == "" {
		return errors.New("scan data must have image_id")
	}

	imageID := data.Scan.GetImageId()

	return pgutils.Retry(ctx, func() error {
		tx, ctx, err := postgres.GetTransaction(ctx, s.db)
		if err != nil {
			return errors.Wrap(err, "getting transaction")
		}

		// Delete old data (cascades to components + findings via FK)
		if _, err := tx.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE imageid = $1", imageScanV2Table), imageID); err != nil {
			if errTx := tx.Rollback(ctx); errTx != nil {
				return errors.Wrapf(errTx, "rolling back transaction due to: %v", err)
			}
			return errors.Wrap(err, "deleting old scan data")
		}

		// Insert scan
		if err := s.insertScan(ctx, tx, data.Scan); err != nil {
			if errTx := tx.Rollback(ctx); errTx != nil {
				return errors.Wrapf(errTx, "rolling back transaction due to: %v", err)
			}
			return errors.Wrap(err, "inserting scan")
		}

		// Bulk insert components
		if len(data.Components) > 0 {
			if err := s.bulkInsertComponents(ctx, tx, data.Components); err != nil {
				if errTx := tx.Rollback(ctx); errTx != nil {
					return errors.Wrapf(errTx, "rolling back transaction due to: %v", err)
				}
				return errors.Wrap(err, "inserting components")
			}
		}

		// Bulk insert findings
		if len(data.Findings) > 0 {
			if err := s.bulkInsertFindings(ctx, tx, data.Findings); err != nil {
				if errTx := tx.Rollback(ctx); errTx != nil {
					return errors.Wrapf(errTx, "rolling back transaction due to: %v", err)
				}
				return errors.Wrap(err, "inserting findings")
			}
		}

		return tx.Commit(ctx)
	})
}

func (s *storeImpl) insertScan(ctx context.Context, tx *postgres.Tx, scan *storage.ImageScanV2) error {
	serialized, err := scan.MarshalVT()
	if err != nil {
		return errors.Wrap(err, "marshaling scan")
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (id, imageid, scantime, scannerversion, bundleversion, serialized)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, imageScanV2Table)

	_, err = tx.Exec(ctx, query,
		scan.GetId(),
		scan.GetImageId(),
		scan.GetScanTime().AsTime(),
		scan.GetScannerVersion(),
		scan.GetBundleVersion(),
		serialized,
	)
	return err
}

func (s *storeImpl) bulkInsertComponents(ctx context.Context, tx *postgres.Tx, components []*storage.ScanComponent) error {
	copyCols := []string{
		"id",
		"scanid",
		"imageid",
		"name",
		"version",
		"source",
		"location",
		"layerindex",
		"layertype",
		"fixedby",
		"operatingsystem",
		"arch",
		"module",
		"sourcepackagename",
		"sourcepackageversion",
		"cpe",
		"kind",
		"repositoryhint",
		"serialized",
	}

	// Process in batches
	for i := 0; i < len(components); i += batchSize {
		end := min(i+batchSize, len(components))
		batch := components[i:end]

		inputRows := make([][]any, 0, len(batch))
		for _, comp := range batch {
			serialized, err := comp.MarshalVT()
			if err != nil {
				return errors.Wrap(err, "marshaling component")
			}

			var layerIndex any
			if comp.GetHasLayerIndex() != nil {
				layerIndex = comp.GetLayerIndex()
			}

			inputRows = append(inputRows, []any{
				comp.GetId(),
				comp.GetScanId(),
				comp.GetImageId(),
				comp.GetName(),
				comp.GetVersion(),
				comp.GetSource(),
				comp.GetLocation(),
				layerIndex,
				comp.GetLayerType(),
				comp.GetFixedBy(),
				comp.GetOperatingSystem(),
				comp.GetArch(),
				comp.GetModule(),
				comp.GetSourcePackageName(),
				comp.GetSourcePackageVersion(),
				comp.GetCpe(),
				comp.GetKind(),
				comp.GetRepositoryHint(),
				serialized,
			})
		}

		if _, err := tx.CopyFrom(ctx, pgx.Identifier{componentsTable}, copyCols, pgx.CopyFromRows(inputRows)); err != nil {
			return errors.Wrap(err, "copying components")
		}
	}

	return nil
}

func (s *storeImpl) bulkInsertFindings(ctx context.Context, tx *postgres.Tx, findings []*storage.ScanFinding) error {
	copyCols := []string{
		"id",
		"advisoryid",
		"cvename",
		"componentid",
		"scanid",
		"imageid",
		"severity",
		"cvss",
		"cvssversion",
		"nvdcvss",
		"nvdcvssversion",
		"epssprobability",
		"epsspercentile",
		"isfixable",
		"fixedby",
		"fixeddate",
		"description",
		"publisheddate",
		"datasource",
		"sourcename",
		"state",
		"firstimageoccurrence",
		"firstsystemoccurrence",
		"serialized",
	}

	// Process in batches
	for i := 0; i < len(findings); i += batchSize {
		end := min(i+batchSize, len(findings))
		batch := findings[i:end]

		inputRows := make([][]any, 0, len(batch))
		for _, finding := range batch {
			serialized, err := finding.MarshalVT()
			if err != nil {
				return errors.Wrap(err, "marshaling finding")
			}

			var fixedDate, publishedDate, firstImageOccurrence, firstSystemOccurrence any
			if finding.GetFixedDate() != nil {
				fixedDate = finding.GetFixedDate().AsTime()
			}
			if finding.GetPublishedDate() != nil {
				publishedDate = finding.GetPublishedDate().AsTime()
			}
			if finding.GetFirstImageOccurrence() != nil {
				firstImageOccurrence = finding.GetFirstImageOccurrence().AsTime()
			}
			if finding.GetFirstSystemOccurrence() != nil {
				firstSystemOccurrence = finding.GetFirstSystemOccurrence().AsTime()
			}

			inputRows = append(inputRows, []any{
				finding.GetId(),
				finding.GetAdvisoryId(),
				finding.GetCveName(),
				finding.GetComponentId(),
				finding.GetScanId(),
				finding.GetImageId(),
				finding.GetSeverity(),
				finding.GetCvss(),
				finding.GetCvssVersion(),
				finding.GetNvdCvss(),
				finding.GetNvdCvssVersion(),
				finding.GetEpssProbability(),
				finding.GetEpssPercentile(),
				finding.GetIsFixable(),
				finding.GetFixedBy(),
				fixedDate,
				finding.GetDescription(),
				publishedDate,
				finding.GetDataSource(),
				finding.GetSourceName(),
				finding.GetState(),
				firstImageOccurrence,
				firstSystemOccurrence,
				serialized,
			})
		}

		if _, err := tx.CopyFrom(ctx, pgx.Identifier{findingsTable}, copyCols, pgx.CopyFromRows(inputRows)); err != nil {
			return errors.Wrap(err, "copying findings")
		}
	}

	return nil
}

// GetScanDataByImageID returns complete scan data for an image
func (s *storeImpl) GetScanDataByImageID(ctx context.Context, imageID string) (*types.ScanData, error) {
	return pgutils.Retry2(ctx, func() (*types.ScanData, error) {
		// Get scan
		var scanSerialized []byte
		query := fmt.Sprintf("SELECT serialized FROM %s WHERE imageid = $1", imageScanV2Table)
		err := s.db.QueryRow(ctx, query, imageID).Scan(&scanSerialized)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, nil
			}
			return nil, errors.Wrap(err, "fetching scan")
		}

		scan := new(storage.ImageScanV2)
		if err := scan.UnmarshalVT(scanSerialized); err != nil {
			return nil, errors.Wrap(err, "unmarshaling scan")
		}

		// Get components
		query = fmt.Sprintf("SELECT serialized FROM %s WHERE imageid = $1", componentsTable)
		rows, err := s.db.Query(ctx, query, imageID)
		if err != nil {
			return nil, errors.Wrap(err, "fetching components")
		}
		components, err := pgutils.ScanRows[storage.ScanComponent, *storage.ScanComponent](rows)
		if err != nil {
			return nil, errors.Wrap(err, "scanning components")
		}

		// Get findings
		query = fmt.Sprintf("SELECT serialized FROM %s WHERE imageid = $1", findingsTable)
		rows, err = s.db.Query(ctx, query, imageID)
		if err != nil {
			return nil, errors.Wrap(err, "fetching findings")
		}
		findings, err := pgutils.ScanRows[storage.ScanFinding, *storage.ScanFinding](rows)
		if err != nil {
			return nil, errors.Wrap(err, "scanning findings")
		}

		return &types.ScanData{
			Scan:       scan,
			Components: components,
			Findings:   findings,
		}, nil
	})
}

// DeleteByImageID removes all scan data for an image
func (s *storeImpl) DeleteByImageID(ctx context.Context, imageID string) error {
	return pgutils.Retry(ctx, func() error {
		query := fmt.Sprintf("DELETE FROM %s WHERE imageid = $1", imageScanV2Table)
		_, err := s.db.Exec(ctx, query, imageID)
		return err
	})
}

// ListCVEs returns the CVE list page data with GROUP BY aggregation
func (s *storeImpl) ListCVEs(ctx context.Context, limit, offset int) ([]*types.CVEListRow, int, error) {
	type result struct {
		rows  []*types.CVEListRow
		total int
	}

	res, err := pgutils.Retry2(ctx, func() (*result, error) {
		// Get total count
		countQuery := fmt.Sprintf(`
			SELECT COUNT(DISTINCT cvename)
			FROM %s
			WHERE state = 0
		`, findingsTable)
		var total int
		if err := s.db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
			return nil, errors.Wrap(err, "counting CVEs")
		}

		// Get paginated results
		query := fmt.Sprintf(`
			SELECT cvename,
			       MAX(severity)::int as severity,
			       MAX(cvss) as cvss,
			       COUNT(DISTINCT imageid) as image_count,
			       BOOL_OR(isfixable) as fixable,
			       MIN(firstsystemoccurrence) as first_seen,
			       MIN(publisheddate) as published_date,
			       MAX(epssprobability) as epss_probability
			FROM %s
			WHERE state = 0
			GROUP BY cvename
			ORDER BY MAX(severity) DESC, MAX(cvss) DESC
			LIMIT $1 OFFSET $2
		`, findingsTable)

		rows, err := s.db.Query(ctx, query, limit, offset)
		if err != nil {
			return nil, errors.Wrap(err, "querying CVEs")
		}
		defer rows.Close()

		var results []*types.CVEListRow
		for rows.Next() {
			var row types.CVEListRow
			if err := rows.Scan(&row.CVEName, &row.Severity, &row.CVSS, &row.ImageCount, &row.Fixable, &row.FirstSeen, &row.PublishedDate, &row.EPSSProbability); err != nil {
				return nil, errors.Wrap(err, "scanning row")
			}
			results = append(results, &row)
		}

		if err := rows.Err(); err != nil {
			return nil, errors.Wrap(err, "iterating rows")
		}

		return &result{rows: results, total: total}, nil
	})

	if err != nil {
		return nil, 0, err
	}

	return res.rows, res.total, nil
}

// GetFindingsByCVE returns all findings for a specific CVE name
func (s *storeImpl) GetFindingsByCVE(ctx context.Context, cveName string) ([]*storage.ScanFinding, error) {
	return pgutils.Retry2(ctx, func() ([]*storage.ScanFinding, error) {
		query := fmt.Sprintf("SELECT serialized FROM %s WHERE cvename = $1", findingsTable)
		rows, err := s.db.Query(ctx, query, cveName)
		if err != nil {
			return nil, errors.Wrap(err, "querying findings")
		}
		return pgutils.ScanRows[storage.ScanFinding, *storage.ScanFinding](rows)
	})
}

// GetFindingsWithComponentsByCVE returns findings joined with their parent component's metadata.
func (s *storeImpl) GetFindingsWithComponentsByCVE(ctx context.Context, cveName string) ([]*types.FindingWithComponent, error) {
	return pgutils.Retry2(ctx, func() ([]*types.FindingWithComponent, error) {
		query := fmt.Sprintf(`
			SELECT f.serialized, c.name, c.version, c.source
			FROM %s f
			JOIN %s c ON f.componentid = c.id
			WHERE f.cvename = $1
		`, findingsTable, componentsTable)

		rows, err := s.db.Query(ctx, query, cveName)
		if err != nil {
			return nil, errors.Wrap(err, "querying findings with components")
		}
		defer rows.Close()

		var results []*types.FindingWithComponent
		for rows.Next() {
			var serialized []byte
			var compName, compVersion string
			var compSource int32
			if err := rows.Scan(&serialized, &compName, &compVersion, &compSource); err != nil {
				return nil, errors.Wrap(err, "scanning finding row")
			}

			finding := new(storage.ScanFinding)
			if err := finding.UnmarshalVT(serialized); err != nil {
				return nil, errors.Wrap(err, "unmarshaling finding")
			}

			results = append(results, &types.FindingWithComponent{
				Finding:          finding,
				ComponentName:    compName,
				ComponentVersion: compVersion,
				ComponentSource:  compSource,
			})
		}

		if err := rows.Err(); err != nil {
			return nil, errors.Wrap(err, "iterating finding rows")
		}

		return results, nil
	})
}

// GetFindingsWithComponentsByImageID returns findings joined with their parent component's metadata for an image.
func (s *storeImpl) GetFindingsWithComponentsByImageID(ctx context.Context, imageID string) ([]*types.FindingWithComponent, error) {
	return pgutils.Retry2(ctx, func() ([]*types.FindingWithComponent, error) {
		query := fmt.Sprintf(`
			SELECT f.serialized, c.name, c.version, c.source, c.location, c.arch
			FROM %s f
			JOIN %s c ON f.componentid = c.id
			WHERE f.imageid = $1
		`, findingsTable, componentsTable)

		rows, err := s.db.Query(ctx, query, imageID)
		if err != nil {
			return nil, errors.Wrap(err, "querying findings with components by image")
		}
		defer rows.Close()

		var results []*types.FindingWithComponent
		for rows.Next() {
			var serialized []byte
			var compName, compVersion, compLocation, compArch string
			var compSource int32
			if err := rows.Scan(&serialized, &compName, &compVersion, &compSource, &compLocation, &compArch); err != nil {
				return nil, errors.Wrap(err, "scanning finding row")
			}

			finding := new(storage.ScanFinding)
			if err := finding.UnmarshalVT(serialized); err != nil {
				return nil, errors.Wrap(err, "unmarshaling finding")
			}

			results = append(results, &types.FindingWithComponent{
				Finding:           finding,
				ComponentName:     compName,
				ComponentVersion:  compVersion,
				ComponentSource:   compSource,
				ComponentLocation: compLocation,
				ComponentArch:     compArch,
			})
		}

		if err := rows.Err(); err != nil {
			return nil, errors.Wrap(err, "iterating finding rows")
		}

		return results, nil
	})
}

// GetFindingsByImageID returns all findings for an image
func (s *storeImpl) GetFindingsByImageID(ctx context.Context, imageID string) ([]*storage.ScanFinding, error) {
	return pgutils.Retry2(ctx, func() ([]*storage.ScanFinding, error) {
		query := fmt.Sprintf("SELECT serialized FROM %s WHERE imageid = $1", findingsTable)
		rows, err := s.db.Query(ctx, query, imageID)
		if err != nil {
			return nil, errors.Wrap(err, "querying findings")
		}
		return pgutils.ScanRows[storage.ScanFinding, *storage.ScanFinding](rows)
	})
}

// GetImageInfoByDigests looks up image UUID and full name from images_v2 by SHA digest.
func (s *storeImpl) GetImageInfoByDigests(ctx context.Context, digests []string) (map[string]types.ImageBasicInfo, error) {
	if len(digests) == 0 {
		return nil, nil
	}
	return pgutils.Retry2(ctx, func() (map[string]types.ImageBasicInfo, error) {
		query := `SELECT id, digest, name_fullname FROM images_v2 WHERE digest = ANY($1)`
		rows, err := s.db.Query(ctx, query, digests)
		if err != nil {
			return nil, errors.Wrap(err, "querying image info")
		}
		defer rows.Close()
		result := make(map[string]types.ImageBasicInfo, len(digests))
		for rows.Next() {
			var uuid, digest, fullName string
			if err := rows.Scan(&uuid, &digest, &fullName); err != nil {
				return nil, errors.Wrap(err, "scanning image row")
			}
			result[digest] = types.ImageBasicInfo{UUID: uuid, FullName: fullName}
		}
		return result, rows.Err()
	})
}

// ListDeployments returns deployments with their CVE counts and severity.
func (s *storeImpl) ListDeployments(ctx context.Context, limit, offset int) ([]*types.DeploymentListRow, int, error) {
	type result struct {
		rows  []*types.DeploymentListRow
		total int
	}

	res, err := pgutils.Retry2(ctx, func() (*result, error) {
		// Get total count of deployments with at least one CVE
		countQuery := fmt.Sprintf(`
			SELECT COUNT(DISTINCT d.id)
			FROM %s d
			JOIN %s dc ON d.id = dc.deployments_id
			JOIN %s f ON dc.image_id = f.imageid
			WHERE f.state = 0
		`, deploymentsTable, deploymentsContainers, findingsTable)
		var total int
		if err := s.db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
			return nil, errors.Wrap(err, "counting deployments")
		}

		// Get paginated results with cluster name
		query := fmt.Sprintf(`
			SELECT
				d.id,
				d.name,
				d.clusterid,
				COALESCE(c.name, d.clusterid::text) as cluster_name,
				d.namespace,
				COUNT(DISTINCT dc.image_id) as image_count,
				COUNT(DISTINCT f.cvename) as cve_count,
				MAX(f.severity)::int as top_severity,
				BOOL_OR(f.isfixable) as fixable
			FROM %s d
			JOIN %s dc ON d.id = dc.deployments_id
			JOIN %s f ON dc.image_id = f.imageid
			LEFT JOIN %s c ON d.clusterid = c.id
			WHERE f.state = 0
			GROUP BY d.id, d.name, d.clusterid, c.name, d.namespace
			ORDER BY MAX(f.severity) DESC, COUNT(DISTINCT f.cvename) DESC
			LIMIT $1 OFFSET $2
		`, deploymentsTable, deploymentsContainers, findingsTable, clustersTable)

		rows, err := s.db.Query(ctx, query, limit, offset)
		if err != nil {
			return nil, errors.Wrap(err, "querying deployments")
		}
		defer rows.Close()

		var results []*types.DeploymentListRow
		for rows.Next() {
			var row types.DeploymentListRow
			if err := rows.Scan(
				&row.ID,
				&row.Name,
				&row.ClusterID,
				&row.ClusterName,
				&row.Namespace,
				&row.ImageCount,
				&row.CVECount,
				&row.TopSeverity,
				&row.Fixable,
			); err != nil {
				return nil, errors.Wrap(err, "scanning deployment row")
			}
			results = append(results, &row)
		}

		if err := rows.Err(); err != nil {
			return nil, errors.Wrap(err, "iterating deployment rows")
		}

		return &result{rows: results, total: total}, nil
	})

	if err != nil {
		return nil, 0, err
	}

	return res.rows, res.total, nil
}

// GetDeploymentImages returns images for a deployment with CVE summary.
func (s *storeImpl) GetDeploymentImages(ctx context.Context, deploymentID string) ([]*types.DeploymentImageRow, error) {
	return pgutils.Retry2(ctx, func() ([]*types.DeploymentImageRow, error) {
		query := fmt.Sprintf(`
			SELECT
				dc.image_id,
				COALESCE(i.id, '') as image_uuid,
				COALESCE(i.name_fullname, '') as image_name,
				COUNT(DISTINCT f.cvename) as cve_count,
				MAX(f.severity)::int as top_severity,
				BOOL_OR(f.isfixable) as fixable
			FROM %s dc
			JOIN %s f ON dc.image_id = f.imageid
			LEFT JOIN images_v2 i ON dc.image_id = i.digest
			WHERE dc.deployments_id = $1 AND f.state = 0
			GROUP BY dc.image_id, i.id, i.name_fullname
			ORDER BY MAX(f.severity) DESC, COUNT(DISTINCT f.cvename) DESC
		`, deploymentsContainers, findingsTable)

		rows, err := s.db.Query(ctx, query, deploymentID)
		if err != nil {
			return nil, errors.Wrap(err, "querying deployment images")
		}
		defer rows.Close()

		var results []*types.DeploymentImageRow
		for rows.Next() {
			var row types.DeploymentImageRow
			if err := rows.Scan(
				&row.ImageID,
				&row.ImageUUID,
				&row.ImageName,
				&row.CVECount,
				&row.TopSeverity,
				&row.Fixable,
			); err != nil {
				return nil, errors.Wrap(err, "scanning image row")
			}
			results = append(results, &row)
		}

		if err := rows.Err(); err != nil {
			return nil, errors.Wrap(err, "iterating image rows")
		}

		return results, nil
	})
}

// ListAdvisories returns distinct advisories with image counts.
func (s *storeImpl) ListAdvisories(ctx context.Context, limit, offset int) ([]*types.AdvisoryListRow, int, error) {
	type result struct {
		rows  []*types.AdvisoryListRow
		total int
	}

	res, err := pgutils.Retry2(ctx, func() (*result, error) {
		// Get total count of distinct advisories
		countQuery := fmt.Sprintf(`
			SELECT COUNT(DISTINCT advisoryid)
			FROM %s
			WHERE state = 0
		`, findingsTable)
		var total int
		if err := s.db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
			return nil, errors.Wrap(err, "counting advisories")
		}

		// Get paginated results
		query := fmt.Sprintf(`
			SELECT
				f.advisoryid,
				MAX(f.cvename) as cvename,
				MAX(f.severity)::int as severity,
				MAX(f.cvss) as cvss,
				MAX(f.sourcename) as sourcename,
				MAX(f.description) as description,
				MAX(f.fixedby) as fixedby,
				COUNT(DISTINCT f.imageid) as image_count,
				COUNT(DISTINCT c.name || '##' || c.version) as component_count
			FROM %s f
			JOIN %s c ON f.componentid = c.id
			WHERE f.state = 0
			GROUP BY f.advisoryid
			ORDER BY MAX(f.severity) DESC, MAX(f.cvss) DESC
			LIMIT $1 OFFSET $2
		`, findingsTable, componentsTable)

		rows, err := s.db.Query(ctx, query, limit, offset)
		if err != nil {
			return nil, errors.Wrap(err, "querying advisories")
		}
		defer rows.Close()

		var results []*types.AdvisoryListRow
		for rows.Next() {
			var row types.AdvisoryListRow
			if err := rows.Scan(
				&row.AdvisoryID,
				&row.CVEName,
				&row.Severity,
				&row.CVSS,
				&row.SourceName,
				&row.Description,
				&row.FixedBy,
				&row.ImageCount,
				&row.ComponentCount,
			); err != nil {
				return nil, errors.Wrap(err, "scanning advisory row")
			}
			results = append(results, &row)
		}

		if err := rows.Err(); err != nil {
			return nil, errors.Wrap(err, "iterating advisory rows")
		}

		return &result{rows: results, total: total}, nil
	})

	if err != nil {
		return nil, 0, err
	}

	return res.rows, res.total, nil
}

// GetDeploymentByID returns basic deployment info.
func (s *storeImpl) GetDeploymentByID(ctx context.Context, deploymentID string) (*types.DeploymentListRow, error) {
	return pgutils.Retry2(ctx, func() (*types.DeploymentListRow, error) {
		query := fmt.Sprintf(`
			SELECT
				d.id,
				d.name,
				d.clusterid,
				COALESCE(c.name, d.clusterid::text) as cluster_name,
				d.namespace
			FROM %s d
			LEFT JOIN %s c ON d.clusterid = c.id
			WHERE d.id = $1
		`, deploymentsTable, clustersTable)

		var row types.DeploymentListRow
		err := s.db.QueryRow(ctx, query, deploymentID).Scan(
			&row.ID,
			&row.Name,
			&row.ClusterID,
			&row.ClusterName,
			&row.Namespace,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, nil
			}
			return nil, errors.Wrap(err, "querying deployment")
		}

		return &row, nil
	})
}

// ListComponents returns distinct components with CVE severity breakdown.
func (s *storeImpl) ListComponents(ctx context.Context, limit, offset int) ([]*types.ComponentListRow, int, error) {
	type result struct {
		rows  []*types.ComponentListRow
		total int
	}

	res, err := pgutils.Retry2(ctx, func() (*result, error) {
		// Get total count of distinct components
		countQuery := fmt.Sprintf(`
			SELECT COUNT(DISTINCT c.name)
			FROM %s c
			JOIN %s f ON c.id = f.componentid
			WHERE f.state = 0
		`, componentsTable, findingsTable)
		var total int
		if err := s.db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
			return nil, errors.Wrap(err, "counting components")
		}

		// Get paginated results
		query := fmt.Sprintf(`
			SELECT
				c.name,
				COUNT(DISTINCT c.version) as version_count,
				COUNT(DISTINCT f.cvename) as cve_count,
				COUNT(DISTINCT f.imageid) as image_count,
				MAX(f.severity)::int as top_severity,
				MAX(f.cvss) as top_cvss,
				COUNT(DISTINCT CASE WHEN f.severity = 4 THEN f.cvename END) as critical_count,
				COUNT(DISTINCT CASE WHEN f.severity = 3 THEN f.cvename END) as important_count,
				COUNT(DISTINCT CASE WHEN f.severity = 2 THEN f.cvename END) as moderate_count,
				COUNT(DISTINCT CASE WHEN f.severity = 1 THEN f.cvename END) as low_count
			FROM %s c
			JOIN %s f ON c.id = f.componentid
			WHERE f.state = 0
			GROUP BY c.name
			ORDER BY MAX(f.severity) DESC, COUNT(DISTINCT f.cvename) DESC
			LIMIT $1 OFFSET $2
		`, componentsTable, findingsTable)

		rows, err := s.db.Query(ctx, query, limit, offset)
		if err != nil {
			return nil, errors.Wrap(err, "querying components")
		}
		defer rows.Close()

		var results []*types.ComponentListRow
		for rows.Next() {
			var row types.ComponentListRow
			if err := rows.Scan(
				&row.Name,
				&row.VersionCount,
				&row.CVECount,
				&row.ImageCount,
				&row.TopSeverity,
				&row.TopCVSS,
				&row.CriticalCount,
				&row.ImportantCount,
				&row.ModerateCount,
				&row.LowCount,
			); err != nil {
				return nil, errors.Wrap(err, "scanning component row")
			}
			results = append(results, &row)
		}

		if err := rows.Err(); err != nil {
			return nil, errors.Wrap(err, "iterating component rows")
		}

		return &result{rows: results, total: total}, nil
	})

	if err != nil {
		return nil, 0, err
	}

	return res.rows, res.total, nil
}

// GetComponentVersions returns all versions of a component with CVE data.
func (s *storeImpl) GetComponentVersions(ctx context.Context, componentName string) ([]*types.ComponentVersionInfo, error) {
	return pgutils.Retry2(ctx, func() ([]*types.ComponentVersionInfo, error) {
		query := fmt.Sprintf(`
			SELECT
				c.version,
				c.source,
				c.arch,
				c.module,
				COUNT(DISTINCT f.cvename) as cve_count,
				COUNT(DISTINCT f.imageid) as image_count,
				MAX(f.severity)::int as top_severity,
				MAX(f.cvss) as top_cvss,
				BOOL_OR(f.isfixable) as fixable,
				MAX(f.fixedby) as fixed_by
			FROM %s c
			JOIN %s f ON c.id = f.componentid
			WHERE f.state = 0 AND c.name = $1
			GROUP BY c.version, c.source, c.arch, c.module
			ORDER BY c.version
		`, componentsTable, findingsTable)

		rows, err := s.db.Query(ctx, query, componentName)
		if err != nil {
			return nil, errors.Wrap(err, "querying component versions")
		}
		defer rows.Close()

		var results []*types.ComponentVersionInfo
		for rows.Next() {
			var row types.ComponentVersionInfo
			var source int32
			if err := rows.Scan(
				&row.Version,
				&source,
				&row.Arch,
				&row.Module,
				&row.CVECount,
				&row.ImageCount,
				&row.TopSeverity,
				&row.TopCVSS,
				&row.Fixable,
				&row.FixedBy,
			); err != nil {
				return nil, errors.Wrap(err, "scanning version row")
			}
			row.Source = storage.SourceType(source).String()
			results = append(results, &row)
		}

		if err := rows.Err(); err != nil {
			return nil, errors.Wrap(err, "iterating version rows")
		}

		return results, nil
	})
}

// GetComponentImages returns images containing the named component with CVE summary.
func (s *storeImpl) GetComponentImages(ctx context.Context, componentName string) ([]*types.ComponentImageRow, error) {
	return pgutils.Retry2(ctx, func() ([]*types.ComponentImageRow, error) {
		query := fmt.Sprintf(`
			SELECT
				c.imageid,
				c.version,
				c.arch,
				COUNT(DISTINCT f.cvename) as cve_count,
				MAX(f.severity)::int as top_severity,
				BOOL_OR(f.isfixable) as fixable
			FROM %s c
			JOIN %s f ON c.id = f.componentid
			WHERE c.name = $1 AND f.state = 0
			GROUP BY c.imageid, c.version, c.arch
			ORDER BY MAX(f.severity) DESC, COUNT(DISTINCT f.cvename) DESC
		`, componentsTable, findingsTable)

		rows, err := s.db.Query(ctx, query, componentName)
		if err != nil {
			return nil, errors.Wrap(err, "querying component images")
		}
		defer rows.Close()

		var results []*types.ComponentImageRow
		for rows.Next() {
			var row types.ComponentImageRow
			if err := rows.Scan(
				&row.ImageID,
				&row.Version,
				&row.Arch,
				&row.CVECount,
				&row.TopSeverity,
				&row.Fixable,
			); err != nil {
				return nil, errors.Wrap(err, "scanning component image row")
			}
			results = append(results, &row)
		}

		if err := rows.Err(); err != nil {
			return nil, errors.Wrap(err, "iterating component image rows")
		}

		return results, nil
	})
}

// ListImages returns images with CVE summary data.
func (s *storeImpl) ListImages(ctx context.Context, limit, offset int) ([]*types.ImageListRow, int, error) {
	type result struct {
		rows  []*types.ImageListRow
		total int
	}

	res, err := pgutils.Retry2(ctx, func() (*result, error) {
		// Get total count of distinct images with findings
		countQuery := fmt.Sprintf(`
			SELECT COUNT(DISTINCT s.imageid)
			FROM %s s
			JOIN %s f ON s.imageid = f.imageid
			WHERE f.state = 0
		`, imageScanV2Table, findingsTable)
		var total int
		if err := s.db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
			return nil, errors.Wrap(err, "counting images")
		}

		// Get paginated results
		query := fmt.Sprintf(`
			SELECT
				s.imageid,
				COUNT(DISTINCT f.cvename) as cve_count,
				COUNT(DISTINCT c.id) as component_count,
				MAX(f.severity)::int as top_severity,
				MAX(f.cvss) as top_cvss,
				BOOL_OR(f.isfixable) as fixable,
				s.scantime,
				COUNT(DISTINCT CASE WHEN f.severity = 4 THEN f.cvename END) as critical_count,
				COUNT(DISTINCT CASE WHEN f.severity = 3 THEN f.cvename END) as important_count,
				COUNT(DISTINCT CASE WHEN f.severity = 2 THEN f.cvename END) as moderate_count,
				COUNT(DISTINCT CASE WHEN f.severity = 1 THEN f.cvename END) as low_count
			FROM %s s
			JOIN %s c ON s.id = c.scanid
			JOIN %s f ON c.id = f.componentid
			WHERE f.state = 0
			GROUP BY s.imageid, s.scantime
			ORDER BY MAX(f.severity) DESC, COUNT(DISTINCT f.cvename) DESC
			LIMIT $1 OFFSET $2
		`, imageScanV2Table, componentsTable, findingsTable)

		rows, err := s.db.Query(ctx, query, limit, offset)
		if err != nil {
			return nil, errors.Wrap(err, "querying images")
		}
		defer rows.Close()

		var results []*types.ImageListRow
		for rows.Next() {
			var row types.ImageListRow
			if err := rows.Scan(
				&row.ImageID,
				&row.CVECount,
				&row.ComponentCount,
				&row.TopSeverity,
				&row.TopCVSS,
				&row.Fixable,
				&row.ScanTime,
				&row.CriticalCount,
				&row.ImportantCount,
				&row.ModerateCount,
				&row.LowCount,
			); err != nil {
				return nil, errors.Wrap(err, "scanning image row")
			}
			results = append(results, &row)
		}

		if err := rows.Err(); err != nil {
			return nil, errors.Wrap(err, "iterating image rows")
		}

		return &result{rows: results, total: total}, nil
	})

	if err != nil {
		return nil, 0, err
	}

	return res.rows, res.total, nil
}

// GetComponentCVEs returns CVEs affecting a specific component name+version.
func (s *storeImpl) GetComponentCVEs(ctx context.Context, componentName, componentVersion string) ([]*types.ComponentCVERow, error) {
	return pgutils.Retry2(ctx, func() ([]*types.ComponentCVERow, error) {
		query := fmt.Sprintf(`
			SELECT DISTINCT
				f.cvename,
				MAX(f.severity)::int as severity,
				MAX(f.cvss) as cvss,
				BOOL_OR(f.isfixable) as fixable,
				MAX(f.fixedby) as fixed_by,
				MAX(f.description) as description,
				COUNT(DISTINCT f.imageid) as image_count
			FROM %s f
			JOIN %s c ON f.componentid = c.id
			WHERE c.name = $1 AND c.version = $2 AND f.state = 0
			GROUP BY f.cvename
			ORDER BY MAX(f.severity) DESC, MAX(f.cvss) DESC
		`, findingsTable, componentsTable)

		rows, err := s.db.Query(ctx, query, componentName, componentVersion)
		if err != nil {
			return nil, errors.Wrap(err, "querying component CVEs")
		}
		defer rows.Close()

		var results []*types.ComponentCVERow
		for rows.Next() {
			var row types.ComponentCVERow
			if err := rows.Scan(
				&row.CVEName,
				&row.Severity,
				&row.CVSS,
				&row.Fixable,
				&row.FixedBy,
				&row.Description,
				&row.ImageCount,
			); err != nil {
				return nil, errors.Wrap(err, "scanning component CVE row")
			}
			results = append(results, &row)
		}

		if err := rows.Err(); err != nil {
			return nil, errors.Wrap(err, "iterating component CVE rows")
		}

		return results, nil
	})
}

// GetImageOS returns the operating system for an image.
func (s *storeImpl) GetImageOS(ctx context.Context, imageID string) (string, error) {
	return pgutils.Retry2(ctx, func() (string, error) {
		query := fmt.Sprintf(`
			SELECT DISTINCT operatingsystem
			FROM %s
			WHERE imageid = $1 AND operatingsystem != ''
			LIMIT 1
		`, componentsTable)

		var os string
		err := s.db.QueryRow(ctx, query, imageID).Scan(&os)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return "", nil
			}
			return "", errors.Wrap(err, "querying image OS")
		}

		return os, nil
	})
}
