package helmconfig

import "github.com/stackrox/rox/pkg/env"

var (
	// HelmConfigFingerprint is the environment variable that indicates the fingerprint of the helm cluster
	// config to be used.
	HelmConfigFingerprint = env.RegisterSetting("ROX_HELM_CLUSTER_CONFIG_FP")

	// HelmConfigFile allows to override the helm config.yaml secret file.
	HelmConfigFile = env.RegisterSetting("ROX_HELM_CONFIG_FILE_OVERRIDE", env.WithDefault("/run/secrets/stackrox.io/helm-cluster-config/config.yaml"))

	// HelmClusterNameFile allows to override helm effective cluster name file.
	HelmClusterNameFile = env.RegisterSetting("ROX_HELM_CLUSTER_NAME_FILE_OVERRIDE", env.WithDefault("/run/secrets/stackrox.io/helm-effective-cluster-name/cluster-name"))
)
