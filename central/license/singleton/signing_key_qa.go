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
						"gcp/srox-temp-dev-test",
						"azure/3fe60802-349e-47c6-ba86-4d3bba2b5650",
						"aws/051999192406", // setup-automation
						"aws/880732477823", // k@stackrox.com
						"aws/393282794030", // gavin@stackrox.com
					},
				}
			},
		},
	)
}
