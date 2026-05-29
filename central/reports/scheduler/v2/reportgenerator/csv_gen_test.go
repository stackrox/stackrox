package reportgenerator

import (
	"archive/zip"
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateCSV_ZipEntryHasModifiedTimestamp(t *testing.T) {
	before := time.Now().Add(-time.Minute)
	buf, err := GenerateCSV(nil, "test")
	require.NoError(t, err)

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	require.Len(t, reader.File, 1)
	assert.False(t, reader.File[0].Modified.IsZero(), "ZIP entry Modified timestamp should not be zero")
	assert.True(t, reader.File[0].Modified.After(before), "ZIP entry Modified timestamp should be recent")
}
