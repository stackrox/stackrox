package e2etests

import (
	"os"
	"regexp"
	"slices"
	"testing"

	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/test"
	"github.com/stackrox/rox/scanner/datastore/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	// expectedUpdaterNamePatterns contains updater name patterns that are known.
	//
	// These will need to be updated as updater names change or new ones introduced AFTER
	// evaluating impact as described in below tests.
	expectedUpdaterNamePatterns = map[string]*regexp.Regexp{
		// Derived from ClairCore: alpine/updater.go:Name()
		"alpine": regexp.MustCompile(`^alpine-\S+-\S+-updater$`),

		// Derived from ClairCore: aws/updater.go:Name()
		"aws": regexp.MustCompile(`^aws-\S+-updater$`),

		// Derived from ClairCore: debian/updater.go:Name()
		"debian": regexp.MustCompile(`^debian/updater$`),

		// Derived from ClairCore: oracle/updater.go:Name()
		"oracle": regexp.MustCompile(`^oracle-\S+-updater$`),

		// Derived from ClairCore: updater/osv/osv.go:Name()
		"osv": regexp.MustCompile(`^osv/.+$`),

		// Derived from ClairCore: photon/photon.go:Name()
		"photon": regexp.MustCompile(`^photon-updater-\S+$`),

		// Derived from ClairCore: rhel/vex/updater.go:Name()
		"rhel-vex": regexp.MustCompile(`^rhel-vex$`),

		// Derived from: scanner/updater/manual/manual.go:Name()
		"stackrox-manual": regexp.MustCompile(`^stackrox-manual$`),

		// Derived from ClairCore: suse/factory.go:Name()
		"suse": regexp.MustCompile(`^suse-updater-\S+$`),

		// Derived from ClairCore: ubuntu/updater.go:Name()
		"ubuntu": regexp.MustCompile(`^ubuntu/updater/\S+$`),
	}
)

// TestUpdaterNames ensures all updater names found in the Scanner V4 matcher database are known.
//
// Changes to the names may require changes to Central / Central DB.
//
// This test is expected to fail when new updaters are added or changes made to existing updater names,
// when this fails:
//  1. If existing names were modified, Central DB migrations may be needed.
//  2. Verify that the datasource filtering logic in pkg/scanners/scannerv4/convert.go:vulnDataSource()
//     handles the new names correctly (especially for updaters that represent Red Hat data)
//
// Example test run:
//
// ```sh
// kubectl port-forward -n stackrox svc/scanner-v4-db 5432:5432
//
// export PGPASSWORD="<something>"
// go test -run ^TestUpdaterNames$ github.com/stackrox/rox/scanner/e2etests -v
// ```
func TestUpdaterNames(t *testing.T) {
	ctx := test.Logging(t)

	// Get database connection string
	//
	// For DBs that require auth either set PGPASSWORD env var or add password to connection string.
	connString := os.Getenv("SCANNER_E2E_DB_CONN_STRING")
	if connString == "" {
		connString = "host=127.0.0.1 port=5432 user=postgres sslmode=disable"
	}

	pool, err := postgres.Connect(ctx, connString, "test")
	require.NoError(t, err)
	defer pool.Close()

	// DO NOT run migrations (not needed) - doMigration=false
	store, err := postgres.InitPostgresMatcherStore(ctx, pool, false)
	require.NoError(t, err)

	t.Log("Querying updaters")
	ops, err := store.GetLatestUpdateRefs(ctx, driver.VulnerabilityKind)
	require.NoError(t, err)
	require.NotEmptyf(t, ops, "No update operations found, verify vulnerability data has been loaded into Scanner V4 DB")

	t.Logf("Found %d updaters in database", len(ops))

	// Extract unique updater names
	var updaterNames []string
	for updaterName := range ops {
		updaterNames = append(updaterNames, updaterName)
	}
	slices.Sort(updaterNames)

	for _, n := range updaterNames {
		known := false
		for _, re := range expectedUpdaterNamePatterns {
			if re.MatchString(n) {
				known = true
				break
			}
		}

		assert.Truef(t, known, "Unknown updater name: %q", n)
	}
}
