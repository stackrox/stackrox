package renderer

import (
	"bytes"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/image/sensor"
	"github.com/stackrox/rox/pkg/helm/charts"
	helmUtil "github.com/stackrox/rox/pkg/helm/util"
	"github.com/stackrox/rox/pkg/zip"
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

	fixEmptyImageTag(&values)

	ch, err := helmImage.GetSensorChart(&values, certs)
	if err != nil {
		return nil, errors.Wrap(err, "pre-rendering sensor chart")
	}

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

// fixEmptyImageTag overrides an empty ImageTag in charts.MetaValues to avoid errors when rendering charts.
// This should only be used when rendering TLS charts, because `ImageTag` will never be needed there. Hence, the
// tag value that makes explicit that this value should never appear in a user chart. More info on the issue
// refer to ROX-16212.
//
// Ideally, this should not be needed, and it only happens due to the logic necessary to build manifest bundles.
// Either the logic to render TLS secrets should be decoupled from the manifest bundle, and only the necessary
// files are rendered. Or we fix the logic in the manifest bundle so that ImageTag is always set, regardless of the
// cluster configuration.
func fixEmptyImageTag(values *charts.MetaValues) {
	if values.ImageTag == "" {
		values.ImageTag = "should-never-see-this"
	}
}

// RenderSensor renders the sensorchart and returns rendered files
func RenderSensor(values *charts.MetaValues, certs *sensor.Certs, opts helmUtil.Options) ([]*zip.File, error) {
	helmImage := image.GetDefaultImage()
	ch, err := helmImage.GetSensorChart(values, certs)
	if err != nil {
		return nil, errors.Wrap(err, "pre-rendering sensor chart")
	}

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
