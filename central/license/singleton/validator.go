package singleton

import "github.com/stackrox/rox/pkg/license/validator"

var (
	validatorInstance = validator.New()
)
