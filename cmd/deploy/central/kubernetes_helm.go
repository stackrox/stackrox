package central

import (
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	google_protobuf "github.com/golang/protobuf/ptypes/any"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/zip"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/renderutil"
)

func executeChartFiles(prefix string, c Config, files ...*google_protobuf.Any) ([]*v1.File, error) {
	v1Files := make([]*v1.File, 0, len(files))
	for _, f := range files {
		file, ok, err := executeChartFile(prefix, f.GetTypeUrl(), string(f.GetValue()), c)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		v1Files = append(v1Files, file)
	}
	return v1Files, nil
}

func executeChartFile(prefix string, filename string, templateStr string, c Config) (*v1.File, bool, error) {
	data, err := executeRawTemplate(templateStr, &c)
	if err != nil {
		return nil, false, err
	}
	file, ok := getChartFile(prefix, filename, data)
	return file, ok, nil
}

func getChartFile(prefix, filename string, data []byte) (*v1.File, bool) {
	dataStr := string(data)
	if len(strings.TrimSpace(dataStr)) == 0 {
		return nil, false
	}
	return zip.NewFile(filepath.Join(prefix, filename), data, filepath.Ext(filename) == ".sh"), true
}

// Helm charts consist of Chart.yaml, values.yaml and templates
// We need to
func (k *kubernetes) renderHelmFiles(c Config, path, prefix string) ([]*v1.File, error) {
	ch, err := chartutil.Load(path)
	if err != nil {
		return nil, err
	}
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

	var renderedFiles []*v1.File
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

func chartToFiles(prefix string, ch *chart.Chart, c Config) ([]*v1.File, error) {
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

func (k *kubernetes) renderChart(name, path string, c Config) ([]*v1.File, error) {
	ch, err := chartutil.Load(path)
	if err != nil {
		return nil, err
	}
	return chartToFiles(name, ch, c)
}

func (k *kubernetes) renderHelm(c Config) ([]*v1.File, error) {
	renderedFiles, err := k.renderChart("central", centralChartPath, c)
	if err != nil {
		return nil, err
	}
	clairifyFiles, err := k.renderChart("clairify", clairifyChartPath, c)
	if err != nil {
		return nil, err
	}
	renderedFiles = append(renderedFiles, clairifyFiles...)
	if c.K8sConfig.MonitoringType.OnPrem() {
		monitoringFiles, err := k.renderChart("monitoring", monitoringChartPath, c)
		if err != nil {
			return nil, err
		}
		renderedFiles = append(renderedFiles, monitoringFiles...)
	}
	return renderedFiles, nil
}
