package scheduler

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/csv"
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
	}
)

type vulnObj struct {
	Cve               string        `json:"cve,omitempty"`
	Severity          string        `json:"severity,omitempty"`
	FixedByVersion    string        `json:"fixedByVersion,omitempty"`
	IsFixable         bool          `json:"isFixable,omitempty"`
	DiscoveredAtImage *graphql.Time `json:"discoveredAtImage,omitempty"`
}

type compObj struct {
	Name  string     `json:"name,omitempty"`
	Vulns []*vulnObj `json:"vulns,omitempty"`
}

type imgObj struct {
	Name       *storage.ImageName `json:"name,omitempty"`
	Components []*compObj         `json:"components,omitempty"`
}

type depObj struct {
	Cluster        *storage.Cluster `json:"cluster,omitempty"`
	Namespace      string           `json:"namespace,omitempty"`
	DeploymentName string           `json:"name,omitempty"`
	Images         []*imgObj        `json:"images,omitempty"`
}

type result struct {
	Deployments []*depObj `json:"deployments,omitempty"`
}

// Format takes in the results of vuln report query, converts to CSV and returns zipped CSV data
func Format(results []result) (*bytes.Buffer, error) {
	csvWriter := csv.NewGenericWriter(csvHeader, true)

	for _, r := range results {
		for _, d := range r.Deployments {
			for _, i := range d.Images {
				for _, c := range i.Components {
					for _, v := range c.Vulns {
						csvWriter.AddValue(csv.Value{
							d.Cluster.Name,
							d.Namespace,
							d.DeploymentName,
							i.Name.FullName,
							c.Name,
							v.Cve,
							strconv.FormatBool(v.IsFixable),
							v.FixedByVersion,
							v.Severity,
							v.DiscoveredAtImage.Format(time.RFC822),
						})
					}
				}
			}
		}
	}

	var buf bytes.Buffer
	err := csvWriter.WriteBytes(&buf)
	if err != nil {
		return &bytes.Buffer{}, errors.Wrap(err, "error creating csv report")
	}

	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)
	zipFile, err := zipWriter.Create(fmt.Sprintf("RHACS_Vulnerability_Report_%s.csv", time.Now().Format("02_January_2006")))
	if err != nil {
		return &bytes.Buffer{}, errors.Wrap(err, "unable to create a zip file of the vuln report")

	}
	_, err = zipFile.Write(buf.Bytes())
	if err != nil {
		return &bytes.Buffer{}, errors.Wrap(err, "unable to create a zip file of the vuln report")
	}
	err = zipWriter.Close()
	if err != nil {
		return &bytes.Buffer{}, errors.Wrap(err, "unable to create a zip file of the vuln report")
	}
	return &zipBuf, nil
}
