package renderer

import (
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	google_protobuf "github.com/golang/protobuf/ptypes/any"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/zip"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/renderutil"
)

func executeChartFiles(prefix string, c Config, files ...*google_protobuf.Any) ([]*zip.File, error) {
	zipFiles := make([]*zip.File, 0, len(files))
	for _, f := range files {
		file, ok, err := executeChartFile(prefix, f.GetTypeUrl(), string(f.GetValue()), c)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		zipFiles = append(zipFiles, file)
	}
	return zipFiles, nil
}

func executeChartFile(prefix string, filename string, templateStr string, c Config) (*zip.File, bool, error) {
	data, err := executeRawTemplate(templateStr, &c)
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

// Helm charts consist of Chart.yaml, values.yaml and templates
// We need to
func (k *kubernetes) renderHelmFiles(c Config, ch *chart.Chart, prefix string) ([]*zip.File, error) {
	ch.Metadata = &chart.Metadata{
		Name: prefix,
	}
	valuesData, err := executeRawTemplate(ch.Values.Raw, &c)
	if err != nil {
		return nil, err
	}
	ch.Values.Raw = string(valuesData)
	m, err := renderutil.Render(ch, &chart.Config{Raw: ch.Values.Raw}, renderutil.Options{})
	if err != nil {
		return nil, err
	}

	var renderedFiles []*zip.File
	// For kubectl files, we don't want to have the templates path so we trim it out
	for k, v := range m {
		if file, ok := getChartFile(prefix, filepath.Base(k), []byte(v)); ok {
			renderedFiles = append(renderedFiles, file)
		}
	}
	// execute the extra files (scripts, README, etc)
	files, err := executeChartFiles(prefix, c, ch.Files...)
	if err != nil {
		return nil, err
	}
	return append(renderedFiles, files...), nil
}

func chartToFiles(prefix string, ch *chart.Chart, c Config) ([]*zip.File, error) {
	renderedFiles, err := executeChartFiles(prefix, c, ch.Files...)
	if err != nil {
		return nil, err
	}

	for _, f := range ch.Templates {
		if file, ok := getChartFile(prefix, f.Name, f.GetData()); ok {
			renderedFiles = append(renderedFiles, file)
		}
	}
	// Execute Values template
	valueFile, ok, err := executeChartFile(prefix, "values.yaml", ch.Values.Raw, c)
	if err != nil {
		return nil, err
	}
	if ok {
		// Values potentially contains passwords
		valueFile.Flags |= zip.Sensitive
		renderedFiles = append(renderedFiles, valueFile)
	}

	// Need the chart file :|
	out, err := yaml.Marshal(ch.GetMetadata())
	if err != nil {
		return nil, err
	}
	chartFile, ok, err := executeChartFile(prefix, "Chart.yaml", string(out), c)
	if err != nil {
		return nil, err
	}
	if ok {
		renderedFiles = append(renderedFiles, chartFile)
	}
	return renderedFiles, nil
}

func (k *kubernetes) renderHelm(c Config) ([]*zip.File, error) {
	renderedFiles, err := chartToFiles("central", image.GetCentralChart(), c)
	if err != nil {
		return nil, err
	}
	clairifyFiles, err := chartToFiles("clairify", image.GetClairifyChart(), c)
	if err != nil {
		return nil, err
	}
	renderedFiles = append(renderedFiles, clairifyFiles...)
	if c.K8sConfig.Monitoring.Type.OnPrem() {
		monitoringFiles, err := chartToFiles("monitoring", image.GetMonitoringChart(), c)
		if err != nil {
			return nil, err
		}
		renderedFiles = append(renderedFiles, monitoringFiles...)
	}
	return renderedFiles, nil
}
