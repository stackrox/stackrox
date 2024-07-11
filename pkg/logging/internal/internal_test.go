package internal

import (
	"testing"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestCurrentModule(t *testing.T) {
	assert.Equal(t, "pkg/logging/internal", logging.CurrentModule().Name())
}

func TestLoggerForModule(t *testing.T) {
	assert.Equal(t, "pkg/logging/internal", logging.CurrentModule().Logger().Module().Name())
}

func TestLoggerCreationSite(_ *testing.T) {
	logging.CurrentModule().Logger().Info("CurrentModule().Logger()")
	logging.LoggerForModule().Info("LoggerForModule()")
}
