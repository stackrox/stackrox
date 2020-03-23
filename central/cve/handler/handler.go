package handler

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/graph-gophers/graphql-go"
	clusterMappings "github.com/stackrox/rox/central/cluster/index/mappings"
	clusterCVEEdgeMappings "github.com/stackrox/rox/central/clustercveedge/mappings"
	componentCVEEdgeMappings "github.com/stackrox/rox/central/componentcveedge/mappings"
	cveMappings "github.com/stackrox/rox/central/cve/mappings"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	componentMappings "github.com/stackrox/rox/central/imagecomponent/mappings"
	imageComponentEdgeMappings "github.com/stackrox/rox/central/imagecomponentedge/mappings"
	nsMappings "github.com/stackrox/rox/central/namespace/index/mappings"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
	deploymentMappings "github.com/stackrox/rox/pkg/search/options/deployments"
	imageMappings "github.com/stackrox/rox/pkg/search/options/images"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
)

var (
	log = logging.LoggerForModule()

	nsOnlyOptionsMap = search.Difference(nsMappings.OptionsMap, clusterMappings.OptionsMap)

	deploymentOnlyOptionsMap = search.Difference(deploymentMappings.OptionsMap,
		search.CombineOptionsMaps(
			clusterMappings.OptionsMap,
			nsMappings.OptionsMap,
			imageMappings.ImageOnlyOptionsMap))

	imageOnlyOptionsMap = search.Difference(
		imageMappings.ImageOnlyOptionsMap,
		search.CombineOptionsMaps(
			imageComponentEdgeMappings.OptionsMap,
			componentMappings.OptionsMap,
			componentCVEEdgeMappings.OptionsMap,
			cveMappings.OptionsMap,
		),
	)

	// CVEs must be scoped from lowest entities to highest entities. DO NOT CHANGE THE ORDER.
	scopeLevels = []scopeLevel{
		{v1.SearchCategory_IMAGE_COMPONENTS, componentMappings.OptionsMap},
		{v1.SearchCategory_IMAGES, imageOnlyOptionsMap},
		{v1.SearchCategory_DEPLOYMENTS, deploymentOnlyOptionsMap},
		{v1.SearchCategory_NAMESPACES, nsOnlyOptionsMap},
		{v1.SearchCategory_CLUSTERS, clusterMappings.OptionsMap},
	}

	// idFields holds id search field label for various search category
	idFields = set.NewStringSet(search.ClusterID.String(),
		search.NamespaceID.String(),
		search.DeploymentID.String(),
		search.ImageSHA.String(),
		search.ComponentID.String())
)

type scopeLevel struct {
	category   v1.SearchCategory
	optionsMap search.OptionsMap
}

func writeErr(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(code)
	fmt.Fprint(w, err)
}

type cveRow struct {
	cveID           string
	fixable         string
	cvssScore       string
	envImpact       string
	impactScore     string
	deploymentCount string
	imageCount      string
	componentCount  string
	scannedTime     string
	publishedTime   string
	summary         string
}

type csvResults struct {
	header []string
	values [][]string
}

func (c *csvResults) write(writer *csv.Writer) {
	sort.Slice(c.values, func(i, j int) bool {
		first, second := c.values[i], c.values[j]
		for len(first) > 0 {
			// first has more values, so greater
			if len(second) == 0 {
				return false
			}
			if first[0] < second[0] {
				return true
			}
			if first[0] > second[0] {
				return false
			}
			first = first[1:]
			second = second[1:]
		}
		// second has more values, so first is lesser
		return len(second) > 0
	})
	header := sliceutils.StringClone(c.header)
	header[0] = "\uFEFF" + header[0]
	_ = writer.Write(header)
	for _, v := range c.values {
		_ = writer.Write(v)
	}
}

func (c *csvResults) addRow(row cveRow) {
	// cve, fixable, cvss score, env impact, impact score, deployments, images, components, scanned time, published time, summary
	value := []string{
		row.cveID,
		row.fixable,
		row.cvssScore,
		row.envImpact,
		row.impactScore,
		row.deploymentCount,
		row.imageCount,
		row.componentCount,
		row.scannedTime,
		row.publishedTime,
		row.summary,
	}

	c.values = append(c.values, value)
}

func fromTS(timestamp *graphql.Time) string {
	if timestamp == nil {
		return "-"
	}
	return timestamp.Time.Format(time.RFC1123)
}

func buildQueryFromParams(values url.Values) string {
	var pairs []string
	for k, v := range values {
		vs := strings.Join(v, ",")
		pair := strings.Join([]string{k, vs}, ":")
		pairs = append(pairs, pair)
	}
	return strings.Join(pairs, "+")
}

// CSVHandler is an HTTP handler that outputs CSV exports of CVE data for Vuln Mgmt
func CSVHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := loaders.WithLoaderContext(r.Context())

		queryString := buildQueryFromParams(r.URL.Query())
		rawQuery := resolvers.RawQuery{Query: &queryString}
		q, err := rawQuery.AsV1QueryOrEmpty()
		if err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}

		resolver := resolvers.New()
		vulnResolvers, err := getVulns(ctx, resolver, q)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			log.Errorf("unable to get vulnerabilities for csv export: %v", err)
			return
		}

		var output csvResults
		output.header = []string{"CVE", "Fixable", "CVSS Score", "Env Impact (%)", "Impact Score", "Deployments", "Images", "Components", "Scanned", "Published", "Summary"}
		for _, d := range vulnResolvers {
			var errorList errorhelpers.ErrorList
			dataRow := cveRow{}
			dataRow.cveID = d.Cve(ctx)
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
			deploymentCount, err := d.DeploymentCount(ctx, rawQuery)
			if err != nil {
				errorList.AddError(err)
			}
			dataRow.deploymentCount = fmt.Sprint(deploymentCount)
			imageCount, err := d.ImageCount(ctx, rawQuery)
			if err != nil {
				errorList.AddError(err)
			}
			dataRow.imageCount = fmt.Sprint(imageCount)
			componentCount, err := d.ComponentCount(ctx, rawQuery)
			if err != nil {
				errorList.AddError(err)
			}
			dataRow.componentCount = fmt.Sprint(componentCount)
			scannedTime, err := d.LastScanned(ctx)
			if err != nil {
				errorList.AddError(err)
			}
			dataRow.scannedTime = fromTS(scannedTime)
			publishedTime, err := d.PublishedOn(ctx)
			if err != nil {
				errorList.AddError(err)
			}
			dataRow.publishedTime = fromTS(publishedTime)
			dataRow.summary = d.Summary(ctx)

			output.addRow(dataRow)

			log.Errorf("failed to generate complete csv entry for cve %s: %v", dataRow.cveID, errorList.ToError())
		}

		w.Header().Set("Content-Type", `text/csv; charset="utf-8"`)
		w.Header().Set("Content-Disposition", `attachment; filename="cve_export.csv"`)
		w.WriteHeader(http.StatusOK)
		cw := csv.NewWriter(w)
		cw.UseCRLF = true
		output.write(cw)
		cw.Flush()
	}
}

func getVulns(ctx context.Context, resolver *resolvers.Resolver, q *v1.Query) ([]resolvers.VulnerabilityResolver, error) {
	results, err := runAsScopedQuery(ctx, resolver, q)
	if err != nil {
		return nil, err
	}
	cveQuery := search.NewQueryBuilder().AddExactMatches(search.CVE, search.ResultsToIDs(results)...).Query()
	return resolver.Vulnerabilities(ctx, resolvers.PaginatedQuery{Query: &cveQuery})
}

func runAsScopedQuery(ctx context.Context, resolver *resolvers.Resolver, query *v1.Query) ([]search.Result, error) {
	// We handle scoping per entity only. For example, for query such as `Deployment:r/abc.*`, scoping is not performed.
	// This is done to match csv results with cve list page.
	scopedCtxs, err := getScopeContexts(ctx, resolver, query)
	if err != nil {
		return nil, err
	}

	// This is either incoming ctx or scoped context
	ctx = scopedCtxs[0]
	return resolver.CVEDataStore.Search(ctx, query)
}

func getScopeContexts(ctx context.Context, resolver *resolvers.Resolver, query *v1.Query) ([]context.Context, error) {
	// query does not need scoping
	if !isScopable(query) {
		return []context.Context{ctx}, nil
	}

	for _, scopeLevel := range scopeLevels {
		if !scopeByCategory(query, scopeLevel) {
			continue
		}

		scopeIDs, err := getScopeIDs(ctx, resolver, scopeLevel.category, query)
		if err != nil {
			return nil, err
		}

		ret := make([]context.Context, 0, len(scopeIDs))
		for _, id := range scopeIDs {
			ret = append(ret, scoped.Context(ctx, scoped.Scope{Level: scopeLevel.category, ID: id}))
		}
		return ret, nil
	}
	return []context.Context{ctx}, nil
}

func scopeByCategory(query *v1.Query, scopeLevel scopeLevel) bool {
	local := proto.Clone(query).(*v1.Query)
	notCVEQuery, _ := search.FilterQueryWithMap(local, scopeLevel.optionsMap)
	return notCVEQuery != nil
}

func isScopable(query *v1.Query) bool {
	local := proto.Clone(query).(*v1.Query)
	filtered, _ := search.InverseFilterQueryWithMap(local, search.CombineOptionsMaps(
		cveMappings.OptionsMap, componentCVEEdgeMappings.OptionsMap, clusterCVEEdgeMappings.OptionsMap))
	if filtered == nil {
		return false
	}

	var containsNonIDFields bool
	search.ApplyFnToAllBaseQueries(filtered, func(bq *v1.BaseQuery) {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if ok {
			if !idFields.Contains(matchFieldQuery.MatchFieldQuery.GetField()) {
				containsNonIDFields = true
			}
		}
	})
	return !containsNonIDFields
}

func getScopeIDs(ctx context.Context, resolver *resolvers.Resolver, category v1.SearchCategory, query *v1.Query) ([]string, error) {
	var err error
	var results []search.Result
	if category == v1.SearchCategory_IMAGE_COMPONENTS {
		results, err = resolver.ImageComponentDataStore.Search(ctx, query)
	} else if category == v1.SearchCategory_IMAGES {
		results, err = resolver.ImageDataStore.Search(ctx, query)
	} else if category == v1.SearchCategory_DEPLOYMENTS {
		results, err = resolver.DeploymentDataStore.Search(ctx, query)
	} else if category == v1.SearchCategory_NAMESPACES {
		results, err = resolver.NamespaceDataStore.Search(ctx, query)
	} else if category == v1.SearchCategory_CLUSTERS {
		results, err = resolver.ClusterDataStore.Search(ctx, query)
	}

	if err != nil {
		return nil, err
	}
	return search.ResultsToIDs(results), nil
}
