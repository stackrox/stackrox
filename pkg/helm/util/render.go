package util

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

// Options extends the renderutil.Options struct by allowing to specify the list of available
// Kubernetes versions.
type Options struct {
	ReleaseOptions chartutil.ReleaseOptions
	KubeVersion    string
	HelmVersion    string

	APIVersions chartutil.VersionSet
}

// Render renders a chart locally, like renderutil.Render, but its options struct allows specifying
// the list of supported API versions explicitly.
func Render(c *chart.Chart, values chartutil.Values, opts Options) (map[string]string, error) {
	if err := action.CheckDependencies(c, c.Metadata.Dependencies); err != nil {
		return nil, err
	}

	if err := chartutil.ProcessDependencies(c, values); err != nil {
		return nil, err
	}

	caps := &chartutil.Capabilities{
		APIVersions: chartutil.DefaultVersionSet,
		KubeVersion: chartutil.DefaultCapabilities.KubeVersion,
		HelmVersion: chartutil.DefaultCapabilities.HelmVersion,
	}

	if opts.APIVersions != nil {
		caps.APIVersions = opts.APIVersions
	}

	if opts.KubeVersion != "" {
		kv, verErr := semver.NewVersion(opts.KubeVersion)
		if verErr != nil {
			return nil, errors.Errorf("could not parse Kubernetes version: %v", verErr)
		}
		caps.KubeVersion.Major = fmt.Sprint(kv.Major())
		caps.KubeVersion.Minor = fmt.Sprint(kv.Minor())
		caps.KubeVersion.Version = fmt.Sprintf("v%d.%d.0", kv.Major(), kv.Minor())
	}

	if opts.HelmVersion != "" {
		caps.HelmVersion.Version = opts.HelmVersion
	}

	vals, err := chartutil.ToRenderValues(c, values, opts.ReleaseOptions, caps)
	if err != nil {
		return nil, err
	}

	return engine.Render(c, vals)
}
