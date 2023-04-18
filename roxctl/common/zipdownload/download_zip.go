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
	pkgZip "github.com/stackrox/rox/pkg/zip"
	"github.com/stackrox/rox/roxctl/common/download"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/logger"
)

const (
	inMemFileSizeThreshold = 1 << 20 // 1MB
)

func extractZipToFolder(contents io.ReaderAt, contentsLength int64, bundleType, outputDir string, log logger.Logger) error {
	reader, err := zip.NewReader(contents, contentsLength)
	if err != nil {
		return errors.Wrap(err, "could not read from zip")
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return errors.Wrapf(err, "Unable to create folder %q", outputDir)
	}

	for _, f := range reader.File {
		if err := extractFile(f, outputDir); err != nil {
			return err
		}
	}

	log.InfofLn("Successfully wrote %s folder %q", bundleType, outputDir)
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

// GetZipOptions specifies a request to download a zip file
type GetZipOptions struct {
	Path, Method, BundleType string
	Body                     []byte
	Timeout                  time.Duration
	ExpandZip                bool
	OutputDir                string
}

func storeZipFile(respBody io.Reader, fileName, outputDir, bundleType string, log logger.Logger) error {
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
	log.InfofLn("Successfully wrote %s zip file to %q", bundleType, filepath.Join(outputDir, fileName))

	return nil
}

// GetZip downloads a zip from the given endpoint.
// bundleType is used for logging.
func GetZip(opts GetZipOptions, env environment.Environment) error {
	client, err := env.HTTPClient(opts.Timeout)
	if err != nil {
		return errors.Wrap(err, "creating HTTP client")
	}
	resp, err := client.DoReqAndVerifyStatusCode(opts.Path, opts.Method, http.StatusOK, bytes.NewBuffer(opts.Body))
	if err != nil {
		return errors.Wrap(err, "could not download zip")
	}
	defer utils.IgnoreError(resp.Body.Close)

	zipFileName, err := download.ParseFilenameFromHeader(resp.Header)
	if err != nil {
		zipFileName = fmt.Sprintf("%s.zip", opts.BundleType)
		env.Logger().WarnfLn("could not obtain output file name from HTTP response: %v.", err)
		env.Logger().InfofLn("Defaulting to filename %q", zipFileName)
	}

	// If containerized, then write a zip file to stdout
	if roxctl.InMainImage() {
		if _, err := io.Copy(environment.CLIEnvironment().InputOutput().Out(), resp.Body); err != nil {
			return errors.Wrap(err, "Error writing out zip file")
		}
		env.Logger().InfofLn("Successfully wrote %s zip file", opts.BundleType)
		return nil
	}

	if !opts.ExpandZip {
		return storeZipFile(resp.Body, zipFileName, opts.OutputDir, opts.BundleType, env.Logger())
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

	return extractZipToFolder(contents, size, opts.BundleType, outputDir, env.Logger())
}

// GetZipFiles downloads a zip from the given endpoint and returns a slice of zip Files.
func GetZipFiles(opts GetZipOptions, env environment.Environment) (map[string]*pkgZip.File, error) {
	client, err := env.HTTPClient(opts.Timeout)
	if err != nil {
		return nil, errors.Wrap(err, "creating HTTP client")
	}
	resp, err := client.DoReqAndVerifyStatusCode(opts.Path, opts.Method, http.StatusOK, bytes.NewBuffer(opts.Body))
	if err != nil {
		return nil, errors.Wrap(err, "could not download zip")
	}
	defer utils.IgnoreError(resp.Body.Close)

	buf := ioutils.NewRWBuf(ioutils.RWBufOptions{MemLimit: inMemFileSizeThreshold})
	defer utils.IgnoreError(buf.Close)

	if _, err := io.Copy(buf, resp.Body); err != nil {
		return nil, errors.Wrap(err, "error downloading Zip file")
	}

	contents, size, err := buf.Contents()
	if err != nil {
		return nil, errors.Wrap(err, "accessing buffer contents")
	}

	zipReader, err := zip.NewReader(contents, size)
	if err != nil {
		return nil, errors.Wrap(err, "create reader from zip contents")
	}
	fileMap := make(map[string]*pkgZip.File, len(zipReader.File))
	for _, f := range zipReader.File {
		bytes, err := readContents(f)
		if err != nil {
			return nil, errors.Wrapf(err, "read from zip file %s", f.Name)
		}
		fileMap[f.Name] = pkgZip.NewFile(f.Name, bytes, pkgZip.Sensitive)
		env.Logger().InfofLn("%s extracted", f.Name)
	}
	return fileMap, nil
}

func readContents(file *zip.File) ([]byte, error) {
	rd, err := file.Open()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open zipped file %q", file.Name)
	}
	defer utils.IgnoreError(rd.Close)
	bytes, err := io.ReadAll(rd)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read content from zip file %q", file.Name)
	}
	return bytes, nil
}
