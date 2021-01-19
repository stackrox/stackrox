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
			publickeys.Demos,
			func() validator.SigningKeyRestrictions {
				return validator.SigningKeyRestrictions{
					EarliestNotValidBefore:        buildinfo.BuildTimestamp(),
					LatestNotValidAfter:           buildinfo.BuildTimestamp().Add(90 * 24 * time.Hour),
					MaxDuration:                   30 * 24 * time.Hour,
					AllowOffline:                  true,
					MaxNodeLimit:                  50,
					AllowNoBuildFlavorRestriction: true,
					DeploymentEnvironments:        []string{"gcp/ultra-current-825", "azure/3fe60802-349e-47c6-ba86-4d3bba2b5650", "aws/051999192406"},
				}
			},
		},
	)
}
