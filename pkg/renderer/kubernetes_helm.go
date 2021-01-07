package renderer

import (
	"bytes"
	"io"
	"path"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/image/sensor"
	"github.com/stackrox/rox/pkg/helmutil"
	"github.com/stackrox/rox/pkg/istioutils"
	"github.com/stackrox/rox/pkg/zip"
	"helm.sh/helm/v3/pkg/chart"
)

func executeChartFiles(prefix string, c Config, files ...*chart.File) ([]*zip.File, error) {
	zipFiles := make([]*zip.File, 0, len(files))
	for _, f := range files {
		file, ok, err := executeChartFile(prefix, f.Name, f.Data, c)
		if err != nil {
			return nil, errors.Wrapf(err, "executing template for file %s", f.Name)
		}
		if !ok {
			continue
		}
		zipFiles = append(zipFiles, file)
	}
	return zipFiles, nil
}

func executeChartFile(prefix string, filename string, templateBytes []byte, c Config) (*zip.File, bool, error) {
	data, err := executeRawTemplate(templateBytes, &c)
	if err != nil {
		return nil, false, err
	}
	file, ok := getChartFile(prefix, filename, data)
	return file, ok, nil
}

func getChartFile(prefix, filename string, data []byte) (*zip.File, bool) {
	dataStr := string(data)
	if len(strings.TrimSpace(dataStr)) == 0 {
		return nil, false
	}
	var flags zip.FileFlags
	if filepath.Ext(filename) == ".sh" {
		flags |= zip.Executable
	}
	if strings.HasSuffix(filepath.Base(filename), "-secret.yaml") {
		flags |= zip.Sensitive
	}
	return zip.NewFile(filepath.Join(prefix, filename), data, flags), true
}

func getSensorChartFile(filename string, data []byte) (*zip.File, bool) {
	dataStr := string(data)
	if len(strings.TrimSpace(dataStr)) == 0 {
		return nil, false
	}
	var flags zip.FileFlags
	if filepath.Ext(filename) == ".sh" {
		flags |= zip.Executable
	}
	if strings.HasSuffix(filepath.Base(filename), "-secret.yaml") {
		flags |= zip.Sensitive
	}
	return zip.NewFile(filename, data, flags), true
}

// Helm charts consist of Chart.yaml, values.yaml and templates
func renderHelmFiles(c Config, mode mode, chTpl chartPrefixPair) ([]*zip.File, error) {
	var renderOpts helmutil.Options

	if c.K8sConfig != nil && c.K8sConfig.IstioVersion != "" {
		istioAPIResources, err := istioutils.GetAPIResourcesByVersion(c.K8sConfig.IstioVersion)
		if err != nil {
			return nil, errors.Wrap(err, "obtaining Istio API resources")
		}
		renderOpts.APIVersions = helmutil.VersionSetFromResources(istioAPIResources...)
	}

	ch, err := chTpl.Instantiate(&c)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to instantiate chart %s", chTpl.prefix)
	}
	m, err := helmutil.Render(ch, nil, renderOpts)

	if err != nil {
		return nil, err
	}

	var renderedFiles []*zip.File
	// For kubectl files, we don't want to have the templates path so we trim it out
	for k, v := range m {
		if file, ok := getChartFile(chTpl.prefix, filepath.Base(k), []byte(v)); ok {
			renderedFiles = append(renderedFiles, file)
		}
	}

	if mode == centralTLSOnly || mode == scannerTLSOnly {
		return renderedFiles, nil
	}

	// execute the extra files (scripts, README, etc), but filter out config files (these get rendered into configmaps
	// directly).
	var filteredFiles []*chart.File
	for _, f := range ch.Files {
		if strings.HasPrefix(f.Name, "config/") {
			continue
		}
		filteredFiles = append(filteredFiles, f)
	}

	files, err := executeChartFiles(chTpl.prefix, c, filteredFiles...)
	if err != nil {
		return nil, errors.Wrap(err, "executing chart files")
	}
	return append(renderedFiles, files...), nil
}

func chartToFiles(prefix string, ch *chart.Chart) ([]*zip.File, error) {
	var renderedFiles []*zip.File

	for _, f := range ch.Raw {
		if f.Name == "Chart.yaml" {
			continue
		}

		zf, ok := getChartFile(prefix, f.Name, f.Data)
		if !ok {
			continue
		}

		if f.Name == "values.yaml" {
			// Values potentially contains passwords
			zf.Flags |= zip.Sensitive
		}

		renderedFiles = append(renderedFiles, zf)
	}

	// Need the chart file :|
	out, err := yaml.Marshal(ch.Metadata)
	if err != nil {
		return nil, err
	}

	zf, ok := getChartFile(prefix, "Chart.yaml", out)
	if !ok {
		return nil, errors.New("empty Chart.yaml file")
	}
	renderedFiles = append(renderedFiles, zf)
	return renderedFiles, nil
}

func renderHelm(c Config, centralOverrides map[string]func() io.ReadCloser) ([]*zip.File, error) {
	chartsToProcess, err := getChartsToProcess(c, renderAll, centralOverrides)
	if err != nil {
		return nil, err
	}

	var renderedFiles []*zip.File
	for _, chTpl := range chartsToProcess {
		ch, err := chTpl.Instantiate(&c)
		if err != nil {
			return nil, errors.Wrapf(err, "instantiating chart %s", chTpl.prefix)
		}
		currentRenderedFiles, err := chartToFiles(chTpl.prefix, ch)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to package %s chart", chTpl.prefix)
		}
		renderedFiles = append(renderedFiles, currentRenderedFiles...)
	}
	return renderedFiles, nil
}

// RenderSensorTLSSecretsOnly renders just the TLS secrets from the sensor helm chart, concatenated into one YAML file.
func RenderSensorTLSSecretsOnly(values map[string]interface{}, certs *sensor.Certs) ([]byte, error) {
	metaVals := make(map[string]interface{}, len(values)+1)
	for k, v := range values {
		metaVals[k] = v
	}
	metaVals["CertsOnly"] = true

	ch := image.GetSensorChart(metaVals, certs)

	m, err := helmutil.Render(ch, nil, helmutil.Options{})
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	var firstPrinted bool
	for filePath, fileContents := range m {
		if path.Ext(filePath) != ".yaml" {
			continue
		}

		if len(strings.TrimSpace(fileContents)) == 0 {
			continue
		}
		if firstPrinted {
			_, _ = out.WriteString("---\n")
		}
		_, _ = out.WriteString(fileContents)
		firstPrinted = true
	}
	return out.Bytes(), nil
}

// RenderSensor renders the sensorchart and returns rendered files
func RenderSensor(values map[string]interface{}, certs *sensor.Certs, opts helmutil.Options) ([]*zip.File, error) {
	ch := image.GetSensorChart(values, certs)

	m, err := helmutil.Render(ch, nil, opts)
	if err != nil {
		return nil, err
	}

	var renderedFiles []*zip.File
	// For kubectl files, we don't want to have the templates path so we trim it out
	for k, v := range m {
		if strings.TrimSpace(v) == "" {
			continue
		}
		if file, ok := getSensorChartFile(filepath.Base(k), []byte(v)); ok {
			renderedFiles = append(renderedFiles, file)
		}
	}

	assets, err := LoadAssets(assetFileNameMap)
	if err != nil {
		return nil, err
	}
	renderedFiles = append(renderedFiles, assets...)

	return renderedFiles, nil
}
