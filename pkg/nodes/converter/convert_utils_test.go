package converter

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeVulnConv(t *testing.T) {
	vuln := &storage.EmbeddedVulnerability{}
	require.NoError(t, testutils.FullInit(vuln, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	vuln.SetFixedBy = &storage.EmbeddedVulnerability_FixedBy{FixedBy: "a"}

	nodeVuln := &storage.NodeVulnerability{}
	require.NoError(t, testutils.FullInit(nodeVuln, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	nodeVuln.SetFixedBy = &storage.NodeVulnerability_FixedBy{FixedBy: "a"}
	// EmbeddedVulns do not have a reference field.
	nodeVuln.CveBaseInfo.References = nil
	assert.Equal(t, nodeVuln, EmbeddedVulnerabilityToNodeVulnerability(vuln))
}
