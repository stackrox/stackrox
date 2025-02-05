package detector

import "github.com/stackrox/rox/pkg/env"

var (
	tmpEmptyImageCacheDeployments = env.RegisterBooleanSetting("ROX_TMP_EMPTY_IMAGE_CACHE_ON_DEPL_REPROCESS", false)
	tmpEmptyImageCachePolicies    = env.RegisterBooleanSetting("ROX_TMP_EMPTY_IMAGE_CACHE_ON_POLICY_REPROCESS", true)
)
