package singleton

import (
	"time"

	"github.com/stackrox/rox/pkg/license/publickeys"
	"github.com/stackrox/rox/pkg/license/validator"
	"github.com/stackrox/rox/pkg/timeutil"
)

var (
	// IMPORTANT: CHANGE THIS ONCE YOU ROTATE THE PRODUCTION LICENSE KEY.
	activeProdKey = &publickeys.ProdV2
)

func init() {
	registerValidatorRegistrationArgs(
		// IMPORTANT: IF YOU ADD A NEW PROD KEY, ALSO CHANGE `activeProdKey` ABOVE.
		validatorRegistrationArgs{
			publickeys.ProdV2,
			func() validator.SigningKeyRestrictions {

				return validator.SigningKeyRestrictions{
					EarliestNotValidBefore: timeutil.MustParse(time.RFC3339, "2020-04-27T00:00:00Z"),
					LatestNotValidBefore:   timeutil.MustParse(time.RFC3339, "2022-04-30T00:00:00Z"),
					LatestNotValidAfter:    timeutil.MustParse(time.RFC3339, "2025-05-10T00:00:00Z"),
					// Max license duration is 3 years, add 10 days as leeway to cover leap years or general imprecision etc.
					MaxDuration:                             (3*365 + 10) * 24 * time.Hour,
					AllowOffline:                            true,
					AllowNoNodeLimit:                        true,
					AllowNoBuildFlavorRestriction:           true,
					AllowNoDeploymentEnvironmentRestriction: true,
				}
			},
		},
		validatorRegistrationArgs{
			publickeys.ProdV1,
			func() validator.SigningKeyRestrictions {

				return validator.SigningKeyRestrictions{
					EarliestNotValidBefore: timeutil.MustParse(time.RFC3339, "2018-05-01T00:00:00Z"),
					LatestNotValidBefore:   timeutil.MustParse(time.RFC3339, "2020-04-30T00:00:00Z"),
					LatestNotValidAfter:    timeutil.MustParse(time.RFC3339, "2023-04-30T00:00:00Z"),
					// Max license duration is 3 years, add 10 days as leeway to cover leap years or general imprecision etc.
					MaxDuration:                             (3*365 + 10) * 24 * time.Hour,
					AllowOffline:                            true,
					AllowNoNodeLimit:                        true,
					AllowNoBuildFlavorRestriction:           true,
					AllowNoDeploymentEnvironmentRestriction: true,
				}
			},
		},
	)
}
