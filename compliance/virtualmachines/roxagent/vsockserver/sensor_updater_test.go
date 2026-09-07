package vsockserver

import (
	"os"
	"path/filepath"
	"testing"

	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/virtualmachine/cpemapping"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validMappingJSON = `{"data":{"rhel-8-server-rpms":{"cpes":["cpe:/o:redhat:enterprise_linux:8"]}}}`
const otherValidMappingJSON = `{"data":{"rhel-9-server-rpms":{"cpes":["cpe:/o:redhat:enterprise_linux:9"]}}}`
const invalidMappingJSON = `{not json`

// onChangeCounter is a test double for a MappingProvider's onChange
// callback that records how many times it fired.
type onChangeCounter struct {
	count int
}

func (c *onChangeCounter) fn() { c.count++ }

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
}

func newSensorUpdater(t *testing.T, cachePath, bundledPath string, onChange func()) *SensorUpdater {
	t.Helper()
	u := NewSensorUpdater(cachePath, bundledPath, onChange)
	t.Cleanup(u.WaitPersist)
	return u
}

// waitForCacheContent drains persistAndNotify's background write, then
// checks cachePath, so the assertion cannot race AtomicWriteFile.
func waitForCacheContent(t *testing.T, u *SensorUpdater, cachePath, want string) {
	t.Helper()
	u.WaitPersist()
	b, err := os.ReadFile(cachePath)
	require.NoError(t, err)
	assert.Equal(t, want, string(b), "expected %q to be persisted to %s", want, cachePath)
}

func TestNewSensorUpdater_EmptyStart_NotReady(t *testing.T) {
	dir := t.TempDir()
	counter := &onChangeCounter{}

	u := newSensorUpdater(t, filepath.Join(dir, "cache.json"), "", counter.fn)

	assert.False(t, u.Ready())
	assert.Empty(t, u.Hash())
	assert.Equal(t, 0, counter.count)

	_, err := u.Bytes()
	assert.Error(t, err)
	_, err = u.Path()
	assert.Error(t, err)
}

func TestNewSensorUpdater_BootstrapFromCache(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.json")
	writeFile(t, cachePath, validMappingJSON)
	counter := &onChangeCounter{}

	u := newSensorUpdater(t, cachePath, "", counter.fn)

	require.True(t, u.Ready())
	assert.Equal(t, cpemapping.HashMapping([]byte(validMappingJSON)), u.Hash())
	b, err := u.Bytes()
	require.NoError(t, err)
	assert.Equal(t, validMappingJSON, string(b))
	assert.Equal(t, 1, counter.count, "onChange should fire once for a valid bootstrap")
}

func TestNewSensorUpdater_BootstrapFromBundled_WhenNoCache(t *testing.T) {
	dir := t.TempDir()
	bundledPath := filepath.Join(dir, "bundled.json")
	writeFile(t, bundledPath, validMappingJSON)
	counter := &onChangeCounter{}

	u := newSensorUpdater(t, filepath.Join(dir, "cache.json"), bundledPath, counter.fn)

	require.True(t, u.Ready())
	assert.Equal(t, cpemapping.HashMapping([]byte(validMappingJSON)), u.Hash())
	assert.Equal(t, 1, counter.count)
}

func TestNewSensorUpdater_CacheTakesPrecedenceOverBundled(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.json")
	bundledPath := filepath.Join(dir, "bundled.json")
	writeFile(t, cachePath, validMappingJSON)
	writeFile(t, bundledPath, otherValidMappingJSON)

	u := newSensorUpdater(t, cachePath, bundledPath, func() {})

	b, err := u.Bytes()
	require.NoError(t, err)
	assert.Equal(t, validMappingJSON, string(b), "a valid cache must win over a valid bundled file")
}

func TestNewSensorUpdater_InvalidCacheFallsBackToBundled(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.json")
	bundledPath := filepath.Join(dir, "bundled.json")
	writeFile(t, cachePath, invalidMappingJSON)
	writeFile(t, bundledPath, validMappingJSON)

	u := newSensorUpdater(t, cachePath, bundledPath, func() {})

	require.True(t, u.Ready())
	b, err := u.Bytes()
	require.NoError(t, err)
	assert.Equal(t, validMappingJSON, string(b))
}

func TestNewSensorUpdater_InvalidCacheNoBundled_NotReady(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.json")
	writeFile(t, cachePath, invalidMappingJSON)
	counter := &onChangeCounter{}

	u := newSensorUpdater(t, cachePath, "", counter.fn)

	assert.False(t, u.Ready())
	assert.Equal(t, 0, counter.count)
}

func TestSensorUpdater_Update_OwnsContent(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "cache.json")
	u := newSensorUpdater(t, cachePath, "", func() {})
	content := []byte(validMappingJSON)

	updated, err := u.Update(content)
	require.NoError(t, err)
	require.True(t, updated)
	content[0] ^= 0xff

	b, err := u.Bytes()
	require.NoError(t, err)
	assert.Equal(t, validMappingJSON, string(b), "mutating the caller's buffer must not change the active mapping")
	waitForCacheContent(t, u, cachePath, validMappingJSON)
}

func TestSensorUpdater_Path_WritesCurrentActive(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "cache.json")
	u := newSensorUpdater(t, cachePath, "", func() {})
	_, err := u.Update([]byte(validMappingJSON))
	require.NoError(t, err)
	_, err = u.Update([]byte(otherValidMappingJSON))
	require.NoError(t, err)

	path, err := u.Path()
	require.NoError(t, err)
	onDisk, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, otherValidMappingJSON, string(onDisk),
		"Path must persist the current active mapping, not an in-flight older write")
	waitForCacheContent(t, u, cachePath, otherValidMappingJSON)
}

func TestSensorUpdater_Update_AppliesWhenIdle(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.json")
	counter := &onChangeCounter{}
	u := newSensorUpdater(t, cachePath, "", counter.fn)

	updated, err := u.Update([]byte(validMappingJSON))
	require.NoError(t, err)
	assert.True(t, updated)
	assert.True(t, u.Ready())
	assert.Equal(t, cpemapping.HashMapping([]byte(validMappingJSON)), u.Hash())
	assert.Equal(t, 1, counter.count)
	waitForCacheContent(t, u, cachePath, validMappingJSON)
}

func TestSensorUpdater_Update_NoOpWhenUnchanged(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "cache.json")
	u := newSensorUpdater(t, cachePath, "", func() {})
	_, err := u.Update([]byte(validMappingJSON))
	require.NoError(t, err)
	waitForCacheContent(t, u, cachePath, validMappingJSON)

	updated, err := u.Update([]byte(validMappingJSON))
	require.NoError(t, err)
	assert.False(t, updated, "re-applying identical content must be a no-op")
}

func TestSensorUpdater_Update_RejectsInvalidKeepsLastGood(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "cache.json")
	u := newSensorUpdater(t, cachePath, "", func() {})
	_, err := u.Update([]byte(validMappingJSON))
	require.NoError(t, err)
	wantHash := u.Hash()
	waitForCacheContent(t, u, cachePath, validMappingJSON)

	updated, err := u.Update([]byte(invalidMappingJSON))
	assert.Error(t, err)
	assert.False(t, updated)
	assert.Equal(t, wantHash, u.Hash(), "invalid content must not replace the last-good mapping")
}

func TestSensorUpdater_Update_OversizeRejected(t *testing.T) {
	u := newSensorUpdater(t, filepath.Join(t.TempDir(), "cache.json"), "", func() {})
	oversized := make([]byte, cpemapping.MaxMappingBytes+1)

	updated, err := u.Update(oversized)
	assert.Error(t, err)
	assert.False(t, updated)
	assert.False(t, u.Ready())
}

func TestSensorUpdater_UpdateWhileBusy_DefersUntilIdle(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.json")
	counter := &onChangeCounter{}
	u := newSensorUpdater(t, cachePath, "", counter.fn)
	_, err := u.Update([]byte(validMappingJSON))
	require.NoError(t, err)
	require.Equal(t, 1, counter.count)
	idleHash := u.Hash()
	idleBytes, err := u.Bytes()
	require.NoError(t, err)

	u.MarkScanBusy()

	updated, err := u.Update([]byte(otherValidMappingJSON))
	require.NoError(t, err)
	assert.True(t, updated, "a genuinely new candidate is reported as updated even while deferred")

	// While busy, active content (and therefore Hash/Bytes) must not change:
	// an in-flight scan may still be reading these values.
	assert.Equal(t, idleHash, u.Hash())
	b, err := u.Bytes()
	require.NoError(t, err)
	assert.Equal(t, idleBytes, b)
	assert.Equal(t, 1, counter.count, "onChange must not fire for a deferred apply")

	u.MarkScanIdleAndApplyPending()

	assert.Equal(t, cpemapping.HashMapping([]byte(otherValidMappingJSON)), u.Hash())
	b, err = u.Bytes()
	require.NoError(t, err)
	assert.Equal(t, otherValidMappingJSON, string(b))
	assert.Equal(t, 2, counter.count, "onChange must fire once the pending mapping is promoted")
	waitForCacheContent(t, u, cachePath, otherValidMappingJSON)
}

func TestSensorUpdater_UpdateWhileBusy_SameAsPending_NoOp(t *testing.T) {
	u := newSensorUpdater(t, filepath.Join(t.TempDir(), "cache.json"), "", func() {})
	u.MarkScanBusy()

	updated, err := u.Update([]byte(validMappingJSON))
	require.NoError(t, err)
	assert.True(t, updated)

	updated, err = u.Update([]byte(validMappingJSON))
	require.NoError(t, err)
	assert.False(t, updated, "re-staging identical pending content must be a no-op")
}

// TestSensorUpdater_UpdateWhileBusy_RevertToActive_ClearsStalePending covers
// a revert push landing after a newer one was already staged: Sensor pushes
// are ordered, so the newest push (matching active) must win, not the
// earlier, now-stale pending content.
func TestSensorUpdater_UpdateWhileBusy_RevertToActive_ClearsStalePending(t *testing.T) {
	u := newSensorUpdater(t, filepath.Join(t.TempDir(), "cache.json"), "", func() {})
	_, err := u.Update([]byte(validMappingJSON))
	require.NoError(t, err)
	u.MarkScanBusy()

	updated, err := u.Update([]byte(otherValidMappingJSON))
	require.NoError(t, err)
	require.True(t, updated)

	updated, err = u.Update([]byte(validMappingJSON))
	require.NoError(t, err)
	assert.False(t, updated, "reverting to the already-active content is not itself an update")

	u.MarkScanIdleAndApplyPending()

	assert.Equal(t, cpemapping.HashMapping([]byte(validMappingJSON)), u.Hash(),
		"the newest push (matching active) must win over the stale pending mapping")
}

func TestSensorUpdater_MarkScanIdleAndApplyPending_NoPending_NoOp(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "cache.json")
	counter := &onChangeCounter{}
	u := newSensorUpdater(t, cachePath, "", counter.fn)
	_, err := u.Update([]byte(validMappingJSON))
	require.NoError(t, err)
	require.Equal(t, 1, counter.count)
	waitForCacheContent(t, u, cachePath, validMappingJSON)

	u.MarkScanBusy()
	u.MarkScanIdleAndApplyPending()

	assert.Equal(t, cpemapping.HashMapping([]byte(validMappingJSON)), u.Hash())
	assert.Equal(t, 1, counter.count, "onChange must not fire when there is nothing pending to apply")
}

func TestSensorUpdater_UpdatePath_IsSensor(t *testing.T) {
	u := newSensorUpdater(t, filepath.Join(t.TempDir(), "cache.json"), "", func() {})
	assert.Equal(t, pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_SENSOR, u.UpdatePath())
}
