package singleton

import (
	"time"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/license/publickeys"
	"github.com/stackrox/rox/pkg/license/validator"
)

func init() {
	registerValidatorRegistrationArgs(
		validatorRegistrationArgs{
			publickeys.QA,
			func() validator.SigningKeyRestrictions {
				return validator.SigningKeyRestrictions{
					EarliestNotValidBefore:        buildinfo.BuildTimestamp().Add(-7 * 24 * time.Hour),
					LatestNotValidAfter:           buildinfo.BuildTimestamp().Add(180 * 24 * time.Hour),
					MaxDuration:                   16 * 24 * time.Hour,
					AllowOffline:                  true,
					AllowNoNodeLimit:              true,
					AllowNoBuildFlavorRestriction: true,
					DeploymentEnvironments: []string{
						"gcp/ultra-current-825",
						"azure/66c57ff5-f49f-4510-ae04-e26d3ad2ee63",
						"aws/051999192406",
						"aws/880732477823", // k@stackrox.com"
						"aws/522993616158", // rhama@stackrox.com
						"aws/598979716678", // akshat@stackrox.com
					},
				}
			},
		},
	)
}
