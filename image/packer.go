package image

import (
	"path"
	"strings"
	"text/template"

	"github.com/gobuffalo/packd"
	"github.com/gobuffalo/packr"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/templates"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

const templatePath = "templates"

// These are the go based files from packr
var (
	K8sBox       = packr.NewBox("./templates/kubernetes")
	OpenshiftBox = packr.NewBox("./templates/openshift")
	AssetBox     = packr.NewBox("./assets")

	allBoxes = []*packr.Box{
		&K8sBox,
		&OpenshiftBox,
		&AssetBox,
	}
)

// LoadFileContents resolves a given file's contents across all boxes.
func LoadFileContents(filename string) (string, error) {
	for _, box := range allBoxes {
		boxPath := strings.TrimRight(strings.TrimPrefix(box.Path, "./"), "/") + "/"
		if strings.HasPrefix(filename, boxPath) {
			relativeFilename := strings.TrimPrefix(filename, boxPath)
			return box.FindString(relativeFilename)
		}
	}
	return "", errors.Errorf("file %q could not be located in any box", filename)
}

// ReadFileAndTemplate reads and renders the template for the file
func ReadFileAndTemplate(pathToFile string, funcs template.FuncMap) (*template.Template, error) {
	templatePath := path.Join(templatePath, pathToFile)
	contents, err := LoadFileContents(templatePath)
	if err != nil {
		return nil, err
	}

	tpl := template.New(templatePath)
	if funcs != nil {
		tpl = tpl.Funcs(funcs)
	}
	return tpl.Parse(contents)
}

func mustGetChart(box packr.Box, prefix string) *chart.Chart {
	ch, err := getChart(box, prefix)
	utils.Must(err)
	return ch
}

// GetCentralChart returns the Helm chart for Central
func GetCentralChart() *chart.Chart {
	return mustGetChart(K8sBox, "helm/centralchart/")
}

// GetScannerChart returns the Helm chart for the scanner
func GetScannerChart() *chart.Chart {
	return mustGetChart(K8sBox, "helm/scannerchart/")
}

// GetScannerV2Chart returns the Helm chart for ScannerV2
func GetScannerV2Chart() *chart.Chart {
	return mustGetChart(K8sBox, "helm/scannerv2chart/")
}

// GetMonitoringChart returns the Helm chart for Monitoring
func GetMonitoringChart() *chart.Chart {
	return mustGetChart(K8sBox, "helm/monitoringchart/")
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
