package updater

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestUpdaterSetNames will fail if updater sets are added/removed in scanner/updater/export.go
//
// Updater names from vulnerability reports are used by Central for associating CVE's
// with additional metadata (such as fixed date).
//
// If changes were made to the configured updater sets:
//  1. If existing names were modified, Central DB migrations may be needed.
//  2. Verify that the datasource filtering logic in pkg/scanners/scannerv4/convert.go:vulnDataSource()
//     handles the new names correctly (especially for updaters that represent Red Hat data)
func TestUpdaterSetNames(t *testing.T) {
	// evaluatedSets are the updater sets that have been reviewed and appropriate filtering
	// logic update in pkg/scanners/scannerv4/convert.go.
	evaluatedSets := []string{
		"alpine",
		"aws",
		"debian",
		"oracle",
		"osv",
		"photon",
		"rhel-vex",
		"suse",
		"ubuntu",
	}

	assert.ElementsMatch(t, evaluatedSets, ccUpdaterSets,
		"Updater sets in export.go don't match expected sets. "+
			"If you added/removed updater sets in scanner/updater/export.go, "+
			"update this test after evaluating filters / usages.")
}
