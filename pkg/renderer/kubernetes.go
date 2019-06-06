package renderer

import (
	"encoding/base64"
	"fmt"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/images/utils"
	kubernetesPkg "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/zip"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

const (
	defaultMonitoringPort = 443
)

// mode is the mode we want the renderer to function in.
//go:generate stringer -type=mode
type mode int

const (
	// renderAll renders all objects (central/scanner/monitoring).
	renderAll mode = iota
	// scannerOnly renders only the scanner.
	scannerOnly
)

func renderKubectl(c Config, mode mode) ([]*zip.File, error) {
	type chartPrefixPair struct {
		chart  *chart.Chart
		prefix string
	}

	var chartsToProcess []chartPrefixPair
	scannerChart := chartPrefixPair{image.GetScannerChart(), "scanner"}
	if mode == scannerOnly {
		chartsToProcess = []chartPrefixPair{scannerChart}
	} else {
		chartsToProcess = []chartPrefixPair{
			{image.GetCentralChart(), "central"},
			scannerChart,
		}
		if c.K8sConfig.Monitoring.Type.OnPrem() {
			chartsToProcess = append(chartsToProcess,
				chartPrefixPair{image.GetMonitoringChart(), "monitoring"},
			)
		}
	}

	var renderedFiles []*zip.File
	for _, chart := range chartsToProcess {
		chartRenderedFiles, err := renderHelmFiles(c, chart.chart, chart.prefix)
		if err != nil {
			return nil, errors.Wrapf(err, "error rendering %s files", chart.prefix)
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
			return errors.Wrapf(err, "error parsing monitoring image: ")
		}
		c.K8sConfig.Monitoring.Image = monitoringImage
		c.K8sConfig.Monitoring.Endpoint = netutil.WithDefaultPort(c.K8sConfig.Monitoring.Endpoint, defaultMonitoringPort)
	}

	return nil
}

// Render renders a bunch of zip files based on the given config.
func Render(c Config) ([]*zip.File, error) {
	return render(c, renderAll)
}

// RenderScannerOnly renders the zip files for the scanner based on the given config.
func RenderScannerOnly(c Config) ([]*zip.File, error) {
	return render(c, scannerOnly)
}

func render(c Config, mode mode) ([]*zip.File, error) {
	err := postProcessConfig(&c, mode)
	if err != nil {
		return nil, err
	}

	var renderedFiles []*zip.File
	if c.K8sConfig.DeploymentFormat == v1.DeploymentFormat_HELM {
		if mode != renderAll {
			return nil, fmt.Errorf("mode %s not supported in helm", mode)
		}
		renderedFiles, err = renderHelm(c)
	} else {
		renderedFiles, err = renderKubectl(c, mode)
	}
	if err != nil {
		return nil, err
	}
	renderedFiles = append(renderedFiles, dockerAuthFile)

	readmeFile, err := generateReadmeFile(&c, mode)
	if err != nil {
		return nil, err
	}
	renderedFiles = append(renderedFiles, readmeFile)

	return renderedFiles, nil
}

func getTag(imageStr string) (string, error) {
	imageName, err := utils.GenerateImageFromString(imageStr)
	if err != nil {
		return "", err
	}
	return imageName.GetName().GetTag(), nil
}

func injectImageTags(c *Config) error {
	var err error
	c.K8sConfig.ScannerImageTag, err = getTag(c.K8sConfig.ScannerImage)
	if err != nil {
		return err
	}
	c.K8sConfig.MainImageTag, err = getTag(c.K8sConfig.MainImage)
	if err != nil {
		return err
	}
	return nil
}
