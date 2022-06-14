package helmconfig

import "github.com/stackrox/rox/pkg/env"

var (
	// HelmConfigFingerprint is the environment variable that indicates the fingerprint of the helm cluster
	// config to be used.
	HelmConfigFingerprint = env.RegisterSetting("ROX_HELM_CLUSTER_CONFIG_FP")
)
