package postgres

import (
	"context"
	"fmt"
	"time"

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
	imageScanV2Table = pkgSchema.ImageScanV2TableName
	componentsTable  = pkgSchema.ScanComponentsTableName
	findingsTable    = pkgSchema.ScanFindingsTableName
	batchSize        = 500
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
			       MIN(firstsystemoccurrence) as first_seen
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
			var firstSeen *time.Time
			if err := rows.Scan(&row.CVEName, &row.Severity, &row.CVSS, &row.ImageCount, &row.Fixable, &firstSeen); err != nil {
				return nil, errors.Wrap(err, "scanning row")
			}
			row.FirstSeen = firstSeen
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
