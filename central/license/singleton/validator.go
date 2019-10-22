package singleton

import (
	"github.com/stackrox/rox/pkg/license/validator"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once                    sync.Once
	signingKeyRegisterFuncs []func(validator.Validator) error
	v                       validator.Validator
)

func validatorSingleton() validator.Validator {
	once.Do(func() {
		v = validator.New()
		for _, f := range signingKeyRegisterFuncs {
			utils.Must(f(v))
		}
	})

	return v
}

func registerSigningKeyRegisterFuncs(funcs ...func(validator.Validator) error) {
	signingKeyRegisterFuncs = append(signingKeyRegisterFuncs, funcs...)
}
