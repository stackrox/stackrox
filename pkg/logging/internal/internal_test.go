package internal

import (
	"testing"

	"github.com/stackrox/stackrox/pkg/logging"
	"gotest.tools/assert"
)

func TestCurrentModule(t *testing.T) {
	assert.Equal(t, "pkg/logging/internal", logging.CurrentModule().Name())
}

func TestLoggerForModule(t *testing.T) {
	assert.Equal(t, "pkg/logging/internal", logging.LoggerForModule().Module().Name())
}

func TestLoggerCreationSite(t *testing.T) {
	logging.CurrentModule().Logger().Info("CurrentModule().Logger()")
	logging.LoggerForModule().Info("LoggerForModule()")
}
