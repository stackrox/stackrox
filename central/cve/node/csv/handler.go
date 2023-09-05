package csv

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

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
		"Node CVE",
		"Fixable",
		"CVSS Score",
		"Env Impact (%s)",
		"Impact Score",
		"Nodes",
		"Node Components",
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
		// Node CVEs must be scoped from lowest entities to highest entities. DO NOT CHANGE THE ORDER.
		[]*csvCommon.SearchWrapper{
			csvCommon.NewSearchWrapper(v1.SearchCategory_NODE_COMPONENTS, schema.NodeComponentsSchema.OptionsMap, resolver.NodeComponentDataStore),
			csvCommon.NewSearchWrapper(v1.SearchCategory_NODES, csvCommon.NodeOnlyOptionsMap, resolver.NodeDataStore),
			csvCommon.NewSearchWrapper(v1.SearchCategory_CLUSTERS, schema.ClustersSchema.OptionsMap, resolver.ClusterDataStore),
		},
	)
}

// NodeCVERow represents a row in Node CVE csv export
type NodeCVERow struct {
	CVE            string
	Fixable        string
	CvssScore      string
	EnvImpact      string
	ImpactScore    string
	NodeCount      string
	ComponentCount string
	ScannedTime    string
	PublishedTime  string
	Summary        string
}

type csvResults struct {
	*csv.GenericWriter
}

func newCSVResults(header []string, sort bool) csvResults {
	return csvResults{
		GenericWriter: csv.NewGenericWriter(header, sort),
	}
}

func (c *csvResults) addRow(row *NodeCVERow) {
	// node cve, fixable, cvss score, env impact, impact score, nodes, node components, scanned time, published time, summary
	value := []string{
		row.CVE,
		row.Fixable,
		row.CvssScore,
		row.EnvImpact,
		row.ImpactScore,
		row.NodeCount,
		row.ComponentCount,
		row.ScannedTime,
		row.PublishedTime,
		row.Summary,
	}

	c.AddValue(value)
}

// NodeCVECSVHandler returns a handler func to serve csv export requests of Node CVE data for Vuln Mgmt
func NodeCVECSVHandler() http.HandlerFunc {
	once.Do(initialize)

	return func(w http.ResponseWriter, r *http.Request) {
		query, rQuery, err := parser.ParseURLQuery(r.URL.Query())
		if err != nil {
			csv.WriteErrorWithCode(w, http.StatusBadRequest, err)
			return
		}
		rawQuery, paginatedQuery := resolvers.V1RawQueryAsResolverQuery(rQuery)

		cveRows, err := NodeCVECSVRows(loaders.WithLoaderContext(r.Context()), query, rawQuery, paginatedQuery)
		if err != nil {
			csv.WriteErrorWithCode(w, http.StatusInternalServerError, err)
			return
		}

		postSortRequired := paginatedQuery.Pagination == nil ||
			paginatedQuery.Pagination.SortOption == nil ||
			paginatedQuery.Pagination.SortOption.Field == nil

		output := newCSVResults(csvHeader, postSortRequired)
		for _, row := range cveRows {
			output.addRow(row)
		}
		filename := time.Now().Format("node_cve_export_2006_01_02_15_04_05")
		output.Write(w, filename)
	}
}

// NodeCVECSVRows returns data rows for Node CVE csv export
func NodeCVECSVRows(c context.Context, query *v1.Query, rawQuery resolvers.RawQuery, paginatedQuery resolvers.PaginatedQuery) ([]*NodeCVERow, error) {
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
		log.Error("Unexpected value (nil) for resolver in Handler")
		return nil, errors.New("Resolver not initialized in handler")
	}
	vulnResolvers, err := res.NodeVulnerabilities(ctx, paginatedQuery)
	if err != nil {
		log.Errorf("unable to get node vulnerabilities for csv export: %v", err)
		return nil, err
	}

	cveRows := make([]*NodeCVERow, 0, len(vulnResolvers))
	for _, d := range vulnResolvers {
		var errorList errorhelpers.ErrorList
		dataRow := &NodeCVERow{}
		dataRow.CVE = d.CVE(ctx)
		// query to IsFixable should not have Fixable field
		rawQueryWithoutFixable := resolvers.FilterFieldFromRawQuery(rawQuery, search.Fixable)
		isFixable, err := d.IsFixable(ctx, rawQueryWithoutFixable)
		if err != nil {
			errorList.AddError(err)
		}
		dataRow.Fixable = strconv.FormatBool(isFixable)
		dataRow.CvssScore = fmt.Sprintf("%.2f (%s)", d.Cvss(ctx), d.ScoreVersion(ctx))
		envImpact, err := d.EnvImpact(ctx)
		if err != nil {
			errorList.AddError(err)
		}
		dataRow.EnvImpact = fmt.Sprintf("%.2f", envImpact*100)
		dataRow.ImpactScore = fmt.Sprintf("%.2f", d.ImpactScore(ctx))
		// Entity counts should be scoped to CVE only
		nodeCount, err := d.NodeCount(ctx, resolvers.RawQuery{})
		if err != nil {
			errorList.AddError(err)
		}
		dataRow.NodeCount = fmt.Sprint(nodeCount)
		// Entity counts should be scoped to CVE only
		componentCount, err := d.NodeComponentCount(ctx, resolvers.RawQuery{})
		if err != nil {
			errorList.AddError(err)
		}
		dataRow.ComponentCount = fmt.Sprint(componentCount)
		scannedTime, err := d.LastScanned(ctx)
		if err != nil {
			errorList.AddError(err)
		}
		dataRow.ScannedTime = csv.FromGraphQLTime(scannedTime)
		publishedTime, err := d.PublishedOn(ctx)
		if err != nil {
			errorList.AddError(err)
		}
		dataRow.PublishedTime = csv.FromGraphQLTime(publishedTime)
		dataRow.Summary = d.Summary(ctx)

		cveRows = append(cveRows, dataRow)
		if err := errorList.ToError(); err != nil {
			log.Errorf("failed to generate complete csv entry for cve %s: %v", dataRow.CVE, err)
		}
	}
	return cveRows, nil
}
