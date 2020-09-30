package renderer

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/zip"
)

func renderLegacy(c Config, mode mode, centralOverrides map[string]func() io.ReadCloser) ([]*zip.File, error) {
	var renderedFiles []*zip.File
	var err error
	if c.K8sConfig.DeploymentFormat == v1.DeploymentFormat_HELM {
		if mode != renderAll {
			return nil, fmt.Errorf("mode %s not supported in helm", mode)
		}
		if c.K8sConfig != nil && c.K8sConfig.IstioVersion != "" {
			return nil, errors.New("setting an istio version is not supported when outputting Helm charts")
		}
		renderedFiles, err = renderHelm(c, centralOverrides)
	} else if c.K8sConfig.DeploymentFormat == v1.DeploymentFormat_KUBECTL {
		renderedFiles, err = renderKubectl(c, mode, centralOverrides)
	} else {
		return nil, errors.Errorf("unsupported deployment format %v", c.K8sConfig.DeploymentFormat)
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
		renderedFiles = append(renderedFiles, withPrefix("central", caSetupFiles)...)
	}

	// Add docker-auth.sh script to the relevant scripts directories.
	assets, err := LoadAssets(assetFileNameMap)
	if err != nil {
		return nil, err
	}
	if mode == renderAll {
		renderedFiles = append(renderedFiles, withPrefix("central/scripts", assets)...)
	}
	renderedFiles = append(renderedFiles, withPrefix("scanner/scripts", assets)...)

	readmeFile, err := generateReadmeFile(&c, mode)
	if err != nil {
		return nil, err
	}
	renderedFiles = append(renderedFiles, readmeFile)

	return renderedFiles, nil
}
