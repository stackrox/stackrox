package cvss

import (
	"archive/zip"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/quay/claircore/enricher/cvss"
	"github.com/stackrox/rox/central/scannerdefinitions/file"
	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

const (
	defURL = "https://storage.googleapis.com/scanner-v4-test/nvddata/"
)

func assertOnFileExistence(t *testing.T, path string, shouldExist bool) {
	exists, err := fileutils.Exists(path)
	require.NoError(t, err)
	assert.Equal(t, shouldExist, exists)
}

func TestUpdate(t *testing.T) {
	ctx := context.Background()
	filePath := filepath.Join(t.TempDir(), "cvss.zip")
	u := NewUpdaterWithCvssEnricher(file.New(filePath), &http.Client{Timeout: 30 * time.Second}, defURL, 1*time.Hour)

	// Should fetch first time.
	require.NoError(t, u.doUpdate(ctx))
	assertOnFileExistence(t, filePath, true)
}

func TestUpdateAndParse(t *testing.T) {
	ctx := context.Background()
	filePath := filepath.Join(t.TempDir(), "cvss.zip")
	u := NewUpdaterWithCvssEnricher(file.New(filePath), &http.Client{Timeout: 30 * time.Second}, defURL, 1*time.Hour)

	// Should fetch first time.
	require.NoError(t, u.doUpdate(ctx))

	ccu, err := InitializeClaircoreEnricher()
	if err != nil {
		t.Fatal(err)
	}
	log.Info(filePath)
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Error stating file: %v", err)
	}
	if fileInfo.Size() == 0 {
		t.Fatalf("File is empty: %v", filePath)
	}
	rc, err := unzipSingleJSONFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := rc.Close()
		if err != nil {
			log.Errorf("Error closing the zip file: %v", err)
		}
	}()

	res, err := ccu.ParseEnrichment(ctx, rc)
	if err != nil {
		t.Fatal(err)
	}
	assert.True(t, len(res) > 0, "Expected length of res to be greater than 0")

}

func InitializeClaircoreEnricher() (*cvss.Enricher, error) {
	client := &http.Client{}

	enricher := &cvss.Enricher{}

	cfg := cvss.Config{}

	configUnmarshaler := func(any interface{}) error {
		*any.(*cvss.Config) = cfg
		return nil
	}

	err := enricher.Configure(context.Background(), configUnmarshaler, client)
	if err != nil {
		return nil, err
	}

	return enricher, nil
}

func unzipSingleJSONFile(filePath string) (io.ReadCloser, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, err
	}

	rc, err := r.File[0].Open()
	if err != nil {
		r.Close()
		return nil, err
	}

	return &wrappedReadCloser{Reader: rc, Closer: r}, nil
}

type wrappedReadCloser struct {
	io.Reader
	Closer *zip.ReadCloser
}

func (w *wrappedReadCloser) Close() error {
	err := w.Reader.(io.ReadCloser).Close()
	if err != nil {
		return err
	}
	return w.Closer.Close()
}
