// +build !release

package singleton

import (
	"time"

	"github.com/stackrox/rox/pkg/license/publickeys"
	"github.com/stackrox/rox/pkg/license/validator"
	"github.com/stackrox/rox/pkg/timeutil"
)

func getDevSigningKeyRestrictions(earliestNotValidBefore, latestNotValidAfter time.Time) validator.SigningKeyRestrictions {
	return validator.SigningKeyRestrictions{
		EarliestNotValidBefore:                  earliestNotValidBefore,
		LatestNotValidAfter:                     latestNotValidAfter,
		MaxDuration:                             30 * 24 * time.Hour,
		AllowOffline:                            true,
		MaxNodeLimit:                            50,
		BuildFlavors:                            []string{"development"},
		AllowNoDeploymentEnvironmentRestriction: true,
	}
}

func init() {
	registerValidatorRegistrationArgs(
		validatorRegistrationArgs{
			publickeys.Dev,
			func() validator.SigningKeyRestrictions {
				return getDevSigningKeyRestrictions(
					timeutil.MustParse(time.RFC3339, "2019-03-01T00:00:00Z"),
					timeutil.MustParse(time.RFC3339, "2020-07-01T00:00:00Z"),
				)
			},
		},
		// OLD VERSION - NO LONGER USED FOR NEW LICENSES
		validatorRegistrationArgs{
			publickeys.DevOld,
			func() validator.SigningKeyRestrictions {
				return getDevSigningKeyRestrictions(
					timeutil.MustParse(time.RFC3339, "2019-12-01T00:00:00Z"),
					timeutil.MustParse(time.RFC3339, "2020-04-01T00:00:00Z"),
				)
			},
		},
	)
}
