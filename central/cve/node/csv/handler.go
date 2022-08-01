package csv

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/csv"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/parser"
	"github.com/stackrox/rox/pkg/search/scoped"

	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	nodeOnlyOptionsMap = search.Difference(
		schema.NodesSchema.OptionsMap,
		search.CombineOptionsMaps(
			schema.NodeComponentEdgesSchema.OptionsMap,
			//imageComponentEdgeMappings.OptionsMap,
			schema.NodeComponentsSchema.OptionsMap,
			//componentMappings.OptionsMap,
			schema.NodeComponentsCvesEdgesSchema.OptionsMap,
			//componentCVEEdgeMappings.OptionsMap,
			schema.NodeCvesSchema.OptionsMap,
			//cveMappings.OptionsMap,
		),
	)

	once       sync.Once
	csvHandler *handlerImpl

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

type handlerImpl struct {
	resolver       *resolvers.Resolver
	searchWrappers []*searchWrapper
}

func initialize() {
	csvHandler = newHandler(resolvers.New())
}

func newHandler(resolver *resolvers.Resolver) *handlerImpl {
	return &handlerImpl{
		resolver: resolver,
		// Node CVEs must be scoped from lowest entities to highest entities. DO NOT CHANGE THE ORDER.
		searchWrappers: []*searchWrapper{
			{v1.SearchCategory_NODE_COMPONENTS, schema.NodeComponentsSchema.OptionsMap, resolver.NodeComponentDataStore},
			{v1.SearchCategory_NODES, nodeOnlyOptionsMap, resolver.NodeGlobalDataStore},
			{v1.SearchCategory_CLUSTERS, schema.ClustersSchema.OptionsMap, resolver.ClusterDataStore},
		},
	}
}

type searchWrapper struct {
	category   v1.SearchCategory
	optionsMap search.OptionsMap
	searcher   search.Searcher
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

func (c *csvResults) addRow(row nodeCveRow) {
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

func NodeCVECSVHandler() http.HandlerFunc {
	once.Do(initialize)

	return func(w http.ResponseWriter, r *http.Request) {
		query, rQuery, err := parser.ParseURLQuery(r.URL.Query())
		if err != nil {
			csv.WriteError(w, http.StatusBadRequest, err)
			return
		}

		ctx, err := csvHandler.getScopeContext(loaders.WithLoaderContext(r.Context()), query)
		if err != nil {
			csv.WriteError(w, http.StatusInternalServerError, err)
			log.Errorf("unable to determine resource scope for query %q: %v", query.String(), err)
			return
		}

		rawQuery, paginatedQuery := resolvers.V1RawQueryAsResolverQuery(rQuery)
		vulnResolvers, err := csvHandler.resolver.NodeVulnerabilities(ctx, paginatedQuery)
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
			isFixable, err := d.IsFixable(ctx, rawQuery)
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

func (h *handlerImpl) getScopeContext(ctx context.Context, query *v1.Query) (context.Context, error) {
	if _, ok := scoped.GetScope(ctx); ok {
		return ctx, nil
	}

	cloned := query.Clone()
	// Remove pagination since we are only determining the resource category which should scope the query.
	cloned.Pagination = nil
	for _, searchWrapper := range h.searchWrappers {
		// Filter the query by resource categories to determine the category that should scope the query.
		// Note that the resource categories are ordered from NODE COMPONENTS to CLUSTERS.
		filteredQ, _ := search.FilterQueryWithMap(cloned, searchWrapper.optionsMap)
		if filteredQ == nil {
			continue
		}

		result, err := searchWrapper.searcher.Search(ctx, filteredQ)
		if err != nil {
			return nil, err
		}

		if len(result) == 0 {
			continue
		}

		// Add searchWrapper only if we get exactly one match. Currently only scoping by one resource is supported in search.
		if len(result) == 1 {
			return scoped.Context(ctx, scoped.Scope{Level: searchWrapper.category, ID: result[0].ID}), nil
		}
	}
	return ctx, nil
}
