package csv

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

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
			csvCommon.NewSearchWrapper(v1.SearchCategory_NODES, csvCommon.NodeOnlyOptionsMap, resolver.NodeGlobalDataStore),
			csvCommon.NewSearchWrapper(v1.SearchCategory_CLUSTERS, schema.ClustersSchema.OptionsMap, resolver.ClusterDataStore),
		},
	)
}

type nodeCveRow struct {
	cve            string
	fixable        string
	cvssScore      string
	envImpact      string
	impactScore    string
	nodeCount      string
	componentCount string
	scannedTime    string
	publishedTime  string
	summary        string
}

type csvResults struct {
	*csv.GenericWriter
}

func newCSVResults(header []string, sort bool) csvResults {
	return csvResults{
		GenericWriter: csv.NewGenericWriter(header, sort),
	}
}

func (c *csvResults) addRow(row *nodeCveRow) {
	// node cve, fixable, cvss score, env impact, impact score, nodes, node components, scanned time, published time, summary
	value := []string{
		row.cve,
		row.fixable,
		row.cvssScore,
		row.envImpact,
		row.impactScore,
		row.nodeCount,
		row.componentCount,
		row.scannedTime,
		row.publishedTime,
		row.summary,
	}

	c.AddValue(value)
}

// NodeCVECSVHandler returns a handler func to serve csv export requests of Node CVE data for Vuln Mgmt
func NodeCVECSVHandler() http.HandlerFunc {
	once.Do(initialize)

	return func(w http.ResponseWriter, r *http.Request) {
		query, rQuery, err := parser.ParseURLQuery(r.URL.Query())
		if err != nil {
			csv.WriteError(w, http.StatusBadRequest, err)
			return
		}

		ctx, err := csvHandler.GetScopeContext(loaders.WithLoaderContext(r.Context()), query)
		if err != nil {
			csv.WriteError(w, http.StatusInternalServerError, err)
			log.Errorf("unable to determine resource scope for query %q: %v", query.String(), err)
			return
		}

		rawQuery, paginatedQuery := resolvers.V1RawQueryAsResolverQuery(rQuery)
		res := csvHandler.GetResolver()
		if res == nil {
			csv.WriteError(w, http.StatusInternalServerError, err)
			log.Errorf("Unexpected value (nil) for resolver in Handler")
		}
		vulnResolvers, err := res.NodeVulnerabilities(ctx, paginatedQuery)
		if err != nil {
			csv.WriteError(w, http.StatusInternalServerError, err)
			log.Errorf("unable to get node vulnerabilities for csv export: %v", err)
			return
		}

		postSortRequired := paginatedQuery.Pagination == nil ||
			paginatedQuery.Pagination.SortOption == nil ||
			paginatedQuery.Pagination.SortOption.Field == nil

		output := newCSVResults(csvHeader, postSortRequired)
		for _, d := range vulnResolvers {
			var errorList errorhelpers.ErrorList
			dataRow := nodeCveRow{}
			dataRow.cve = d.CVE(ctx)
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
			nodeCount, err := d.NodeCount(ctx, resolvers.RawQuery{})
			if err != nil {
				errorList.AddError(err)
			}
			dataRow.nodeCount = fmt.Sprint(nodeCount)
			// Entity counts should be scoped to CVE only
			componentCount, err := d.NodeComponentCount(ctx, resolvers.RawQuery{})
			if err != nil {
				errorList.AddError(err)
			}
			dataRow.componentCount = fmt.Sprint(componentCount)
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

			output.addRow(dataRow)

			if err := errorList.ToError(); err != nil {
				log.Errorf("failed to generate complete csv entry for cve %s: %v", dataRow.cve, err)
			}
		}

		output.Write(w, "node_cve_export")
	}
}

func NodeCVECSVRows(c context.Context, query *v1.Query, rawQuery resolvers.RawQuery, paginatedQuery resolvers.PaginatedQuery) ([]*nodeCveRow, error) {
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
	vulnResolvers, err := res.NodeVulnerabilities(ctx, paginatedQuery)
	if err != nil {
		log.Errorf("unable to get vulnerabilities for csv export: %v", err)
		return nil, err
	}

	cveRows := make([]*nodeCveRow, 0, len(vulnResolvers))
	for _, d := range vulnResolvers {
		
	}
}
