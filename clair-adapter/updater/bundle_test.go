package updater

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnpackBundle(t *testing.T) {
	ctx := t.Context()
	_ = ctx

	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "vulnerabilities.zip")

	// Create test data
	alpineData := []byte("alpine vulnerability data")
	nvdData := []byte("nvd vulnerability data")

	// Create a ZIP file with test bundles
	zipFile, err := os.Create(zipPath)
	require.NoError(t, err)
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)

	// Add alpine bundle
	alpineWriter, err := zipWriter.Create("alpine.json.zst")
	require.NoError(t, err)
	_, err = alpineWriter.Write(alpineData)
	require.NoError(t, err)

	// Add nvd bundle
	nvdWriter, err := zipWriter.Create("nvd.json.zst")
	require.NoError(t, err)
	_, err = nvdWriter.Write(nvdData)
	require.NoError(t, err)

	require.NoError(t, zipWriter.Close())
	require.NoError(t, zipFile.Close())

	// Unpack the bundle
	bundles, err := UnpackBundle(zipPath)
	require.NoError(t, err)
	require.Len(t, bundles, 2)

	// Verify bundles
	bundleMap := make(map[string]*BundleData)
	for _, bundle := range bundles {
		bundleMap[bundle.Name] = bundle
	}

	// Check alpine bundle
	alpine, ok := bundleMap["alpine.json"]
	require.True(t, ok, "alpine bundle not found")
	assert.Equal(t, alpineData, alpine.Data)

	// Verify fingerprint is correct SHA256
	expectedFingerprint := sha256.Sum256(alpineData)
	assert.Equal(t, hex.EncodeToString(expectedFingerprint[:]), alpine.Fingerprint)

	// Check nvd bundle
	nvd, ok := bundleMap["nvd.json"]
	require.True(t, ok, "nvd bundle not found")
	assert.Equal(t, nvdData, nvd.Data)

	expectedFingerprint = sha256.Sum256(nvdData)
	assert.Equal(t, hex.EncodeToString(expectedFingerprint[:]), nvd.Fingerprint)
}

func TestUnpackBundleFromReader(t *testing.T) {
	// Create test data
	alpineData := []byte("alpine vulnerability data")
	nvdData := []byte("nvd vulnerability data")

	// Create a ZIP in memory
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	// Add alpine bundle
	alpineWriter, err := zipWriter.Create("alpine.json.zst")
	require.NoError(t, err)
	_, err = alpineWriter.Write(alpineData)
	require.NoError(t, err)

	// Add nvd bundle
	nvdWriter, err := zipWriter.Create("nvd.json.zst")
	require.NoError(t, err)
	_, err = nvdWriter.Write(nvdData)
	require.NoError(t, err)

	require.NoError(t, zipWriter.Close())

	// Unpack from reader
	reader := bytes.NewReader(buf.Bytes())
	bundles, err := UnpackBundleFromReader(reader, int64(buf.Len()))
	require.NoError(t, err)
	require.Len(t, bundles, 2)

	// Verify bundles
	bundleMap := make(map[string]*BundleData)
	for _, bundle := range bundles {
		bundleMap[bundle.Name] = bundle
	}

	assert.Contains(t, bundleMap, "alpine.json")
	assert.Contains(t, bundleMap, "nvd.json")
}

func TestUnpackBundle_EmptyZip(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "empty.zip")

	// Create an empty ZIP
	zipFile, err := os.Create(zipPath)
	require.NoError(t, err)
	zipWriter := zip.NewWriter(zipFile)
	require.NoError(t, zipWriter.Close())
	require.NoError(t, zipFile.Close())

	// Unpack should succeed but return empty slice
	bundles, err := UnpackBundle(zipPath)
	require.NoError(t, err)
	assert.Empty(t, bundles)
}

func TestUnpackBundle_NonExistentFile(t *testing.T) {
	bundles, err := UnpackBundle("/nonexistent/path/vulnerabilities.zip")
	require.Error(t, err)
	assert.Nil(t, bundles)
}

func TestUnpackBundle_IgnoresDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "vulnerabilities.zip")

	// Create a ZIP with a directory
	zipFile, err := os.Create(zipPath)
	require.NoError(t, err)
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)

	// Add a directory entry
	_, err = zipWriter.Create("subdir/")
	require.NoError(t, err)

	// Add a file
	fileWriter, err := zipWriter.Create("alpine.json.zst")
	require.NoError(t, err)
	_, err = fileWriter.Write([]byte("data"))
	require.NoError(t, err)

	require.NoError(t, zipWriter.Close())
	require.NoError(t, zipFile.Close())

	// Unpack should only return the file, not the directory
	bundles, err := UnpackBundle(zipPath)
	require.NoError(t, err)
	require.Len(t, bundles, 1)
	assert.Equal(t, "alpine.json", bundles[0].Name)
}
