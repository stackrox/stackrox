package datastore

import (
	"encoding/pem"
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
	// pem.EncodeToMemory always appends a trailing newline.
	require.Equal(t, validPublicKeyPEM+"\n", keys[0].GetPublicKeyPemEnc())
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
	// pem.EncodeToMemory always appends a trailing newline.
	require.Equal(t, validPublicKeyPEM+"\n", keys[0].GetPublicKeyPemEnc())
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

func TestLoadKeysFromDir_TrailingWhitespaceAccepted(t *testing.T) {
	dir := t.TempDir()
	// File ends with double newline — was previously rejected with misleading warning.
	writePEMFile(t, dir, "key.pub", validPublicKeyPEM+"\n\n")

	keys, err := loadKeysFromDir(dir)
	require.NoError(t, err)
	require.Len(t, keys, 1, "trailing whitespace must not cause the key to be rejected")
}

func TestLoadKeysFromDir_DuplicateWithTrailingWhitespaceDeduplicated(t *testing.T) {
	dir := t.TempDir()
	// Two files: one with and one without trailing newlines — same underlying key.
	writePEMFile(t, dir, "key-a.pub", validPublicKeyPEM)
	writePEMFile(t, dir, "key-b.pub", validPublicKeyPEM+"\n\n")

	keys, err := loadKeysFromDir(dir)
	require.NoError(t, err)
	require.Len(t, keys, 1, "whitespace-differing duplicates must be deduplicated")
}

func TestLoadKeysFromDir_GPGStyleHeaderStripped(t *testing.T) {
	dir := t.TempDir()
	// Simulate a file with GPG-style metadata lines before the PEM block.
	gpgStyleContent := "Red Hat Release Key (release-key-3)\nVersion: GnuPG v1\n\n" + validPublicKeyPEM
	writePEMFile(t, dir, "key.pub", gpgStyleContent)

	keys, err := loadKeysFromDir(dir)
	require.NoError(t, err)
	require.Len(t, keys, 1, "GPG-style header before PEM block must be accepted")

	// The stored PEM must be canonical — no leading GPG header.
	require.Equal(t, validPublicKeyPEM+"\n", keys[0].GetPublicKeyPemEnc(),
		"stored PEM must be stripped of GPG-style headers")
}

func TestLoadKeysFromDir_StoredPEMIsCanonical(t *testing.T) {
	dir := t.TempDir()
	writePEMFile(t, dir, "key.pub", validPublicKeyPEM)

	keys, err := loadKeysFromDir(dir)
	require.NoError(t, err)
	require.Len(t, keys, 1)

	// The stored value must equal pem.EncodeToMemory of the decoded block.
	block, _ := pem.Decode([]byte(validPublicKeyPEM))
	require.NotNil(t, block)
	require.Equal(t, string(pem.EncodeToMemory(block)), keys[0].GetPublicKeyPemEnc(),
		"stored PEM must be in canonical pem.EncodeToMemory form")
}
