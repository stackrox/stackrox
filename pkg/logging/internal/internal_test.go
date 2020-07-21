package internal

import (
	"testing"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/uuid"
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
	logging.NewOrGet(uuid.NewV4().String()).Info("NewOrGet()")
}
