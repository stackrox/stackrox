package customresource

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/custom_resource.yaml
var templateFile string

func TestConvertToCR(t *testing.T) {
	policy := fixtures.GetPolicy()
	converted, err := GenerateCustomResource(policy)
	require.NoError(t, err)
	fmt.Println(converted)
	assert.YAMLEq(t, templateFile, converted)
}
