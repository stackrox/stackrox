package e2etests

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/stackrox/rox/pkg/scannerv4/client"
	"github.com/stackrox/rox/scanner/indexer"
	"github.com/stretchr/testify/require"
)

// Named CVEs alone do not cover the aggregate findings expected by backend QA.
func TestCIMinimalBundleStruts(t *testing.T) {
	indexerAddr := os.Getenv("SCANNER_E2E_INDEXER_ADDRESS")
	if indexerAddr == "" {
		indexerAddr = ":8443"
	}
	matcherAddr := os.Getenv("SCANNER_E2E_MATCHER_ADDRESS")
	if matcherAddr == "" {
		matcherAddr = ":8443"
	}
	ctx := context.Background()
	c, err := client.NewGRPCScanner(ctx, client.WithIndexerAddress(indexerAddr),
		client.WithMatcherAddress(matcherAddr), client.SkipTLSVerification)
	require.NoError(t, err)

	tc := TestCase{TestArgs: TestArgs{
		Image:    "quay.io/rhacs-eng/qa-multi-arch:struts-app",
		Username: "QUAY_RHACS_ENG_RO_USERNAME",
		Password: "QUAY_RHACS_ENG_RO_PASSWORD",
	}}
	ref, err := name.ParseReference(tc.Image)
	require.NoError(t, err)
	digest, err := indexer.GetDigestFromReference(ref, tc.authConfig())
	require.NoError(t, err)
	vr, err := c.IndexAndScanImage(ctx, digest, tc.authConfig(), client.ImageRegistryOpt{})
	require.NoError(t, err)

	var count int
	var strutsCVE bool
	for packageID, ids := range vr.GetPackageVulnerabilities() {
		names := make(map[string]struct{})
		for _, id := range ids.GetValues() {
			v := vr.GetVulnerabilities()[id]
			names[v.GetName()] = struct{}{}
			if vr.GetContents().GetPackages()[packageID].GetName() != "org.apache.struts:struts2-core" {
				continue
			}
			strutsCVE = strutsCVE || v.GetName() == "CVE-2017-5638"
			for _, alias := range v.GetAliases() {
				strutsCVE = strutsCVE || (alias.GetSpace() == "CVE" && alias.GetName() == "2017-5638")
			}
		}
		count += len(names)
	}
	require.GreaterOrEqual(t, count, 138)
	require.True(t, strutsCVE, "CVE-2017-5638 must match struts2-core, not just exist as enrichment")
}
