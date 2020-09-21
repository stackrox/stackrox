package image

import (
	"io"
	"io/ioutil"
	"path"
	"strings"
	"text/template"

	"github.com/gobuffalo/packd"
	"github.com/gobuffalo/packr"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image/sensor"
	"github.com/stackrox/rox/pkg/helmtpl"
	"github.com/stackrox/rox/pkg/k8sutil/k8sobjects"
	"github.com/stackrox/rox/pkg/namespaces"
	rendererUtils "github.com/stackrox/rox/pkg/renderer/utils"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/templates"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	templatePath          = "templates"
	sensorChartPrefix     = "helm/sensorchart/"
	centralChartPrefix    = "helm/centralchart/"
	scannerChartPrefix    = "helm/scannerchart/"
	monitoringChartPrefix = "helm/monitoringchart/"
	chartYamlFile         = "Chart.yaml"
	// CentralServicesChartPrefix points to the new stackrox-central-services Helm Chart.
	CentralServicesChartPrefix = "helm/stackrox-central/"
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

func getChartTemplate(prefix string) (*helmtpl.ChartTemplate, error) {
	// Retrieve template files from box.
	chartTplFiles, err := GetFilesFromBox(K8sBox, prefix)
	if err != nil {
		return nil, errors.Wrapf(err, "fetching %s chart files from box", prefix)
	}
	chartTpl, err := helmtpl.Load(chartTplFiles)
	if err != nil {
		return nil, errors.Wrapf(err, "loading %s helmtpl", prefix)
	}

	return chartTpl, nil
}

func mustGetChart(box packr.Box, overrides map[string]func() io.ReadCloser, prefixes ...string) []*loader.BufferedFile {
	ch, err := getChartFiles(box, prefixes, overrides)
	utils.Must(err)
	return ch
}
func mustGetSensorChart(box packr.Box, values map[string]interface{}, certs *sensor.Certs) *chart.Chart {
	ch, err := getSensorChart(box, values, certs)
	utils.Must(err)
	return ch
}

// GetCentralChart returns the Helm chart for Central
func GetCentralChart(overrides map[string]func() io.ReadCloser) []*loader.BufferedFile {
	return mustGetChart(K8sBox, overrides, centralChartPrefix)
}

// GetScannerChart returns the Helm chart for the scanner
func GetScannerChart() []*loader.BufferedFile {
	return mustGetChart(K8sBox, nil, scannerChartPrefix)
}

// GetMonitoringChart returns the Helm chart for Monitoring
func GetMonitoringChart() []*loader.BufferedFile {
	return mustGetChart(K8sBox, nil, monitoringChartPrefix)
}

// GetSensorChart returns the Helm chart for sensor
func GetSensorChart(values map[string]interface{}, certs *sensor.Certs) *chart.Chart {
	return mustGetSensorChart(K8sBox, values, certs)
}

// GetCentralServicesChartTemplate retrieves the StackRox Central Services Helm chart template.
func GetCentralServicesChartTemplate() (*helmtpl.ChartTemplate, error) {
	return getChartTemplate(CentralServicesChartPrefix)
}

var (
	secretGVK = schema.GroupVersionKind{Version: "v1", Kind: "Secret"}
	// SensorCertObjectRefs are the objects in the sensor bundle that represents tls certs.
	SensorCertObjectRefs = map[k8sobjects.ObjectRef]struct{}{
		{GVK: secretGVK, Name: "sensor-tls", Namespace: namespaces.StackRox}:            {},
		{GVK: secretGVK, Name: "collector-tls", Namespace: namespaces.StackRox}:         {},
		{GVK: secretGVK, Name: "admission-control-tls", Namespace: namespaces.StackRox}: {},
	}
)

// This block enumerates the files in the various charts that have TLS secrets relevant for mTLS.
// A unit test ensures that it is in sync with the contents of the YAML files.
var (
	SensorMTLSFiles = set.NewFrozenStringSet("admission-controller-secret.yaml", "collector-secret.yaml", "sensor-secret.yaml")

	CentralMTLSFiles = set.NewFrozenStringSet("tls-secret.yaml")
	ScannerMTLSFiles = set.NewFrozenStringSet("tls-secret.yaml")
)

// We need to stamp in the version to the Chart.yaml files prior to loading the chart
// or it will fail
func getChartFiles(box packr.Box, prefixes []string, overrides map[string]func() io.ReadCloser) ([]*loader.BufferedFile, error) {
	var chartFiles []*loader.BufferedFile
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
					"Name":    prefix,
				})
				if err != nil {
					return err
				}
			}
			chartFiles = append(chartFiles, &loader.BufferedFile{
				Name: trimmedPath,
				Data: data,
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return chartFiles, nil
}

// GetFilesFromBox returns all files from the box matching the provided prefix.
func GetFilesFromBox(box packr.Box, prefix string) ([]*loader.BufferedFile, error) {
	normPrefix := path.Clean(prefix)
	if normPrefix == "." {
		normPrefix = ""
	} else {
		normPrefix = strings.TrimRight(normPrefix, "/") + "/"
	}

	var files []*loader.BufferedFile
	err := box.WalkPrefix(normPrefix, func(path string, file packd.File) error {
		relativePath := strings.TrimPrefix(path, normPrefix)
		contents, err := ioutil.ReadAll(file)
		if err != nil {
			return errors.Wrapf(err, "reading file %s from packr box", path)
		}
		files = append(files, &loader.BufferedFile{
			Name: relativePath,
			Data: contents,
		})
		return nil
	})

	if err != nil {
		return nil, err
	}
	return files, nil
}

// GetSensorChartTemplate loads the Sensor helmtpl meta-template from the given Box.
func GetSensorChartTemplate(box packr.Box) (*helmtpl.ChartTemplate, error) {
	chartTplFiles, err := GetFilesFromBox(box, sensorChartPrefix)
	if err != nil {
		return nil, errors.Wrap(err, "fetching sensor chart files from box")
	}

	return helmtpl.Load(chartTplFiles)
}

func getSensorChart(box packr.Box, values map[string]interface{}, certs *sensor.Certs) (*chart.Chart, error) {
	chartTpl, err := GetSensorChartTemplate(box)
	if err != nil {
		return nil, errors.Wrap(err, "loading sensor chart template")
	}

	renderedFiles, err := chartTpl.InstantiateRaw(values)
	if err != nil {
		return nil, errors.Wrap(err, "instantiating sensor chart template")
	}

	for certPath, data := range certs.Files {
		renderedFiles = append(renderedFiles, &loader.BufferedFile{
			Name: certPath,
			Data: data,
		})
	}

	if certOnly, _ := values["CertsOnly"].(bool); !certOnly {
		scriptFiles, err := addScripts(box, values)
		if err != nil {
			return nil, err
		}

		renderedFiles = append(renderedFiles, scriptFiles...)
	}

	return loader.LoadFiles(renderedFiles)
}

func addScripts(box packr.Box, values map[string]interface{}) ([]*loader.BufferedFile, error) {
	if values["ClusterType"] == storage.ClusterType_KUBERNETES_CLUSTER.String() {
		return scripts(box, values, k8sScriptsFileMap)
	} else if values["ClusterType"] == storage.ClusterType_OPENSHIFT_CLUSTER.String() {
		return scripts(box, values, osScriptsFileMap)
	} else {
		return nil, errors.Errorf("unable to create sensor bundle, invalid cluster type for cluster %s",
			values["ClusterName"])
	}
}

func scripts(box packr.Box, values map[string]interface{}, filenameMap map[string]string) ([]*loader.BufferedFile, error) {
	var chartFiles []*loader.BufferedFile
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
		chartFiles = append(chartFiles, &loader.BufferedFile{
			Name: dstFile,
			Data: data,
		})
	}

	return chartFiles, nil
}
