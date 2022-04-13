package csv

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	clusterMappings "github.com/stackrox/stackrox/central/cluster/index/mappings"
	componentCVEEdgeMappings "github.com/stackrox/stackrox/central/componentcveedge/mappings"
	cveMappings "github.com/stackrox/stackrox/central/cve/mappings"
	"github.com/stackrox/stackrox/central/graphql/resolvers"
	"github.com/stackrox/stackrox/central/graphql/resolvers/loaders"
	componentMappings "github.com/stackrox/stackrox/central/imagecomponent/mappings"
	imageComponentEdgeMappings "github.com/stackrox/stackrox/central/imagecomponentedge/mappings"
	nsMappings "github.com/stackrox/stackrox/central/namespace/index/mappings"
	nodeMappings "github.com/stackrox/stackrox/central/node/index/mappings"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/csv"
	"github.com/stackrox/stackrox/pkg/errorhelpers"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/search"
	deploymentMappings "github.com/stackrox/stackrox/pkg/search/options/deployments"
	imageMappings "github.com/stackrox/stackrox/pkg/search/options/images"
	"github.com/stackrox/stackrox/pkg/search/parser"
	"github.com/stackrox/stackrox/pkg/search/scoped"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	nsOnlyOptionsMap = search.Difference(nsMappings.OptionsMap, clusterMappings.OptionsMap)

	deploymentOnlyOptionsMap = search.Difference(deploymentMappings.OptionsMap,
		search.CombineOptionsMaps(
			clusterMappings.OptionsMap,
			nsMappings.OptionsMap,
			imageMappings.OptionsMap))

	imageOnlyOptionsMap = search.Difference(
		imageMappings.OptionsMap,
		search.CombineOptionsMaps(
			imageComponentEdgeMappings.OptionsMap,
			componentMappings.OptionsMap,
			componentCVEEdgeMappings.OptionsMap,
			cveMappings.OptionsMap,
		),
	)

	nodeOnlyOptionsMap = search.Difference(
		nodeMappings.OptionsMap,
		search.CombineOptionsMaps(
			imageComponentEdgeMappings.OptionsMap,
			componentMappings.OptionsMap,
			componentCVEEdgeMappings.OptionsMap,
			cveMappings.OptionsMap,
		),
	)

	once       sync.Once
	csvHandler *handlerImpl

	csvHeader = []string{
		"CVE",
		"CVE Type(s)",
		"Fixable",
		"CVSS Score",
		"Env Impact (%s)",
		"Impact Score",
		"Deployments",
		"Images",
		"Nodes",
		"Components",
		"Scanned",
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
		// CVEs must be scoped from lowest entities to highest entities. DO NOT CHANGE THE ORDER.
		searchWrappers: []*searchWrapper{
			{v1.SearchCategory_IMAGE_COMPONENTS, componentMappings.OptionsMap, resolver.ImageComponentDataStore},
			{v1.SearchCategory_IMAGES, imageOnlyOptionsMap, resolver.ImageDataStore},
			{v1.SearchCategory_DEPLOYMENTS, deploymentOnlyOptionsMap, resolver.DeploymentDataStore},
			{v1.SearchCategory_NAMESPACES, nsOnlyOptionsMap, resolver.NamespaceDataStore},
			{v1.SearchCategory_NODES, nodeOnlyOptionsMap, resolver.NodeGlobalDataStore},
			{v1.SearchCategory_CLUSTERS, clusterMappings.OptionsMap, resolver.ClusterDataStore},
		},
	}
}

type searchWrapper struct {
	category   v1.SearchCategory
	optionsMap search.OptionsMap
	searcher   search.Searcher
}

type cveRow struct {
	cveID           string
	cveTypes        string
	fixable         string
	cvssScore       string
	envImpact       string
	impactScore     string
	deploymentCount string
	imageCount      string
	nodeCount       string
	componentCount  string
	scannedTime     string
	publishedTime   string
	summary         string
}

type csvResults struct {
	*csv.GenericWriter
}

func newCSVResults(header []string, sort bool) csvResults {
	return csvResults{
		GenericWriter: csv.NewGenericWriter(header, sort),
	}
}

func (c *csvResults) addRow(row cveRow) {
	// cve, cveTypes, fixable, cvss score, env impact, impact score, deployments, images, nodes, components, scanned time, published time, summary
	value := []string{
		row.cveID,
		row.cveTypes,
		row.fixable,
		row.cvssScore,
		row.envImpact,
		row.impactScore,
		row.deploymentCount,
		row.imageCount,
		row.nodeCount,
		row.componentCount,
		row.scannedTime,
		row.publishedTime,
		row.summary,
	}

	c.AddValue(value)
}

// CVECSVHandler is an HTTP handlerImpl that outputs CSV exports of CVE data for Vuln Mgmt
func CVECSVHandler() http.HandlerFunc {
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
		vulnResolvers, err := csvHandler.resolver.Vulnerabilities(ctx, paginatedQuery)
		if err != nil {
			csv.WriteError(w, http.StatusInternalServerError, err)
			log.Errorf("unable to get vulnerabilities for csv export: %v", err)
			return
		}

		postSortRequired := paginatedQuery.Pagination == nil ||
			paginatedQuery.Pagination.SortOption == nil ||
			paginatedQuery.Pagination.SortOption.Field == nil

		output := newCSVResults(csvHeader, postSortRequired)
		for _, d := range vulnResolvers {
			var errorList errorhelpers.ErrorList
			dataRow := cveRow{}
			dataRow.cveID = d.CVE(ctx)
			dataRow.cveTypes = strings.Join(d.VulnerabilityTypes(), " ")
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
			deploymentCount, err := d.DeploymentCount(ctx, resolvers.RawQuery{})
			if err != nil {
				errorList.AddError(err)
			}
			dataRow.deploymentCount = fmt.Sprint(deploymentCount)
			// Entity counts should be scoped to CVE only
			imageCount, err := d.ImageCount(ctx, resolvers.RawQuery{})
			if err != nil {
				errorList.AddError(err)
			}
			dataRow.imageCount = fmt.Sprint(imageCount)
			// Entity counts should be scoped to CVE only
			nodeCount, err := d.NodeCount(ctx, resolvers.RawQuery{})
			if err != nil {
				errorList.AddError(err)
			}
			dataRow.nodeCount = fmt.Sprint(nodeCount)
			// Entity counts should be scoped to CVE only
			componentCount, err := d.ComponentCount(ctx, resolvers.RawQuery{})
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
				log.Errorf("failed to generate complete csv entry for cve %s: %v", dataRow.cveID, err)
			}
		}

		output.Write(w, "cve_export")
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
		// Note that the resource categories are ordered from COMPONENTS to CLUSTERS.
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
