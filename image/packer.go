package image

import (
	"io"
	"io/ioutil"
	"path"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/gobuffalo/packd"
	"github.com/gobuffalo/packr"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image/sensor"
	rendererUtils "github.com/stackrox/rox/pkg/renderer/utils"
	"github.com/stackrox/rox/pkg/templates"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

const (
	templatePath                      = "templates"
	sensorChartPrefix                 = "helm/sensorchart/"
	centralChartPrefix                = "helm/centralchart/"
	scannerChartPrefix                = "helm/scannerchart/"
	monitoringChartPrefix             = "helm/monitoringchart/"
	centralChartWithDiagnosticsPrefix = "helm/centralchart-diagnostics/"
	chartYamlFile                     = "Chart.yaml"
	valuesYamlFile                    = "values.yaml"
)

// These are the go based files from packr
var (
	K8sBox   = packr.NewBox("./templates")
	AssetBox = packr.NewBox("./assets")

	allBoxes = []*packr.Box{
		&K8sBox,
		&AssetBox,
	}

	k8sScriptsFileMap = map[string]string{
		"sensor/kubernetes/sensor.sh":        "templates/sensor.sh",
		"sensor/kubernetes/delete-sensor.sh": "templates/delete-sensor.sh",
		"common/ca-setup.sh":                 "templates/ca-setup-sensor.sh",
		"common/delete-ca.sh":                "templates/delete-ca-sensor.sh",
	}

	osScriptsFileMap = map[string]string{
		"sensor/openshift/sensor.sh":        "templates/sensor.sh",
		"sensor/openshift/delete-sensor.sh": "templates/delete-sensor.sh",
		"common/ca-setup.sh":                "templates/ca-setup-sensor.sh",
		"common/delete-ca.sh":               "templates/delete-ca-sensor.sh",
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

func mustGetChart(box packr.Box, overrides map[string]func() io.ReadCloser, prefixes ...string) *chart.Chart {
	ch, err := getChart(box, prefixes, overrides)
	utils.Must(err)
	return ch
}
func mustGetSensorChart(box packr.Box, values map[string]interface{}, certs *sensor.Certs) *chart.Chart {
	ch, err := getSensorChart(box, values, certs)
	utils.Must(err)
	return ch
}

// GetCentralChart returns the Helm chart for Central
func GetCentralChart(overrides map[string]func() io.ReadCloser) *chart.Chart {
	prefixes := []string{centralChartPrefix}
	return mustGetChart(K8sBox, overrides, prefixes...)
}

// GetScannerChart returns the Helm chart for the scanner
func GetScannerChart() *chart.Chart {
	return mustGetChart(K8sBox, nil, scannerChartPrefix)
}

// GetMonitoringChart returns the Helm chart for Monitoring
func GetMonitoringChart() *chart.Chart {
	return mustGetChart(K8sBox, nil, monitoringChartPrefix)
}

// GetSensorChart returns the Helm chart for sensor
func GetSensorChart(values map[string]interface{}, certs *sensor.Certs) *chart.Chart {
	return mustGetSensorChart(K8sBox, values, certs)
}

// We need to stamp in the version to the Chart.yaml files prior to loading the chart
// or it will fail
func getChart(box packr.Box, prefixes []string, overrides map[string]func() io.ReadCloser) (*chart.Chart, error) {
	var chartFiles []*chartutil.BufferedFile
	for _, prefix := range prefixes {
		err := box.WalkPrefix(prefix, func(name string, file packd.File) error {
			trimmedPath := strings.TrimPrefix(name, prefix)
			dataReader := ioutil.NopCloser(file)

			if overrideFunc := overrides[trimmedPath]; overrideFunc != nil {
				dataReader = overrideFunc()
			}
			defer utils.IgnoreError(dataReader.Close)

			data, err := ioutil.ReadAll(dataReader)
			if err != nil {
				return errors.Wrapf(err, "failed to read file %s", trimmedPath)
			}

			// if chart file, then render the version into it
			if trimmedPath == chartYamlFile {
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
	}

	return chartutil.LoadFiles(chartFiles)
}

func processSensorChartFile(box packr.Box, path string, file packd.File, chartFiles *[]*chartutil.BufferedFile, values map[string]interface{}) error {
	if path == "main.go" || path == ".helmignore" ||
		path == "README.md" ||
		strings.HasPrefix(path, "scripts") {
		return nil
	}

	// Render the versions into the files that need it
	t, err := template.New(strings.TrimSuffix(path, ".yaml")).
		Delims("!!", "!!").Funcs(rendererUtils.BuiltinFuncs).Funcs(sprig.TxtFuncMap()).
		Parse(file.String())
	if err != nil {
		return err
	}

	data, err := templates.ExecuteToBytes(t, values)

	if err != nil {
		return err
	}

	*chartFiles = append(*chartFiles, &chartutil.BufferedFile{
		Name: path,
		Data: data,
	})
	return nil
}

func getSensorChart(box packr.Box, values map[string]interface{}, certs *sensor.Certs) (*chart.Chart, error) {
	chartFiles := make([]*chartutil.BufferedFile, 0)

	err := box.WalkPrefix(sensorChartPrefix, func(name string, file packd.File) error {
		trimmedPath := strings.TrimPrefix(name, sensorChartPrefix)
		return processSensorChartFile(box, trimmedPath, file, &chartFiles, values)
	})

	if err != nil {
		return nil, err
	}

	for path, data := range certs.Files {
		chartFiles = append(chartFiles, &chartutil.BufferedFile{
			Name: path,
			Data: data,
		})
	}

	scriptFiles, err := addScripts(box, values)
	if err != nil {
		return nil, err
	}

	chartFiles = append(chartFiles, scriptFiles...)

	return chartutil.LoadFiles(chartFiles)
}

func addScripts(box packr.Box, values map[string]interface{}) ([]*chartutil.BufferedFile, error) {
	if values["ClusterType"] == storage.ClusterType_KUBERNETES_CLUSTER.String() {
		return scripts(box, values, k8sScriptsFileMap)
	} else if values["ClusterType"] == storage.ClusterType_OPENSHIFT_CLUSTER.String() {
		return scripts(box, values, osScriptsFileMap)
	} else {
		return nil, errors.Errorf("unable to create sensor bundle, invalid cluster type for cluster %s",
			values["ClusterName"])
	}
}

func scripts(box packr.Box, values map[string]interface{}, filenameMap map[string]string) ([]*chartutil.BufferedFile, error) {
	var chartFiles []*chartutil.BufferedFile
	for srcFile, dstFile := range filenameMap {
		fileData, err := box.Find(srcFile)
		if err != nil {
			return nil, err
		}
		t, err := template.New("temp").Funcs(rendererUtils.BuiltinFuncs).Parse(string(fileData))
		if err != nil {
			return nil, err
		}
		data, err := templates.ExecuteToBytes(t, values)
		if err != nil {
			return nil, err
		}
		chartFiles = append(chartFiles, &chartutil.BufferedFile{
			Name: dstFile,
			Data: data,
		})
	}

	return chartFiles, nil
}
