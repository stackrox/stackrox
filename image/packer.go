package image

import (
	"strings"
	"text/template"

	"github.com/gobuffalo/packd"
	"github.com/gobuffalo/packr"
	"github.com/stackrox/rox/pkg/templates"
	"github.com/stackrox/rox/pkg/version"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

// These are the go based files from packr
var (
	K8sBox       = packr.NewBox("./templates/kubernetes")
	OpenshiftBox = packr.NewBox("./templates/openshift")
	AssetBox     = packr.NewBox("./assets")
)

// GetCentralChart returns the Helm chart for Central
func GetCentralChart() *chart.Chart {
	ch, err := getChart(K8sBox, "helm/centralchart/")
	if err != nil {
		panic(err)
	}
	return ch
}

// GetScannerChart returns the Helm chart for the scanner
func GetScannerChart() *chart.Chart {
	ch, err := getChart(K8sBox, "helm/scannerchart/")
	if err != nil {
		panic(err)
	}
	return ch
}

// GetMonitoringChart returns the Helm chart for Monitoring
func GetMonitoringChart() *chart.Chart {
	chart, err := getChart(K8sBox, "helm/monitoringchart/")
	if err != nil {
		panic(err)
	}
	return chart
}

// We need to stamp in the version to the Chart.yaml files prior to loading the chart
// or it will fail
func getChart(box packr.Box, prefix string) (*chart.Chart, error) {
	var chartFiles []*chartutil.BufferedFile
	err := box.WalkPrefix(prefix, func(name string, file packd.File) error {
		trimmedPath := strings.TrimPrefix(name, prefix)
		data := []byte(file.String())
		// if chart file, then render the version into it
		if trimmedPath == "Chart.yaml" {
			t, err := template.New("chart").Parse(file.String())
			if err != nil {
				return err
			}
			data, err = templates.ExecuteToBytes(t, map[string]string{
				"Version": version.GetMainVersion(),
			})
			if err != nil {
				return err
			}
		}
		chartFiles = append(chartFiles, &chartutil.BufferedFile{
			Name: trimmedPath,
			Data: data,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return chartutil.LoadFiles(chartFiles)
}
