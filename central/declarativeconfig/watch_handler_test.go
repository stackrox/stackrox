package declarativeconfig

import (
	"context"
	"crypto/md5"
	"os"
	"path"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/declarativeconfig/mocks"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestWatchHandler_CompareHashesForChanges(t *testing.T) {
	cases := map[string]struct {
		fileContents        map[string][]byte
		initialCachedFiles  map[string]md5CheckSum
		expectedResult      bool
		expectedCachedFiles map[string]md5CheckSum
	}{
		"empty cache should signal updated files and cache should be populated": {
			fileContents: map[string][]byte{
				"test-file": []byte("test content"),
			},
			initialCachedFiles: map[string]md5CheckSum{},
			expectedResult:     true,
			expectedCachedFiles: map[string]md5CheckSum{
				"test-file": md5.Sum([]byte("test content")),
			},
		},
		"pre-populated cache containing the new file should not signal updated files": {
			fileContents: map[string][]byte{
				"test-file":        []byte("test content"),
				"test-second-file": []byte("second test content"),
			},
			initialCachedFiles: map[string]md5CheckSum{
				"test-file":        md5.Sum([]byte("test content")),
				"test-second-file": md5.Sum([]byte("second test content")),
			},
			expectedCachedFiles: map[string]md5CheckSum{
				"test-file":        md5.Sum([]byte("test content")),
				"test-second-file": md5.Sum([]byte("second test content")),
			},
		},
		"pre-populated cache containing the new file but different contents should signal updated files": {
			fileContents: map[string][]byte{
				"test-file":        []byte("test content"),
				"test-second-file": []byte("second test content"),
			},
			initialCachedFiles: map[string]md5CheckSum{
				"test-file":        md5.Sum([]byte("test content but different")),
				"test-second-file": md5.Sum([]byte("second test content")),
			},
			expectedResult: true,
			expectedCachedFiles: map[string]md5CheckSum{
				"test-file":        md5.Sum([]byte("test content")),
				"test-second-file": md5.Sum([]byte("second test content")),
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			wh := &watchHandler{cachedFileHashes: c.initialCachedFiles}
			res := wh.compareHashesForChanges(c.fileContents)
			assert.Equal(t, c.expectedResult, res)
			assert.Equal(t, c.expectedCachedFiles, wh.cachedFileHashes)
		})
	}
}

func TestWatchHandler_CheckForDeletedFiles(t *testing.T) {
	cases := map[string]struct {
		fileContents        map[string][]byte
		initialCachedFiles  map[string]md5CheckSum
		expectedResult      bool
		expectedCachedFiles map[string]md5CheckSum
	}{
		"cache containing the file contents should not signal update": {
			fileContents: map[string][]byte{
				"test-file": []byte("test content"),
			},
			initialCachedFiles: map[string]md5CheckSum{
				"test-file": md5.Sum([]byte("test content")),
			},
			expectedCachedFiles: map[string]md5CheckSum{
				"test-file": md5.Sum([]byte("test content")),
			},
		},
		"cache containing a deleted file should signal update": {
			fileContents: map[string][]byte{
				"test-file": []byte("test content"),
			},
			initialCachedFiles: map[string]md5CheckSum{
				"test-file":        md5.Sum([]byte("test content")),
				"second-test-file": md5.Sum([]byte("other test content")),
			},
			expectedCachedFiles: map[string]md5CheckSum{
				"test-file": md5.Sum([]byte("test content")),
			},
			expectedResult: true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			wh := &watchHandler{cachedFileHashes: c.initialCachedFiles}
			res := wh.checkForDeletedFiles(c.fileContents)
			assert.Equal(t, c.expectedResult, res)
			assert.Equal(t, c.expectedCachedFiles, wh.cachedFileHashes)
		})
	}
}

func TestWatchHandler_WithEmptyDirectory(t *testing.T) {
	// 0. Create a watch handler with a lower interval.
	updaterMock := mocks.NewMockdeclarativeConfigReconciler(gomock.NewController(t))

	wh := newWatchHandler(updaterMock)
	opts := k8scfgwatch.Options{
		Interval: 10 * time.Millisecond,
	}

	dirToWatch := t.TempDir()
	// 1. Start the watch handler. Specifically check the returned error, as we do not specify the force flag in the
	// 	  options.
	err := k8scfgwatch.WatchConfigMountDir(context.Background(), dirToWatch, k8scfgwatch.DeduplicateWatchErrors(wh), opts)
	require.NoError(t, err)

	// 2. Add a valid YAML file to the directory the handler is watching.
	role := declarativeconfig.Role{
		Name:          "Head Master",
		Description:   "Head Master of Hogwarts",
		AccessScope:   "Hogwarts",
		PermissionSet: "Everything",
	}
	yamlBytes, err := yaml.Marshal(&role)
	require.NoError(t, err)

	// 2.1 Set the expected calls to the updater.
	updaterMock.EXPECT().ReconcileDeclarativeConfigs([][]byte{yamlBytes})

	filePath := path.Join(dirToWatch, "role")
	f, err := os.Create(filePath)
	require.NoError(t, err)
	defer utils.IgnoreError(f.Close)

	_, err = f.Write(yamlBytes)
	require.NoError(t, err)

	// 3. Wait to ensure the watch handler has been triggered.
	time.Sleep(100 * time.Millisecond)

	// 4. Assert on the cached file hashes.
	expectedCache := map[string]md5CheckSum{
		"role": md5.Sum(yamlBytes),
	}
	assert.Equal(t, expectedCache, wh.cachedFileHashes)
}

func TestWatchHandler_WithPrefilledDirectory(t *testing.T) {
	// 0. Create a watch handler with a lower interval.
	updaterMock := mocks.NewMockdeclarativeConfigReconciler(gomock.NewController(t))

	wh := newWatchHandler(updaterMock)
	opts := k8scfgwatch.Options{
		Interval: 10 * time.Millisecond,
	}

	dirToWatch := t.TempDir()

	// 1. Add a valid YAML file to the directory the handler will be watching.
	role := declarativeconfig.Role{
		Name:          "Head Master",
		Description:   "Head Master of Hogwarts",
		AccessScope:   "Hogwarts",
		PermissionSet: "Everything",
	}
	roleBytes, err := yaml.Marshal(&role)
	require.NoError(t, err)

	// 1.1 Set the expected calls to the updater.
	updaterMock.EXPECT().ReconcileDeclarativeConfigs([][]byte{roleBytes})

	rolePath := path.Join(dirToWatch, "role")
	roleF, err := os.Create(rolePath)
	require.NoError(t, err)
	defer utils.IgnoreError(roleF.Close)

	_, err = roleF.Write(roleBytes)
	require.NoError(t, err)

	// 2. Start the watch handler. Specifically check the returned error, as we do not specify the force flag in the
	// 	  options.
	err = k8scfgwatch.WatchConfigMountDir(context.Background(), dirToWatch, k8scfgwatch.DeduplicateWatchErrors(wh), opts)
	require.NoError(t, err)

	// 3. Wait to ensure the watch handler has been triggered.
	time.Sleep(100 * time.Millisecond)

	// 4. Assert on the cached file hashes.
	expectedCache := map[string]md5CheckSum{
		"role": md5.Sum(roleBytes),
	}
	assert.Equal(t, expectedCache, wh.cachedFileHashes)

	// 5.  Add another valid YAML file to the directory the handler will be watching.
	permissionSet := declarativeconfig.PermissionSet{
		Name:        "Everything",
		Description: "One that can do everything",
		Resources:   nil,
	}
	permissionSetBytes, err := yaml.Marshal(&permissionSet)
	require.NoError(t, err)

	// 5.1 Set the expected calls to the updater.
	updaterMock.EXPECT().ReconcileDeclarativeConfigs(gomock.InAnyOrder([][]byte{permissionSetBytes, roleBytes}))

	permissionSetPath := path.Join(dirToWatch, "permission-set")
	permissionSetF, err := os.Create(permissionSetPath)
	require.NoError(t, err)
	defer utils.IgnoreError(permissionSetF.Close)

	_, err = permissionSetF.Write(permissionSetBytes)
	require.NoError(t, err)

	// 6. Wait to ensure the watch handler has been triggered.
	time.Sleep(100 * time.Millisecond)

	// 7. Assert on the cached file hashes.
	expectedCache = map[string]md5CheckSum{
		"role":           md5.Sum(roleBytes),
		"permission-set": md5.Sum(permissionSetBytes),
	}
	assert.Equal(t, expectedCache, wh.cachedFileHashes)
}

func TestWatchHandler_WithRemovedFiles(t *testing.T) {
	// 0. Create a watch handler with a lower interval.
	updaterMock := mocks.NewMockdeclarativeConfigReconciler(gomock.NewController(t))

	wh := newWatchHandler(updaterMock)
	opts := k8scfgwatch.Options{
		Interval: 10 * time.Millisecond,
	}

	dirToWatch := t.TempDir()
	// 1. Start the watch handler. Specifically check the returned error, as we do not specify the force flag in the
	// 	  options.
	err := k8scfgwatch.WatchConfigMountDir(context.Background(), dirToWatch, k8scfgwatch.DeduplicateWatchErrors(wh), opts)
	require.NoError(t, err)

	// 2. Add a valid YAML file to the directory the handler is watching.
	role := declarativeconfig.Role{
		Name:          "Head Master",
		Description:   "Head Master of Hogwarts",
		AccessScope:   "Hogwarts",
		PermissionSet: "Everything",
	}
	yamlBytes, err := yaml.Marshal(&role)
	require.NoError(t, err)

	// 2.1 Set the expected calls to the updater.
	updaterMock.EXPECT().ReconcileDeclarativeConfigs([][]byte{yamlBytes})

	filePath := path.Join(dirToWatch, "role")
	f, err := os.Create(filePath)
	require.NoError(t, err)
	defer utils.IgnoreError(f.Close)

	_, err = f.Write(yamlBytes)
	require.NoError(t, err)

	// 3. Wait to ensure the watch handler has been triggered.
	time.Sleep(100 * time.Millisecond)

	// 4. Assert on the cached file hashes.
	expectedCache := map[string]md5CheckSum{
		"role": md5.Sum(yamlBytes),
	}

	wh.mutex.RLock()
	assert.Equal(t, expectedCache, wh.cachedFileHashes)
	wh.mutex.RUnlock()

	// 5.Set the expected calls to the updater.
	updaterMock.EXPECT().ReconcileDeclarativeConfigs([][]byte{})

	// 6. Remove the previously added YAML file.
	err = os.Remove(filePath)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// 7. Assert on the cached file hashes.
	expectedCache = map[string]md5CheckSum{}
	wh.mutex.RLock()
	assert.Equal(t, expectedCache, wh.cachedFileHashes)
	wh.mutex.RUnlock()
}
