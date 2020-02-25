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
					DeploymentEnvironments:        []string{"gcp/ultra-current-825", "azure/66c57ff5-f49f-4510-ae04-e26d3ad2ee63", "aws/051999192406"},
				}
			},
		},
	)
}
