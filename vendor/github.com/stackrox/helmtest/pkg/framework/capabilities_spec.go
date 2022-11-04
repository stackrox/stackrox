package framework

import "helm.sh/helm/v3/pkg/chartutil"

func (c *CapabilitiesSpec) toHelmKubeVersion() chartutil.KubeVersion {
	kubeVersion := chartutil.KubeVersion{}

	ver := c.KubeVersion
	if ver.Major != "" {
		kubeVersion.Major = ver.Major
	}
	if ver.Minor != "" {
		kubeVersion.Minor = ver.Minor
	}
	if ver.Version != "" {
		kubeVersion.Version = ver.Version
	}

	return kubeVersion
}
