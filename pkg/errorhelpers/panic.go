package errorhelpers

import (
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// PanicOnDevelopment will panic if we are in a development build (environment variable will be injected by dev scripts)
// It will not panic in a release version and instead log the error
func PanicOnDevelopment(err error) {
	if env.DevelopmentBuild.Setting() == "true" {
		panic(err)
	}
	log.Error(err)
}
