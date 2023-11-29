package image

import (
	"embed"
	"io/fs"
	"path"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image/sensor"
	"github.com/stackrox/rox/pkg/helm/charts"
	helmTemplate "github.com/stackrox/rox/pkg/helm/template"
	helmUtil "github.com/stackrox/rox/pkg/helm/util"
	"github.com/stackrox/rox/pkg/k8sutil/k8sobjects"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/templates"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

//go:embed templates/* assets/* templates/helm/stackrox-central/* templates/helm/stackrox-central/templates/* templates/helm/stackrox-secured-cluster/templates/* templates/helm/stackrox-secured-cluster/* templates/helm/shared/* templates/helm/shared/templates/*

// AssetFS holds the helm charts
var AssetFS embed.FS

// ChartPrefix defines a chart's prefix pointing to the chart in the embedded filesystem.
type ChartPrefix string

// Path returns the string representation of the prefix path
func (c ChartPrefix) Path() string {
	return string(c)
}

const (
	templatePath = "templates"

	// CentralServicesChartPrefix points to the new stackrox-central-services Helm Chart.
	CentralServicesChartPrefix ChartPrefix = "templates/helm/stackrox-central"
	// SecuredClusterServicesChartPrefix points to the new stackrox-secured-cluster-services Helm Chart.
	SecuredClusterServicesChartPrefix ChartPrefix = "templates/helm/stackrox-secured-cluster"

	// sharedFilesPrefix points to the path to the files shared between all charts
	sharedFilesPrefix = "templates/helm/shared"
)

// These are the go based files from embedded chart filesystem
var (
	k8sScriptsFileMap = map[string]string{
		"templates/sensor/kubernetes/sensor.sh":        "templates/sensor.sh",
		"templates/sensor/kubernetes/delete-sensor.sh": "templates/delete-sensor.sh",
		"templates/common/ca-setup.sh":                 "templates/ca-setup-sensor.sh",
		"templates/common/delete-ca.sh":                "templates/delete-ca-sensor.sh",
	}

	osScriptsFileMap = map[string]string{
		"templates/sensor/openshift/sensor.sh":        "templates/sensor.sh",
		"templates/sensor/openshift/delete-sensor.sh": "templates/delete-sensor.sh",
		"templates/common/ca-setup.sh":                "templates/ca-setup-sensor.sh",
		"templates/common/delete-ca.sh":               "templates/delete-ca-sensor.sh",
	}
)

// Image holds the filesystem
type Image struct {
	fs fs.FS
}

// NewImage returns a new image instance, if a nil filesystem is given the default FS is used
func NewImage(fs fs.FS) *Image {
	return &Image{fs: fs}
}

var defaultImage = NewImage(AssetFS)

// GetDefaultImage returns an image with it's default embedded filesystem
func GetDefaultImage() *Image {
	return defaultImage
}

// LoadFileContents resolves a given file's contents.
func (i *Image) LoadFileContents(filename string) (string, error) {
	content, err := fs.ReadFile(AssetFS, filename)
	if err != nil {
		return "", errors.Wrapf(err, "could not read file %q", filename)
	}
	return string(content), nil
}

// ReadFileAndTemplate reads and renders the template for the file
func (i *Image) ReadFileAndTemplate(pathToFile string) (*template.Template, error) {
	templatePath := path.Join(templatePath, pathToFile)
	contents, err := i.LoadFileContents(templatePath)
	if err != nil {
		return nil, err
	}
	parse, err := helmTemplate.InitTemplate(templatePath).Parse(contents)
	return parse, errors.Wrapf(err, "could not render template %q with file %q", templatePath, pathToFile)
}

// GetChartTemplate loads the chart based on the given prefix.
func (i *Image) GetChartTemplate(chartPrefixPath ChartPrefix) (*helmTemplate.ChartTemplate, error) {
	chartTplFiles, err := i.getChartFiles(chartPrefixPath)
	if err != nil {
		return nil, errors.Wrapf(err, "fetching %s chart files from embedded filesystem", chartPrefixPath)
	}

	chartTpl, err := helmTemplate.Load(chartTplFiles)
	return chartTpl, errors.Wrapf(err, "loading %s helm chart template", chartPrefixPath)
}

// GetCentralServicesChartTemplate retrieves the StackRox Central Services Helm chart template.
func (i *Image) GetCentralServicesChartTemplate() (*helmTemplate.ChartTemplate, error) {
	return i.GetChartTemplate(CentralServicesChartPrefix)
}

// GetSecuredClusterServicesChartTemplate retrieves the StackRox Secured Cluster Services Helm chart template.
func (i *Image) GetSecuredClusterServicesChartTemplate() (*helmTemplate.ChartTemplate, error) {
	return i.GetChartTemplate(SecuredClusterServicesChartPrefix)
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

	pspGVK = schema.GroupVersionKind{Group: "policy", Version: "v1beta1", Kind: "PodSecurityPolicy"}
	// SensorPSPObjectRefs are the objects in the sensor bundle that represents pod security policies.
	SensorPSPObjectRefs = map[k8sobjects.ObjectRef]struct{}{
		{GVK: pspGVK, Name: "stackrox-sensor"}:            {},
		{GVK: pspGVK, Name: "stackrox-collector"}:         {},
		{GVK: pspGVK, Name: "stackrox-admission-control"}: {},
	}
)

// LoadAndInstantiateChartTemplate loads a Helm chart (meta-)template from an embed.FS, and instantiates
// it, using default chart values.
func (i *Image) LoadAndInstantiateChartTemplate(chartPrefixPath ChartPrefix, metaVals *charts.MetaValues) ([]*loader.BufferedFile, error) {
	chartTpl, err := i.GetChartTemplate(chartPrefixPath)
	if err != nil {
		return nil, err
	}

	// Render template files.
	renderedChartFiles, err := chartTpl.InstantiateRaw(metaVals)
	if err != nil {
		return nil, errors.Wrapf(err, "instantiating %s helmtpl", chartPrefixPath)
	}

	// Apply .helmignore filtering rules, to be on the safe side (but keep .helmignore).
	renderedChartFiles, err = helmUtil.FilterFiles(renderedChartFiles)
	if err != nil {
		return nil, errors.Wrap(err, "filtering instantiated helm chart files")
	}

	return renderedChartFiles, nil
}

// getChartFiles returns all files associated with the given chart, including shared files.
func (i *Image) getChartFiles(prefix ChartPrefix) ([]*loader.BufferedFile, error) {
	chartFiles, err := i.getFiles(prefix.Path())
	if err != nil {
		return nil, err
	}

	sharedFiles, err := i.getFiles(sharedFilesPrefix)
	if err != nil {
		return nil, err
	}

	for _, sharedFile := range sharedFiles {
		for _, chartFile := range chartFiles {
			if sharedFile.Name == chartFile.Name {
				return nil, errors.Errorf("Shared file %q already exists in Chart at %q.", sharedFile.Name, path.Join(prefix.Path(), chartFile.Name))
			}
		}
	}

	return append(chartFiles, sharedFiles...), nil
}

// getFiles returns all files recursively under a given path.
func (i *Image) getFiles(prefixPath string) ([]*loader.BufferedFile, error) {
	prefixPath = strings.TrimSuffix(prefixPath, "/")
	var files []*loader.BufferedFile
	err := fs.WalkDir(i.fs, prefixPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		data, err := fs.ReadFile(i.fs, p)
		if err != nil {
			return errors.Wrapf(err, "could not read file %q", p)
		}

		newPath := strings.TrimPrefix(p, prefixPath+"/")
		files = append(files, &loader.BufferedFile{
			Name: newPath,
			Data: data,
		})
		return nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "errors when reading dir %q", prefixPath)
	}

	return files, nil
}

// LoadChart loads the given Helm chart template and renders it as a Helm chart
func (i *Image) LoadChart(chartPrefix ChartPrefix, metaValues *charts.MetaValues) (*chart.Chart, error) {
	renderedChartFiles, err := i.LoadAndInstantiateChartTemplate(chartPrefix, metaValues)
	if err != nil {
		return nil, errors.Wrapf(err, "loading and instantiating embedded chart %q failed", chartPrefix)
	}

	c, err := loader.LoadFiles(renderedChartFiles)
	if err != nil {
		return nil, errors.Wrapf(err, "loading %q helm chart files failed", chartPrefix)
	}
	return c, nil
}

// GetSensorChart returns the Helm chart for sensor
func (i *Image) GetSensorChart(values *charts.MetaValues, certs *sensor.Certs) (*chart.Chart, error) {
	chartTpl, err := i.GetSecuredClusterServicesChartTemplate()
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

	if !values.CertsOnly {
		scriptFiles, err := i.addScripts(values)
		if err != nil {
			return nil, err
		}

		renderedFiles = append(renderedFiles, scriptFiles...)
	}

	files, err := loader.LoadFiles(renderedFiles)
	return files, errors.Wrap(err, "could not load files")
}

func (i *Image) addScripts(values *charts.MetaValues) ([]*loader.BufferedFile, error) {
	if values.ClusterType == storage.ClusterType_KUBERNETES_CLUSTER.String() {
		return i.scripts(values, k8sScriptsFileMap)
	} else if values.ClusterType == storage.ClusterType_OPENSHIFT_CLUSTER.String() || values.ClusterType == storage.ClusterType_OPENSHIFT4_CLUSTER.String() {
		return i.scripts(values, osScriptsFileMap)
	}
	return nil, errors.Errorf("unable to create sensor bundle, invalid cluster type for cluster %s",
		values.ClusterName)
}

func (i *Image) scripts(values *charts.MetaValues, filenameMap map[string]string) ([]*loader.BufferedFile, error) {
	var chartFiles []*loader.BufferedFile
	for srcFile, dstFile := range filenameMap {
		fileData, err := AssetFS.ReadFile(srcFile)
		if err != nil {
			return nil, errors.Wrapf(err, "could not read file: %q", srcFile)
		}
		t, err := helmTemplate.InitTemplate(srcFile).Parse(string(fileData))
		if err != nil {
			return nil, errors.Wrapf(err, "could not render template: %q", srcFile)
		}
		data, err := templates.ExecuteToBytes(t, values)
		if err != nil {
			return nil, errors.Wrap(err, "could not execute template")
		}
		chartFiles = append(chartFiles, &loader.BufferedFile{
			Name: dstFile,
			Data: data,
		})
	}

	return chartFiles, nil
}
