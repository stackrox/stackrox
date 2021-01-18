package image

import (
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
	"github.com/stackrox/rox/pkg/templates"
	"github.com/stackrox/rox/pkg/utils"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	templatePath = "templates"

	// CentralServicesChartPrefix points to the new stackrox-central-services Helm Chart.
	CentralServicesChartPrefix = "helm/stackrox-central/"
	// SecuredClusterServicesChartPrefix points to the new stackrox-secured-cluster-services Helm Chart.
	SecuredClusterServicesChartPrefix = "helm/stackrox-secured-cluster/"
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

func mustGetSensorChart(box packr.Box, values map[string]interface{}, certs *sensor.Certs) *chart.Chart {
	ch, err := getSensorChart(box, values, certs)
	utils.Must(err)
	return ch
}

// GetSensorChart returns the Helm chart for sensor
func GetSensorChart(values map[string]interface{}, certs *sensor.Certs) *chart.Chart {
	return mustGetSensorChart(K8sBox, values, certs)
}

// GetCentralServicesChartTemplate retrieves the StackRox Central Services Helm chart template.
func GetCentralServicesChartTemplate() (*helmtpl.ChartTemplate, error) {
	return getChartTemplate(CentralServicesChartPrefix)
}

// GetSecuredClusterServicesChartTemplate retrieves the StackRox Secured Cluster Services Helm chart template.
func GetSecuredClusterServicesChartTemplate() (*helmtpl.ChartTemplate, error) {
	return getChartTemplate(SecuredClusterServicesChartPrefix)
}

var (
	secretGVK = schema.GroupVersionKind{Version: "v1", Kind: "Secret"}
	// SensorCertObjectRefs are the objects in the sensor bundle that represents tls certs.
	SensorCertObjectRefs = map[k8sobjects.ObjectRef]struct{}{
		{GVK: secretGVK, Name: "sensor-tls", Namespace: namespaces.StackRox}:            {},
		{GVK: secretGVK, Name: "collector-tls", Namespace: namespaces.StackRox}:         {},
		{GVK: secretGVK, Name: "admission-control-tls", Namespace: namespaces.StackRox}: {},
	}
	// AdditionalCASensorSecretRef is the object in the sensor bundle that represents additional ca certs.
	AdditionalCASensorSecretRef = k8sobjects.ObjectRef{
		GVK:       secretGVK,
		Name:      "additional-ca-sensor",
		Namespace: namespaces.StackRox,
	}
)

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
	chartTplFiles, err := GetFilesFromBox(box, SecuredClusterServicesChartPrefix)
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
