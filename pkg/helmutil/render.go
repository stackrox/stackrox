package helmutil

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/engine"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/renderutil"
	tversion "k8s.io/helm/pkg/version"
)

// Options extends the renderutil.Options struct by allowing to specify the list of available
// Kubernetes versions.
type Options struct {
	renderutil.Options

	APIVersions chartutil.VersionSet
}

// Render renders a chart locally, like renderutil.Render, but its options struct allows specifying
// the list of supported API versions explicitly.
func Render(c *chart.Chart, config *chart.Config, opts Options) (map[string]string, error) {
	if req, err := chartutil.LoadRequirements(c); err == nil {
		if err := renderutil.CheckDependencies(c, req); err != nil {
			return nil, err
		}
	} else if err != chartutil.ErrRequirementsNotFound {
		return nil, errors.Errorf("cannot load requirements: %v", err)
	}

	err := chartutil.ProcessRequirementsEnabled(c, config)
	if err != nil {
		return nil, err
	}
	err = chartutil.ProcessRequirementsImportValues(c)
	if err != nil {
		return nil, err
	}

	// Set up engine.
	renderer := engine.New()

	caps := &chartutil.Capabilities{
		APIVersions:   chartutil.DefaultVersionSet,
		KubeVersion:   chartutil.DefaultKubeVersion,
		TillerVersion: tversion.GetVersionProto(),
	}

	if opts.APIVersions != nil {
		caps.APIVersions = opts.APIVersions
	}

	if opts.KubeVersion != "" {
		kv, verErr := semver.NewVersion(opts.KubeVersion)
		if verErr != nil {
			return nil, errors.Errorf("could not parse a kubernetes version: %v", verErr)
		}
		caps.KubeVersion.Major = fmt.Sprint(kv.Major())
		caps.KubeVersion.Minor = fmt.Sprint(kv.Minor())
		caps.KubeVersion.GitVersion = fmt.Sprintf("v%d.%d.0", kv.Major(), kv.Minor())
	}

	vals, err := chartutil.ToRenderValuesCaps(c, config, opts.ReleaseOptions, caps)
	if err != nil {
		return nil, err
	}

	return renderer.Render(c, vals)
}
