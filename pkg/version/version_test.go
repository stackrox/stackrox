package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCurrentVersion(t *testing.T) {
	_, err := parseMainVersion(GetMainVersion())
	assert.NoError(t, err)
}
