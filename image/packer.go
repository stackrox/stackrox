package image

import (
	"strings"

	"github.com/gobuffalo/packd"
	"github.com/gobuffalo/packr"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

// These are the go based files from packr
var (
	SwarmBox     = packr.NewBox("./templates/swarm")
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

// GetClairifyChart returns the Helm chart for Clairify
func GetClairifyChart() *chart.Chart {
	ch, err := getChart(K8sBox, "helm/clairifychart/")
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

func getChart(box packr.Box, prefix string) (*chart.Chart, error) {
	var chartFiles []*chartutil.BufferedFile
	err := box.WalkPrefix(prefix, func(name string, file packd.File) error {
		trimmedPath := strings.TrimPrefix(name, prefix)
		chartFiles = append(chartFiles, &chartutil.BufferedFile{
			Name: trimmedPath,
			Data: []byte(file.String()),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return chartutil.LoadFiles(chartFiles)
}
