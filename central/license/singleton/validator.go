package singleton

import (
	"github.com/stackrox/rox/pkg/license/publickeys"
	"github.com/stackrox/rox/pkg/license/validator"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once                   sync.Once
	validatorRegistrations []validatorRegistrationArgs
	v                      validator.Validator
)

type validatorRegistrationArgs struct {
	keyAndAlgo publickeys.KeyAndAlgo
	// We have a func that returns the restrictions because we don't want the validator restrictions
	// to be computed at program init time -- this affects unit tests because the buildinfo.Timestamp
	// that some of the restrictions use is not stamped in unit tests.
	restrictionsFunc func() validator.SigningKeyRestrictions
}

func validatorSingleton() validator.Validator {
	once.Do(func() {
		v = validator.New()
		for _, args := range validatorRegistrations {
			restrictions := args.restrictionsFunc()
			utils.Must(v.RegisterSigningKey(args.keyAndAlgo.Algo, args.keyAndAlgo.Key, &restrictions))
		}
	})

	return v
}

func registerValidatorRegistrationArgs(args ...validatorRegistrationArgs) {
	validatorRegistrations = append(validatorRegistrations, args...)
}
