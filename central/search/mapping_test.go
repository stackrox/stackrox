package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOptionsFields(t *testing.T) {
	assert.Equal(t, len(allOptionsMaps), len(AlertOptionsMap)+len(PolicyOptionsMap)+len(ImageOptionsMap)+len(DeploymentOptionsMap))
}
