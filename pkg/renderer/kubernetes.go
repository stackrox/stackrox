package renderer

import (
	"encoding/base64"
	"fmt"
	"io"
	"path/filepath"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	kubernetesPkg "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/zip"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

const (
	defaultMonitoringPort = 443
)

var (
	assetFileNameMap = NewFileNameMap("docker-auth.sh")

	caSetupScriptsFileNameMap = FileNameMap{
		"common/ca-setup.sh":  "central/scripts/ca-setup.sh",
		"common/delete-ca.sh": "central/scripts/delete-ca.sh",
	}
)

// mode is the mode we want the renderer to function in.
//go:generate stringer -type=mode
type mode int

const (
	// renderAll renders all objects (central/scanner/monitoring).
	renderAll mode = iota
	// scannerOnly renders only the scanner.
	scannerOnly
	// centralTLSOnly renders only the central tls secret.
	centralTLSOnly
	// scannerTLSOnly renders only the scanner tls secret
	scannerTLSOnly
)

type chartPrefixPair struct {
	chart  *chart.Chart
	prefix string
}

func getCentralChart(centralOverrides map[string]func() io.ReadCloser) chartPrefixPair {
	return chartPrefixPair{image.GetCentralChart(centralOverrides), "central"}

}

func getScannerChart() chartPrefixPair {
	return chartPrefixPair{image.GetScannerChart(), "scanner"}
}

func filterChartToFiles(ch *chart.Chart, files set.FrozenStringSet) error {
	var relevantTemplates []*chart.Template
	for _, template := range ch.Templates {
		if files.Contains(filepath.Base(template.Name)) {
			relevantTemplates = append(relevantTemplates, template)
		}
	}
	if len(relevantTemplates) != files.Cardinality() {
		return utils.Should(errors.Errorf(
			"did not find all expected mTLS files in %q chart (found %+v, expected %+v)",
			ch.GetMetadata().GetName(), relevantTemplates, files.AsSlice(),
		))
	}
	ch.Templates = relevantTemplates
	return nil
}

func getChartsToProcess(c Config, mode mode, centralOverrides map[string]func() io.ReadCloser) ([]chartPrefixPair, error) {
	switch mode {
	case scannerOnly:
		return []chartPrefixPair{getScannerChart()}, nil
	case centralTLSOnly:
		centralChart := getCentralChart(centralOverrides)
		if err := filterChartToFiles(centralChart.chart, image.CentralMTLSFiles); err != nil {
			return nil, err
		}
		return []chartPrefixPair{centralChart}, nil
	case scannerTLSOnly:
		scannerChart := getScannerChart()
		if err := filterChartToFiles(scannerChart.chart, image.ScannerMTLSFiles); err != nil {
			return nil, err
		}
		return []chartPrefixPair{scannerChart}, nil
	}

	chartsToProcess := []chartPrefixPair{getCentralChart(centralOverrides), getScannerChart()}
	if c.K8sConfig.Monitoring.Type.OnPrem() {
		chartsToProcess = append(chartsToProcess,
			chartPrefixPair{image.GetMonitoringChart(), "monitoring"},
		)
	}
	return chartsToProcess, nil
}

func renderKubectl(c Config, mode mode, centralOverrides map[string]func() io.ReadCloser) ([]*zip.File, error) {
	var renderedFiles []*zip.File
	chartsToProcess, err := getChartsToProcess(c, mode, centralOverrides)
	if err != nil {
		return nil, err
	}
	for _, chartPrefixPair := range chartsToProcess {
		chartRenderedFiles, err := renderHelmFiles(c, mode, chartPrefixPair.chart, chartPrefixPair.prefix)
		if err != nil {
			return nil, errors.Wrapf(err, "error rendering %s files", chartPrefixPair.prefix)
		}
		renderedFiles = append(renderedFiles, chartRenderedFiles...)
	}

	return renderedFiles, nil
}

func postProcessConfig(c *Config, mode mode) error {
	// Make all items in SecretsByteMap base64 encoded
	c.SecretsBase64Map = make(map[string]string)
	for k, v := range c.SecretsByteMap {
		c.SecretsBase64Map[k] = base64.StdEncoding.EncodeToString(v)
	}
	if mode == centralTLSOnly || mode == scannerTLSOnly {
		return nil
	}
	if c.ClusterType == storage.ClusterType_KUBERNETES_CLUSTER {
		c.K8sConfig.Command = "kubectl"
	} else {
		c.K8sConfig.Command = "oc"
	}

	var err error
	if mode == renderAll {
		c.K8sConfig.Registry, err = kubernetesPkg.GetResolvedRegistry(c.K8sConfig.MainImage)
		if err != nil {
			return err
		}
	}

	c.K8sConfig.ScannerRegistry, err = kubernetesPkg.GetResolvedRegistry(c.K8sConfig.ScannerImage)
	if err != nil {
		return err
	}
	if c.K8sConfig.Registry != c.K8sConfig.ScannerRegistry {
		c.K8sConfig.ScannerSecretName = "stackrox-scanner"
	} else {
		c.K8sConfig.ScannerSecretName = "stackrox"
	}

	if mode == renderAll {
		if err := injectImageTags(c); err != nil {
			return err
		}
		monitoringImage, err := generateMonitoringImage(c.K8sConfig.MainImage, c.K8sConfig.MonitoringImage)
		if err != nil {
			return errors.Wrap(err, "error parsing monitoring image: ")
		}
		c.K8sConfig.Monitoring.Image = monitoringImage
		c.K8sConfig.Monitoring.Endpoint = netutil.WithDefaultPort(c.K8sConfig.Monitoring.Endpoint, defaultMonitoringPort)
	}

	return nil
}

// Render renders a bunch of zip files based on the given config.
func Render(c Config) ([]*zip.File, error) {
	return render(c, renderAll, nil)
}

// RenderWithOverrides renders a bunch of zip files based on the given config, allowing to selectively override some of
// the bundled files.
func RenderWithOverrides(c Config, centralOverrides map[string]func() io.ReadCloser) ([]*zip.File, error) {
	return render(c, renderAll, centralOverrides)
}

// RenderScannerOnly renders the zip files for the scanner based on the given config.
func RenderScannerOnly(c Config) ([]*zip.File, error) {
	return render(c, scannerOnly, nil)
}

func renderAndExtractSingleFileContents(c Config, mode mode) ([]byte, error) {
	files, err := render(c, mode, nil)
	if err != nil {
		return nil, err
	}

	if len(files) != 1 {
		return nil, utils.Should(errors.Errorf("got unexpected number of files when rendering in mode %s: %d", mode, len(files)))
	}
	return files[0].Content, nil
}

// RenderCentralTLSSecretOnly renders just the file that contains the central-tls secret.
func RenderCentralTLSSecretOnly(c Config) ([]byte, error) {
	return renderAndExtractSingleFileContents(c, centralTLSOnly)
}

// RenderScannerTLSSecretOnly renders just the file that contains the scanner-tls secret.
func RenderScannerTLSSecretOnly(c Config) ([]byte, error) {
	return renderAndExtractSingleFileContents(c, scannerTLSOnly)
}

func render(c Config, mode mode, centralOverrides map[string]func() io.ReadCloser) ([]*zip.File, error) {
	err := postProcessConfig(&c, mode)
	if err != nil {
		return nil, err
	}

	var renderedFiles []*zip.File
	if c.K8sConfig.DeploymentFormat == v1.DeploymentFormat_HELM {
		if mode != renderAll {
			return nil, fmt.Errorf("mode %s not supported in helm", mode)
		}
		renderedFiles, err = renderHelm(c, centralOverrides)
	} else {
		renderedFiles, err = renderKubectl(c, mode, centralOverrides)
	}
	if err != nil {
		return nil, err
	}
	if mode == centralTLSOnly || mode == scannerTLSOnly {
		return renderedFiles, nil
	}

	if mode == renderAll {
		caSetupFiles, err := RenderFiles(caSetupScriptsFileNameMap, c)
		if err != nil {
			return nil, err
		}
		renderedFiles = append(renderedFiles, caSetupFiles...)
	}

	assets, err := LoadAssets(assetFileNameMap)
	if err != nil {
		return nil, err
	}
	renderedFiles = append(renderedFiles, assets...)

	readmeFile, err := generateReadmeFile(&c, mode)
	if err != nil {
		return nil, err
	}
	renderedFiles = append(renderedFiles, readmeFile)

	return renderedFiles, nil
}

func getTag(imageStr string) (string, error) {
	imageName, err := imageUtils.GenerateImageFromString(imageStr)
	if err != nil {
		return "", err
	}
	return imageName.GetName().GetTag(), nil
}

func injectImageTags(c *Config) error {
	var err error
	c.K8sConfig.MainImageTag, err = getTag(c.K8sConfig.MainImage)
	if err != nil {
		return err
	}
	return nil
}
