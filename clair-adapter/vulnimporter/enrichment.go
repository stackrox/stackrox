package vulnimporter

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stackrox/rox/clair-adapter/mappers"
)

// EnrichmentFetcher queries Clair's enrichment table directly
// to retrieve NVD and EPSS data for a set of CVE IDs.
type EnrichmentFetcher struct {
	pool *pgxpool.Pool
}

func NewEnrichmentFetcher(pool *pgxpool.Pool) *EnrichmentFetcher {
	return &EnrichmentFetcher{pool: pool}
}

type enrichmentRow struct {
	Tags []string
	Data json.RawMessage
}

const enrichmentQuery = `
SELECT e.tags, e.data
FROM enrichment e
JOIN uo_enrich ue ON ue.enrich = e.id
JOIN latest_update_operations l ON l.id = ue.uo
WHERE l.updater = $1
  AND l.kind = 'enrichment'
  AND e.tags && $2::text[];`

func (f *EnrichmentFetcher) fetchRows(ctx context.Context, updater string, cves []string) ([]enrichmentRow, error) {
	rows, err := f.pool.Query(ctx, enrichmentQuery, updater, cves)
	if err != nil {
		return nil, fmt.Errorf("querying enrichment for %s: %w", updater, err)
	}
	defer rows.Close()

	var result []enrichmentRow
	for rows.Next() {
		var r enrichmentRow
		if err := rows.Scan(&r.Tags, &r.Data); err != nil {
			return nil, fmt.Errorf("scanning enrichment row: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// nvdData matches the JSON stored in the enrichment table by the StackRox NVD enricher.
type nvdData struct {
	ID      string `json:"id"`
	Metrics struct {
		CvssMetricV2  []cvssMetric `json:"cvssMetricV2"`
		CvssMetricV31 []cvssMetric `json:"cvssMetricV31"`
		CvssMetricV30 []cvssMetric `json:"cvssMetricV30"`
		CvssMetricV40 []cvssMetric `json:"cvssMetricV40"`
	} `json:"metrics"`
}

type cvssMetric struct {
	CvssData struct {
		Version      string  `json:"version"`
		BaseScore    float64 `json:"baseScore"`
		VectorString string  `json:"vectorString"`
	} `json:"cvssData"`
}

// FetchNVD queries the enrichment table for NVD data matching the given CVE IDs.
func (f *EnrichmentFetcher) FetchNVD(ctx context.Context, cves []string) map[string]*mappers.NVDItem {
	if len(cves) == 0 {
		return nil
	}
	rows, err := f.fetchRows(ctx, "nvd", cves)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch NVD enrichments", "error", err)
		return nil
	}
	result := make(map[string]*mappers.NVDItem, len(rows))
	for _, r := range rows {
		var d nvdData
		if err := json.Unmarshal(r.Data, &d); err != nil {
			continue
		}
		item := &mappers.NVDItem{}
		if len(d.Metrics.CvssMetricV31) > 0 {
			m := d.Metrics.CvssMetricV31[0]
			item.CVSSv3 = &mappers.CVSSScore{BaseScore: m.CvssData.BaseScore, Vector: m.CvssData.VectorString}
		} else if len(d.Metrics.CvssMetricV30) > 0 {
			m := d.Metrics.CvssMetricV30[0]
			item.CVSSv3 = &mappers.CVSSScore{BaseScore: m.CvssData.BaseScore, Vector: m.CvssData.VectorString}
		}
		if len(d.Metrics.CvssMetricV2) > 0 {
			m := d.Metrics.CvssMetricV2[0]
			item.CVSSv2 = &mappers.CVSSScore{BaseScore: m.CvssData.BaseScore, Vector: m.CvssData.VectorString}
		}
		result[d.ID] = item
	}
	slog.DebugContext(ctx, "fetched NVD enrichments", "requested", len(cves), "found", len(result))
	return result
}

// epssData matches the JSON stored in the enrichment table by the EPSS enricher.
type epssData struct {
	CVE          string  `json:"cve"`
	ModelVersion string  `json:"modelVersion"`
	Date         string  `json:"date"`
	EPSS         float64 `json:"epss"`
	Percentile   float64 `json:"percentile"`
}

// FetchEPSS queries the enrichment table for EPSS data matching the given CVE IDs.
func (f *EnrichmentFetcher) FetchEPSS(ctx context.Context, cves []string) map[string]*mappers.EPSSItem {
	if len(cves) == 0 {
		return nil
	}
	rows, err := f.fetchRows(ctx, "clair.epss", cves)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch EPSS enrichments", "error", err)
		return nil
	}
	result := make(map[string]*mappers.EPSSItem, len(rows))
	for _, r := range rows {
		var d epssData
		if err := json.Unmarshal(r.Data, &d); err != nil {
			continue
		}
		result[d.CVE] = &mappers.EPSSItem{
			ModelVersion: d.ModelVersion,
			Date:         d.Date,
			Probability:  d.EPSS,
			Percentile:   d.Percentile,
		}
	}
	slog.DebugContext(ctx, "fetched EPSS enrichments", "requested", len(cves), "found", len(result))
	return result
}
