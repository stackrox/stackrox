package manager

import (
	"testing"

	"github.com/stackrox/rox/central/globaldb/v2backuprestore/formats"
	"github.com/stretchr/testify/assert"
)

func TestSupportedFormatsAreNotEmpty(t *testing.T) {
	assert.NotEmpty(t, formats.RegistrySingleton().GetSupportedFormats())
}
