package errorhelpers

import (
	"fmt"

	"github.com/stackrox/rox/pkg/devbuild"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// PanicOnDevelopment will panic if we are in a development build (environment variable will be injected by dev scripts)
// It will not panic in a release version and instead log the error
func PanicOnDevelopment(err error) error {
	if err == nil {
		return nil
	}

	if devbuild.IsEnabled() {
		panic(err)
	}
	log.Error(err)
	return err
}

// PanicOnDevelopmentf will panic if we are in a development build (environment variable will be injected by dev scripts)
// It will not panic in a release version and instead log the error
func PanicOnDevelopmentf(format string, args ...interface{}) error {
	err := fmt.Errorf(format, args...)
	return PanicOnDevelopment(err)
}
