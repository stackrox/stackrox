package zip

import (
	"archive/zip"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testZipReaderFile = "readerTestFile"
)

var testZipStructure = []string{
	"f1",
	"p1/f2",
	"p1/p2/f3",
	"f4",
}

func createZipFile(t *testing.T, tempFile *os.File) {
	zipWriter := zip.NewWriter(tempFile)
	defer func() { _ = zipWriter.Close() }()
	for _, fPath := range testZipStructure {
		f, err := zipWriter.Create(fPath)
		require.NoError(t, err)
		_, err = f.Write([]byte("content: " + fPath))
		require.NoError(t, err)
	}
	require.NoError(t, zipWriter.Close())
}

func TestReaderContainsFile(t *testing.T) {
	tempFile, err := os.CreateTemp("", testZipReaderFile)
	require.NoError(t, err)
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()
	createZipFile(t, tempFile)
	reader, err := NewReader(tempFile.Name())
	require.NoError(t, err)

	testCases := []struct {
		description string
		fileName    string
		exists      bool
	}{
		{
			description: "File does not exist with directory",
			fileName:    "p1/x1",
			exists:      false,
		},
		{
			description: "File does not exist w/o directory",
			fileName:    "x1",
			exists:      false,
		},
		{
			description: "File exists with directory",
			fileName:    "p1/f2",
			exists:      true,
		},
		{
			description: "File exists w/o directory",
			fileName:    "f1",
			exists:      true,
		},
		{
			description: "Name is a directory",
			fileName:    "p1",
			exists:      false,
		},
	}

	for _, testCase := range testCases {
		c := testCase
		t.Run(c.description, func(t *testing.T) {
			assert.Equal(t, reader.ContainsFile(c.fileName), c.exists)
		})
	}
	require.NoError(t, tempFile.Close())
	require.NoError(t, os.Remove(tempFile.Name()))
}

func TestReaderReadFrom(t *testing.T) {
	tempFile, err := os.CreateTemp("", testZipReaderFile)
	require.NoError(t, err)
	defer func() {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
	}()
	createZipFile(t, tempFile)
	reader, err := NewReader(tempFile.Name())
	require.NoError(t, err)
	for _, fPath := range testZipStructure {
		content, err := reader.ReadFrom(fPath)
		assert.NoError(t, err)
		assert.Equal(t, string(content), "content: "+fPath)
	}
	require.NoError(t, tempFile.Close())
	require.NoError(t, os.Remove(tempFile.Name()))
}
