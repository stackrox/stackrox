package check

import (
	"testing"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestLoggerModule(t *testing.T) {
	assert.Equal(t, "pkg/logging/check", logging.LoggerForModule().GetModule())
}
