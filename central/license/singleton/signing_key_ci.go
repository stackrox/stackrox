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
			publickeys.CI,
			func() validator.SigningKeyRestrictions {
				return validator.SigningKeyRestrictions{
					EarliestNotValidBefore:        buildinfo.BuildTimestamp(),
					LatestNotValidAfter:           buildinfo.BuildTimestamp().Add(ciSigningKeyLatestNotValidAfterOffset),
					MaxDuration:                   6 * time.Hour,
					AllowOffline:                  true,
					MaxNodeLimit:                  10,
					AllowNoBuildFlavorRestriction: true,
					DeploymentEnvironments: []string{
						"gcp/stackrox-ci",
						"aws/051999192406",
						"azure/3fe60802-349e-47c6-ba86-4d3bba2b5650",
					},
				}
			},
		},
	)
}
