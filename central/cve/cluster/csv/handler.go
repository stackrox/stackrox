package csv

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	csvCommon "github.com/stackrox/rox/central/cve/common/csv"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/csv"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/parser"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	once       sync.Once
	csvHandler *csvCommon.HandlerImpl

	csvHeader = []string{
		"Cluster CVE",
		"CVE Type(s)",
		"Fixable",
		"CVSS Score",
		"Env Impact (%s)",
		"Impact Score",
		"Clusters",
		"Last Scanned",
		"Published",
		"Summary",
	}
)

func initialize() {
	csvHandler = newHandler(resolvers.New())
}

func newHandler(resolver *resolvers.Resolver) *csvCommon.HandlerImpl {
	return csvCommon.NewCSVHandler(
		resolver,
		// CVEs must be scoped from lowest entities to highest entities. DO NOT CHANGE THE ORDER.
		[]*csvCommon.SearchWrapper{
			csvCommon.NewSearchWrapper(v1.SearchCategory_CLUSTERS, schema.ClustersSchema.OptionsMap, resolver.ClusterDataStore),
		},
	)
}

type ClusterCVERow struct {
	cve           string
	cveTypes      string
	fixable       string
	cvssScore     string
	envImpact     string
	impactScore   string
	clusterCount  string
	scannedTime   string
	publishedTime string
	summary       string
}

type csvResults struct {
	*csv.GenericWriter
}

func newCSVResults(header []string, sort bool) csvResults {
	return csvResults{
		GenericWriter: csv.NewGenericWriter(header, sort),
	}
}

func (c *csvResults) addRow(row *ClusterCVERow) {
	// platform cve, cveTypes, fixable, cvss score, env impact, impact score, clusters, scanned time, published time, summary
	value := []string{
		row.cve,
		row.cveTypes,
		row.fixable,
		row.cvssScore,
		row.envImpact,
		row.impactScore,
		row.clusterCount,
		row.scannedTime,
		row.publishedTime,
		row.summary,
	}

	c.AddValue(value)
}

// ClusterCVECSVHandler returns a handler func to serve csv export requests of Cluster CVE data for Vuln Mgmt
func ClusterCVECSVHandler() http.HandlerFunc {
	once.Do(initialize)

	return func(w http.ResponseWriter, r *http.Request) {
		query, rQuery, err := parser.ParseURLQuery(r.URL.Query())
		if err != nil {
			csv.WriteError(w, http.StatusBadRequest, err)
			return
		}
		rawQuery, paginatedQuery := resolvers.V1RawQueryAsResolverQuery(rQuery)

		cveRows, err := ClusterCVECSVRows(loaders.WithLoaderContext(r.Context()), query, rawQuery, paginatedQuery)
		if err != nil {
			csv.WriteError(w, http.StatusInternalServerError, err)
			return
		}

		postSortRequired := paginatedQuery.Pagination == nil ||
			paginatedQuery.Pagination.SortOption == nil ||
			paginatedQuery.Pagination.SortOption.Field == nil

		output := newCSVResults(csvHeader, postSortRequired)
		for _, row := range cveRows {
			output.addRow(row)
		}
		output.Write(w, "cluster_cve_export")
	}
}

func ClusterCVECSVRows(c context.Context, query *v1.Query, rawQuery resolvers.RawQuery, paginatedQuery resolvers.PaginatedQuery) ([]*ClusterCVERow, error) {
	if csvHandler == nil {
		return nil, errors.New("Handler not initialized")
	}

	ctx, err := csvHandler.GetScopeContext(c, query)
	if err != nil {
		log.Errorf("unable to determine resource scope for query %q: %v", query.String(), err)
		return nil, err
	}

	res := csvHandler.GetResolver()
	if res == nil {
		log.Errorf("Unexpected value (nil) for resolver in Handler")
		return nil, errors.New("Resolver not initialized in handler")
	}
	vulnResolvers, err := res.ClusterVulnerabilities(ctx, paginatedQuery)
	if err != nil {
		log.Errorf("unable to get cluster vulnerabilities for csv export: %v", err)
		return nil, err
	}

	cveRows := make([]*ClusterCVERow, 0, len(vulnResolvers))
	for _, d := range vulnResolvers {
		var errorList errorhelpers.ErrorList
		dataRow := &ClusterCVERow{}
		dataRow.cve = d.CVE(ctx)
		dataRow.cveTypes = strings.Join(d.VulnerabilityTypes(), " ")
		// query to IsFixable should not have Fixable field
		rawQueryWithoutFixable := resolvers.FilterFieldFromRawQuery(rawQuery, search.Fixable)
		isFixable, err := d.IsFixable(ctx, rawQueryWithoutFixable)
		if err != nil {
			errorList.AddError(err)
		}
		dataRow.fixable = strconv.FormatBool(isFixable)
		dataRow.cvssScore = fmt.Sprintf("%.2f (%s)", d.Cvss(ctx), d.ScoreVersion(ctx))
		envImpact, err := d.EnvImpact(ctx)
		if err != nil {
			errorList.AddError(err)
		}
		dataRow.envImpact = fmt.Sprintf("%.2f", envImpact*100)
		dataRow.impactScore = fmt.Sprintf("%.2f", d.ImpactScore(ctx))
		// Entity counts should be scoped to CVE only
		clusterCount, err := d.ClusterCount(ctx, resolvers.RawQuery{})
		if err != nil {
			errorList.AddError(err)
		}
		dataRow.clusterCount = fmt.Sprint(clusterCount)
		scannedTime, err := d.LastScanned(ctx)
		if err != nil {
			errorList.AddError(err)
		}
		dataRow.scannedTime = csv.FromGraphQLTime(scannedTime)
		publishedTime, err := d.PublishedOn(ctx)
		if err != nil {
			errorList.AddError(err)
		}
		dataRow.publishedTime = csv.FromGraphQLTime(publishedTime)
		dataRow.summary = d.Summary(ctx)

		cveRows = append(cveRows, dataRow)
		if err := errorList.ToError(); err != nil {
			log.Errorf("failed to generate complete csv entry for cve %s: %v", dataRow.cve, err)
		}
	}
	return cveRows, nil
}
