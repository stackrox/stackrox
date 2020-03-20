package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers"
	"github.com/stackrox/rox/central/graphql/resolvers/loaders"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sliceutils"
)

var (
	log = logging.LoggerForModule()
)

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

		q := buildQueryFromParams(r.URL.Query())
		rawQuery := resolvers.RawQuery{Query: &q}

		resolver := resolvers.New()
		vulnResolvers, err := resolver.Vulnerabilities(ctx, resolvers.PaginatedQuery{Query: &q})
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err)
			log.Errorf("unable to get vulnerabilities for csv export: %v", err)
			return
		}

		var output csvResults
		var errorList errorhelpers.ErrorList
		output.header = []string{"CVE", "Fixable", "CVSS Score", "Env Impact (%)", "Impact Score", "Deployments", "Images", "Components", "Scanned", "Published", "Summary"}
		for _, d := range vulnResolvers {
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
