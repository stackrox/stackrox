package vuln

import (
	"archive/zip"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/stackrox/rox/scanner/updater/jsonblob"
	"github.com/stretchr/testify/require"
)

func TestCIMinimalBundleValid(t *testing.T) {
	_, sourceFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "locate test source to find the checked-in bundle")
	bundlePath := filepath.Join(filepath.Dir(sourceFile), "../../../../scanner/image/scanner/bundles/ci-minimal/vulnerabilities.zip")

	bundle, err := zip.OpenReader(bundlePath)
	require.NoError(t, err, "open checked-in CI-minimal bundle")
	t.Cleanup(func() { require.NoError(t, bundle.Close()) })
	require.NotEmpty(t, bundle.File)

	for _, file := range bundle.File {
		require.True(t, strings.HasSuffix(file.Name, ".json.zst"), "unexpected bundle member %q", file.Name)
		member, err := file.Open()
		require.NoError(t, err)
		reader, err := zstd.NewReader(member)
		require.NoError(t, err, "decompress %s", file.Name)
		operations, checkErr := jsonblob.Iterate(reader)
		operationCount, recordCount := 0, 0
		operations(func(_ *driver.UpdateOperation, records jsonblob.RecordIter) bool {
			operationCount++
			records(func(_ *claircore.Vulnerability, _ *driver.EnrichmentRecord) bool {
				recordCount++
				return true
			})
			return true
		})
		require.NoError(t, checkErr(), "read %s using the production bundle reader", file.Name)
		require.Positive(t, operationCount, "bundle member %s has no update operations", file.Name)
		require.Positive(t, recordCount, "bundle member %s has no records", file.Name)
		reader.Close()
		require.NoError(t, member.Close())
	}
}
