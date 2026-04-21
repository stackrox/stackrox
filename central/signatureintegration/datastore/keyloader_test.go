package datastore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// validPublicKeyPEM is a minimal RSA public key in PEM format used across keyloader tests.
const validPublicKeyPEM = "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAryQICCl6NZ5gDKrnSztO\n3Hy8PEUcuyvg/ikC+VcIo2SFFSf18a3IMYldIugqqqZCs4/4uVW3sbdLs/6PfgdX\n7O9D22ZiFWHPYA2k2N744MNiCD1UE+tJyllUhSblK48bn+v1oZHCM0nYQ2NqUkvS\nj+hwUU3RiWl7x3D2s9wSdNt7XUtW05a/FXehsPSiJfKvHJJnGOX0BgTvkLnkAOTd\nOrUZ/wK69Dzu4IvrN4vs9Nes8vbwPa/ddZEzGR0cQMt0JBkhk9kU/qwqUseP1QRJ\n5I1jR4g8aYPL/ke9K35PxZWuDp3U0UPAZ3PjFAh+5T+fc7gzCs9dPzSHloruU+gl\nFQIDAQAB\n-----END PUBLIC KEY-----"

const anotherValidPublicKeyPEM = "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEkbBNiNdz/A5HRMVwmQpOgMqBQFNe\nCrZhzDO5cMWezEzKHwRFoKLUZl5U7k2c0SQHa2qAqFMgASlO6mMhXyZPjw==\n-----END PUBLIC KEY-----"

func writePEMFile(t *testing.T, dir, name, contents string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(contents), 0o600))
}

func TestLoadKeysFromDir_DirectoryDoesNotExist(t *testing.T) {
	keys, err := loadKeysFromDir("/nonexistent/path/that/does/not/exist")
	require.NoError(t, err)
	require.Empty(t, keys)
}

func TestLoadKeysFromDir_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	keys, err := loadKeysFromDir(dir)
	require.NoError(t, err)
	require.Empty(t, keys)
}

func TestLoadKeysFromDir_SingleValidPEMFile(t *testing.T) {
	dir := t.TempDir()
	writePEMFile(t, dir, "key.pub", validPublicKeyPEM)

	keys, err := loadKeysFromDir(dir)
	require.NoError(t, err)
	require.Len(t, keys, 1)
	require.Equal(t, "key.pub", keys[0].GetName())
	require.Equal(t, validPublicKeyPEM, keys[0].GetPublicKeyPemEnc())
}

func TestLoadKeysFromDir_MultipleValidPEMFiles(t *testing.T) {
	dir := t.TempDir()
	writePEMFile(t, dir, "key-a.pub", validPublicKeyPEM)
	writePEMFile(t, dir, "key-b.pub", anotherValidPublicKeyPEM)

	keys, err := loadKeysFromDir(dir)
	require.NoError(t, err)
	require.Len(t, keys, 2)

	nameSet := map[string]struct{}{}
	for _, k := range keys {
		nameSet[k.GetName()] = struct{}{}
	}
	require.Contains(t, nameSet, "key-a.pub")
	require.Contains(t, nameSet, "key-b.pub")
}

func TestLoadKeysFromDir_DuplicatePEMContent(t *testing.T) {
	dir := t.TempDir()
	// Two files with identical PEM — only one should be returned.
	writePEMFile(t, dir, "key-a.pub", validPublicKeyPEM)
	writePEMFile(t, dir, "key-b.pub", validPublicKeyPEM)

	keys, err := loadKeysFromDir(dir)
	require.NoError(t, err)
	require.Len(t, keys, 1)
	require.Equal(t, validPublicKeyPEM, keys[0].GetPublicKeyPemEnc())
}

func TestLoadKeysFromDir_InvalidFileSkipped(t *testing.T) {
	dir := t.TempDir()
	writePEMFile(t, dir, "not-a-key.pub", "this is not a PEM block at all")

	keys, err := loadKeysFromDir(dir)
	require.NoError(t, err)
	require.Empty(t, keys)
}

func TestLoadKeysFromDir_MixedValidAndInvalid(t *testing.T) {
	dir := t.TempDir()
	writePEMFile(t, dir, "good.pub", validPublicKeyPEM)
	writePEMFile(t, dir, "bad.pub", "not a PEM block")

	keys, err := loadKeysFromDir(dir)
	require.NoError(t, err)
	require.Len(t, keys, 1)
	require.Equal(t, "good.pub", keys[0].GetName())
}

func TestLoadKeysFromDir_SubdirectoriesIgnored(t *testing.T) {
	dir := t.TempDir()
	// Create a subdirectory — it should be ignored.
	require.NoError(t, os.Mkdir(filepath.Join(dir, "subdir"), 0o755))
	writePEMFile(t, dir, "key.pub", validPublicKeyPEM)

	keys, err := loadKeysFromDir(dir)
	require.NoError(t, err)
	require.Len(t, keys, 1)
	require.Equal(t, "key.pub", keys[0].GetName())
}
