package zipdownload

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/ioutils"
	"github.com/stackrox/rox/pkg/roxctl"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
)

const (
	inMemFileSizeThreshold = 1 << 20 // 1MB
)

func extractZipToFolder(contents io.ReaderAt, contentsLength int64, bundleType, outputDir string) error {
	reader, err := zip.NewReader(contents, contentsLength)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return errors.Wrapf(err, "Unable to create folder %q", outputDir)
	}

	for _, f := range reader.File {
		if err := extractFile(f, outputDir); err != nil {
			return err
		}
	}

	printf("Successfully wrote %s folder %q\n", bundleType, outputDir)
	return nil
}

func extractFile(f *zip.File, outputDir string) error {
	fileReader, err := f.Open()
	if err != nil {
		return errors.Wrapf(err, "Unable to open file %q", f.Name)
	}
	defer utils.IgnoreError(fileReader.Close)

	outputFilePath := filepath.Join(outputDir, f.Name)
	folder := path.Dir(outputFilePath)
	if err := os.MkdirAll(folder, 0755); err != nil {
		return errors.Wrapf(err, "Unable to create folder %q", folder)
	}

	outFile, err := os.OpenFile(outputFilePath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, f.Mode())
	if err != nil {
		return errors.Wrapf(err, "Unable to create output file %q", outputFilePath)
	}
	defer utils.IgnoreError(outFile.Close)

	if _, err := io.Copy(outFile, fileReader); err != nil {
		return errors.Wrapf(err, "Unable to write file %q", f.Name)
	}
	return nil
}

func parseFilenameFromHeader(header http.Header) (string, error) {
	data := header.Get("Content-Disposition")
	if data == "" {
		return data, fmt.Errorf("could not parse filename from header: %+v", header)
	}
	oldLen := len(data)
	data = strings.TrimPrefix(data, "attachment; filename=")
	if len(data) == oldLen {
		return "", fmt.Errorf("cannot parse filename from Content-Disposition header value %q", data)
	}
	return strings.Trim(data, `"`), nil
}

func printf(val string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, val, args...)
}

// GetZipOptions specifies a request to download a zip file
type GetZipOptions struct {
	Path, Method, BundleType string
	Body                     []byte
	Timeout                  time.Duration
	ExpandZip                bool
	OutputDir                string
}

func storeZipFile(respBody io.Reader, fileName, outputDir, bundleType string) error {
	outputFile := fileName
	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return errors.Wrapf(err, "could not create output directory %q", outputDir)
		}
		outputFile = filepath.Join(outputDir, outputFile)
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return errors.Wrapf(err, "Could not create file %q", outputFile)
	}
	if _, err := io.Copy(file, respBody); err != nil {
		_ = file.Close()
		return errors.Wrap(err, "error writing to ZIP file")
	}
	if err := file.Close(); err != nil {
		return errors.Wrap(err, "error writing to ZIP file")
	}
	printf("Successfully wrote %s zip file to %q \n", bundleType, filepath.Join(outputDir, fileName))

	return nil
}

// GetZip downloads a zip from the given endpoint.
// bundleType is used for logging.
func GetZip(opts GetZipOptions) error {
	resp, err := common.DoHTTPRequestAndCheck200(opts.Path, opts.Timeout, opts.Method, bytes.NewBuffer(opts.Body))
	if err != nil {
		return err
	}
	defer utils.IgnoreError(resp.Body.Close)

	zipFileName, err := parseFilenameFromHeader(resp.Header)
	if err != nil {
		zipFileName = fmt.Sprintf("%s.zip", opts.BundleType)
		printf("Warning: could not obtain output file name from HTTP response: %v.", err)
		printf("Defaulting to filename %q", zipFileName)
	}

	// If containerized, then write a zip file to stdout
	if roxctl.InMainImage() {
		if _, err := io.Copy(os.Stdout, resp.Body); err != nil {
			return errors.Wrap(err, "Error writing out zip file")
		}
		printf("Successfully wrote %s zip file\n", opts.BundleType)
		return nil
	}

	if !opts.ExpandZip {
		return storeZipFile(resp.Body, zipFileName, opts.OutputDir, opts.BundleType)
	}

	buf := ioutils.NewRWBuf(ioutils.RWBufOptions{MemLimit: inMemFileSizeThreshold})
	defer utils.IgnoreError(buf.Close)

	if _, err := io.Copy(buf, resp.Body); err != nil {
		return errors.Wrap(err, "error downloading Zip file")
	}

	contents, size, err := buf.Contents()
	if err != nil {
		return errors.Wrap(err, "accessing buffer contents")
	}

	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = strings.TrimSuffix(zipFileName, filepath.Ext(zipFileName))
	}

	return extractZipToFolder(contents, size, opts.BundleType, outputDir)
}
