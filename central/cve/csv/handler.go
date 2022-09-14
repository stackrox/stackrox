package csv

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	clusterMappings "github.com/stackrox/rox/central/cluster/index/mappings"
	csvCommon "github.com/stackrox/rox/central/cve/common/csv"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	componentMappings "github.com/stackrox/rox/central/imagecomponent/mappings"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/csv"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search/parser"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()

	once       sync.Once
	csvHandler *csvCommon.HandlerImpl

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

func initialize() {
	csvHandler = newHandler(resolvers.New())
}

func newHandler(resolver *resolvers.Resolver) *csvCommon.HandlerImpl {
	return csvCommon.NewCSVHandler(
		resolver,
		// CVEs must be scoped from lowest entities to highest entities. DO NOT CHANGE THE ORDER.
		[]*csvCommon.SearchWrapper{
			csvCommon.NewSearchWrapper(v1.SearchCategory_IMAGE_COMPONENTS, componentMappings.OptionsMap, resolver.ImageComponentDataStore),
			csvCommon.NewSearchWrapper(v1.SearchCategory_IMAGES, csvCommon.ImageOnlyOptionsMap, resolver.ImageDataStore),
			csvCommon.NewSearchWrapper(v1.SearchCategory_DEPLOYMENTS, csvCommon.DeploymentOnlyOptionsMap, resolver.DeploymentDataStore),
			csvCommon.NewSearchWrapper(v1.SearchCategory_NAMESPACES, csvCommon.NamespaceOnlyOptionsMap, resolver.NamespaceDataStore),
			csvCommon.NewSearchWrapper(v1.SearchCategory_NODES, csvCommon.NodeOnlyOptionsMap, resolver.NodeGlobalDataStore),
			csvCommon.NewSearchWrapper(v1.SearchCategory_CLUSTERS, clusterMappings.OptionsMap, resolver.ClusterDataStore),
		},
	)
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

func (c *csvResults) addRow(row *cveRow) {
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
		rawQuery, paginatedQuery := resolvers.V1RawQueryAsResolverQuery(rQuery)

		cveRows, err := CVECSVRows(loaders.WithLoaderContext(r.Context()), query, rawQuery, paginatedQuery)
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
		output.Write(w, "cve_export")
	}
}

func CVECSVRows(c context.Context, query *v1.Query, rawQuery resolvers.RawQuery, paginatedQuery resolvers.PaginatedQuery) ([]*cveRow, error) {
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
	vulnResolvers, err := res.Vulnerabilities(ctx, paginatedQuery)
	if err != nil {
		log.Errorf("unable to get vulnerabilities for csv export: %v", err)
		return nil, err
	}

	cveRows := make([]*cveRow, 0, len(vulnResolvers))
	for _, d := range vulnResolvers {
		var errorList errorhelpers.ErrorList
		dataRow := &cveRow{}
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

		cveRows = append(cveRows, dataRow)
		if err := errorList.ToError(); err != nil {
			log.Errorf("failed to generate complete csv entry for cve %s: %v", dataRow.cveID, err)
		}
	}
	return cveRows, nil
}
