package csv

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	clusterMappings "github.com/stackrox/rox/central/cluster/index/mappings"
	clusterCveCsv "github.com/stackrox/rox/central/cve/cluster/csv"
	csvCommon "github.com/stackrox/rox/central/cve/common/csv"
	imageCveCsv "github.com/stackrox/rox/central/cve/image/csv"
	nodeCveCsv "github.com/stackrox/rox/central/cve/node/csv"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	componentMappings "github.com/stackrox/rox/central/imagecomponent/mappings"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/csv"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
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
			csvCommon.NewSearchWrapper(v1.SearchCategory_NODES, csvCommon.NodeOnlyOptionsMap, resolver.NodeDataStore),
			csvCommon.NewSearchWrapper(v1.SearchCategory_CLUSTERS, clusterMappings.OptionsMap, resolver.ClusterDataStore),
		},
	)
}

type cveRow struct {
	CVE             string
	CveTypes        string
	Fixable         string
	CvssScore       string
	EnvImpact       string
	ImpactScore     string
	DeploymentCount string
	ImageCount      string
	NodeCount       string
	ComponentCount  string
	ScannedTime     string
	PublishedTime   string
	Summary         string
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
	// cve, CveTypes, fixable, cvss score, env impact, impact score, deployments, images, nodes, components, scanned time, published time, summary
	value := []string{
		row.CVE,
		row.CveTypes,
		row.Fixable,
		row.CvssScore,
		row.EnvImpact,
		row.ImpactScore,
		row.DeploymentCount,
		row.ImageCount,
		row.NodeCount,
		row.ComponentCount,
		row.ScannedTime,
		row.PublishedTime,
		row.Summary,
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

		cveRows, err := cveCSVRows(loaders.WithLoaderContext(r.Context()), query, rawQuery, paginatedQuery)
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
		filename := time.Now().Format("cve_export_2006_01_02_15_04_05")
		output.Write(w, filename)
	}
}

func cveCSVRows(c context.Context, query *v1.Query, rawQuery resolvers.RawQuery, paginatedQuery resolvers.PaginatedQuery) ([]*cveRow, error) {
	if csvHandler == nil {
		return nil, errors.New("Handler not initialized")
	}

	cveType, found := search.GetFieldValueFromQuery(rawQuery.String(), search.CVEType)
	if !found {
		return nil, errors.New("'CVE Type' filter required but not found in input query")
	}

	if _, ok := storage.CVE_CVEType_value[cveType]; !ok || cveType == storage.CVE_UNKNOWN_CVE.String() {
		return nil, errors.Errorf("Unexpected value for 'CVE Type' filter. Value should be one of '%s', '%s', '%s', '%s', '%s'",
			storage.CVE_IMAGE_CVE.String(), storage.CVE_NODE_CVE.String(), storage.CVE_K8S_CVE.String(), storage.CVE_OPENSHIFT_CVE.String(), storage.CVE_ISTIO_CVE.String())
	}

	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		return cveCSVRowsFromLegacyVulnResolver(c, query, rawQuery, paginatedQuery)
	}

	switch cveType {
	case storage.CVE_IMAGE_CVE.String():
		imageCveRows, err := imageCveCsv.ImageCVECSVRows(c, query, rawQuery, paginatedQuery)
		if err != nil {
			return nil, err
		}
		return imageCVERowsToCVERows(imageCveRows), nil
	case storage.CVE_NODE_CVE.String():
		nodeCveRows, err := nodeCveCsv.NodeCVECSVRows(c, query, rawQuery, paginatedQuery)
		if err != nil {
			return nil, err
		}
		return nodeCVERowsToCVERows(nodeCveRows), nil
	case storage.CVE_K8S_CVE.String(), storage.CVE_OPENSHIFT_CVE.String(), storage.CVE_ISTIO_CVE.String():
		clusterCveRows, err := clusterCveCsv.ClusterCVECSVRows(c, query, rawQuery, paginatedQuery)
		if err != nil {
			return nil, err
		}
		return clusterCVERowsToCVERows(clusterCveRows), nil
	default:
		return nil, errors.Errorf("Unhandled CVEType '%s'", cveType)
	}
}

func cveCSVRowsFromLegacyVulnResolver(c context.Context, query *v1.Query, rawQuery resolvers.RawQuery, paginatedQuery resolvers.PaginatedQuery) ([]*cveRow, error) {
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
		dataRow.CVE = d.CVE(ctx)
		dataRow.CveTypes = strings.Join(d.VulnerabilityTypes(), " ")
		isFixable, err := d.IsFixable(ctx, rawQuery)
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
		deploymentCount, err := d.DeploymentCount(ctx, resolvers.RawQuery{})
		if err != nil {
			errorList.AddError(err)
		}
		dataRow.DeploymentCount = fmt.Sprint(deploymentCount)
		// Entity counts should be scoped to CVE only
		imageCount, err := d.ImageCount(ctx, resolvers.RawQuery{})
		if err != nil {
			errorList.AddError(err)
		}
		dataRow.ImageCount = fmt.Sprint(imageCount)
		// Entity counts should be scoped to CVE only
		nodeCount, err := d.NodeCount(ctx, resolvers.RawQuery{})
		if err != nil {
			errorList.AddError(err)
		}
		dataRow.NodeCount = fmt.Sprint(nodeCount)
		// Entity counts should be scoped to CVE only
		componentCount, err := d.ComponentCount(ctx, resolvers.RawQuery{})
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

func imageCVERowsToCVERows(imageCveRows []*imageCveCsv.ImageCVERow) []*cveRow {
	cveRows := make([]*cveRow, 0, len(imageCveRows))
	for _, d := range imageCveRows {
		dataRow := &cveRow{}
		dataRow.CVE = d.CVE
		dataRow.CveTypes = storage.CVE_IMAGE_CVE.String()
		dataRow.Fixable = d.Fixable
		dataRow.CvssScore = d.CvssScore
		dataRow.EnvImpact = d.EnvImpact
		dataRow.ImpactScore = d.ImpactScore
		dataRow.DeploymentCount = d.DeploymentCount
		dataRow.ImageCount = d.ImageCount
		dataRow.NodeCount = "0"
		dataRow.ComponentCount = d.ComponentCount
		dataRow.ScannedTime = d.ScannedTime
		dataRow.PublishedTime = d.PublishedTime
		dataRow.Summary = d.Summary

		cveRows = append(cveRows, dataRow)
	}
	return cveRows
}

func nodeCVERowsToCVERows(nodeCveRows []*nodeCveCsv.NodeCVERow) []*cveRow {
	cveRows := make([]*cveRow, 0, len(nodeCveRows))
	for _, d := range nodeCveRows {
		dataRow := &cveRow{}
		dataRow.CVE = d.CVE
		dataRow.CveTypes = storage.CVE_NODE_CVE.String()
		dataRow.Fixable = d.Fixable
		dataRow.CvssScore = d.CvssScore
		dataRow.EnvImpact = d.EnvImpact
		dataRow.ImpactScore = d.ImpactScore
		dataRow.DeploymentCount = "0"
		dataRow.ImageCount = "0"
		dataRow.NodeCount = d.NodeCount
		dataRow.ComponentCount = d.ComponentCount
		dataRow.ScannedTime = d.ScannedTime
		dataRow.PublishedTime = d.PublishedTime
		dataRow.Summary = d.Summary

		cveRows = append(cveRows, dataRow)
	}
	return cveRows
}

func clusterCVERowsToCVERows(clusterCveRows []*clusterCveCsv.ClusterCVERow) []*cveRow {
	cveRows := make([]*cveRow, 0, len(clusterCveRows))
	for _, d := range clusterCveRows {
		dataRow := &cveRow{}
		dataRow.CVE = d.CVE
		dataRow.CveTypes = d.CveTypes
		dataRow.Fixable = d.Fixable
		dataRow.CvssScore = d.CvssScore
		dataRow.EnvImpact = d.EnvImpact
		dataRow.ImpactScore = d.ImpactScore
		dataRow.DeploymentCount = "0"
		dataRow.ImageCount = "0"
		dataRow.NodeCount = "0"
		dataRow.ComponentCount = "0"
		dataRow.ScannedTime = d.ScannedTime
		dataRow.PublishedTime = d.PublishedTime
		dataRow.Summary = d.Summary

		cveRows = append(cveRows, dataRow)
	}
	return cveRows
}
