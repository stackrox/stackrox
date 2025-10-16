package converter

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
)

func TestNodeVulnConv(t *testing.T) {
	vuln := &storage.EmbeddedVulnerability{}
	require.NoError(t, testutils.FullInit(vuln, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	vuln.Set_FixedBy("a")

	nodeVuln := &storage.NodeVulnerability{}
	require.NoError(t, testutils.FullInit(nodeVuln, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	nodeVuln.Set_FixedBy("a")
	// EmbeddedVulns do not have a reference field.
	nodeVuln.GetCveBaseInfo().SetReferences(nil)
	nodeVuln.GetCveBaseInfo().SetCvssMetrics(nil)
	nodeVuln.GetCveBaseInfo().ClearEpss()
	embedvuln := EmbeddedVulnerabilityToNodeVulnerability(vuln)
	protoassert.Equal(t, nodeVuln, embedvuln)
}
