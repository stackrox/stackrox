package renderer

import (
	"bytes"
	"path"
	"path/filepath"
	"strings"

	"github.com/stackrox/stackrox/image"
	"github.com/stackrox/stackrox/image/sensor"
	"github.com/stackrox/stackrox/pkg/helm/charts"
	helmUtil "github.com/stackrox/stackrox/pkg/helm/util"
	"github.com/stackrox/stackrox/pkg/zip"
)

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

// RenderSensorTLSSecretsOnly renders just the TLS secrets from the sensor helm chart, concatenated into one YAML file.
func RenderSensorTLSSecretsOnly(values charts.MetaValues, certs *sensor.Certs) ([]byte, error) {
	helmImage := image.GetDefaultImage()
	// Caution: changing `CertsOnly` on `values` shall not affect callers of this function.
	// Currently, we rely on Go to copy the struct as it is passed by value, not by pointer.
	values.CertsOnly = true

	ch := helmImage.GetSensorChart(&values, certs)

	m, err := helmUtil.Render(ch, nil, helmUtil.Options{})
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
func RenderSensor(values *charts.MetaValues, certs *sensor.Certs, opts helmUtil.Options) ([]*zip.File, error) {
	helmImage := image.GetDefaultImage()
	ch := helmImage.GetSensorChart(values, certs)

	m, err := helmUtil.Render(ch, nil, opts)
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
