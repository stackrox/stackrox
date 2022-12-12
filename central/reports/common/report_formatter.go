package common

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/csv"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/stringutils"
)

// This formatter is tightly coupled to the report query. The end goal is to use the CSVPrinter in roxctl, but
// as it stands it has some  limitations which are non trivial to fix, so in the interim we will format the report using
// the formatting logic in this file

var (
	csvHeader = []string{
		"Cluster",
		"Namespace",
		"Deployment",
		"Image",
		"Component",
		"CVE",
		"Fixable",
		"Component Upgrade",
		"Severity",
		"Discovered At",
		"Reference",
	}
)

// ImageVulnerability data used for generating vuln reports
type ImageVulnerability struct {
	Cve               string        `json:"cve,omitempty"`
	Severity          string        `json:"severity,omitempty"`
	FixedByVersion    string        `json:"fixedByVersion,omitempty"`
	IsFixable         bool          `json:"isFixable,omitempty"`
	DiscoveredAtImage *graphql.Time `json:"discoveredAtImage,omitempty"`
	Link              string        `json:"link,omitempty"`
}

// ImageComponent data used for generating vuln reports
type ImageComponent struct {
	Name                 string                `json:"name,omitempty"`
	ImageVulnerabilities []*ImageVulnerability `json:"imageVulnerabilities,omitempty"`
	Vulns                []*ImageVulnerability `json:"vulns,omitempty"`
}

// Image data used for generating vuln reports
type Image struct {
	Name            *storage.ImageName `json:"name,omitempty"`
	ImageComponents []*ImageComponent  `json:"imageComponents,omitempty"`
	Components      []*ImageComponent  `json:"components,omitempty"`
}

// Deployment data used for generating vuln reports
type Deployment struct {
	Cluster        *storage.Cluster `json:"cluster,omitempty"`
	Namespace      string           `json:"namespace,omitempty"`
	DeploymentName string           `json:"name,omitempty"`
	Images         []*Image         `json:"images,omitempty"`
}

// Result is the query results of running a single cvefields query and scope query combination
type Result struct {
	Deployments []*Deployment `json:"deployments,omitempty"`
}

// Format takes in the results of vuln report query, converts to CSV and returns zipped CSV data and
// a flag if the report is empty or not
func Format(results []Result) (*bytes.Buffer, error) {
	csvWriter := csv.NewGenericWriter(csvHeader, true)
	for _, r := range results {
		for _, d := range r.Deployments {
			for _, i := range d.Images {
				for _, c := range i.getComponents() {
					for _, v := range c.getVulnerabilities() {
						discoveredTs := "Not Available"
						if v.DiscoveredAtImage != nil {
							discoveredTs = v.DiscoveredAtImage.Time.Format("January 02, 2006")
						}
						csvWriter.AddValue(csv.Value{
							d.Cluster.Name,
							d.Namespace,
							d.DeploymentName,
							i.Name.FullName,
							c.Name,
							v.Cve,
							strconv.FormatBool(v.IsFixable),
							v.FixedByVersion,
							strings.ToTitle(stringutils.GetUpTo(v.Severity, "_")),
							discoveredTs,
							v.Link,
						})
					}
				}
			}
		}
	}

	if csvWriter.IsEmpty() {
		return nil, nil
	}

	var buf bytes.Buffer
	err := csvWriter.WriteBytes(&buf)
	if err != nil {
		return nil, errors.Wrap(err, "error creating csv report")
	}

	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)
	zipFile, err := zipWriter.Create(fmt.Sprintf("RHACS_Vulnerability_Report_%s.csv", time.Now().Format("02_January_2006")))
	if err != nil {
		return nil, errors.Wrap(err, "unable to create a zip file of the vuln report")

	}
	_, err = zipFile.Write(buf.Bytes())
	if err != nil {
		return nil, errors.Wrap(err, "unable to create a zip file of the vuln report")
	}
	err = zipWriter.Close()
	if err != nil {
		return nil, errors.Wrap(err, "unable to create a zip file of the vuln report")
	}
	return &zipBuf, nil
}

func (img *Image) getComponents() []*ImageComponent {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return img.ImageComponents
	}
	return img.Components
}

func (component *ImageComponent) getVulnerabilities() []*ImageVulnerability {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return component.ImageVulnerabilities
	}
	return component.Vulns
}
