package upload

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/ioutils"
	"github.com/stackrox/rox/pkg/probeupload"
	"github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/central/db/transfer"
)

const (
	kernelModulesDirPrefix = "kernel-modules/"
)

func analyzePackageFile(pkg *zip.Reader) (map[string]*zip.File, bool) {
	hasUnrecognized := false
	probeFiles := make(map[string]*zip.File)
	for _, f := range pkg.File {
		if strings.HasSuffix(f.Name, "/") {
			continue // ignore all directories
		}
		strippedName := strings.TrimPrefix(f.Name, kernelModulesDirPrefix)
		if len(strippedName) == len(f.Name) {
			continue // ignore everything not in the kernel-modules/ directory
		}
		if !probeupload.IsValidFilePath(strippedName) {
			if path.Base(strippedName) != "LICENSE" {
				hasUnrecognized = true
			}
			continue
		}
		probeFiles[strippedName] = f
	}
	return probeFiles, hasUnrecognized
}

func (cmd *collectorSPUploadCommand) retrieveExistingProbeFiles(probeFilesInPackage map[string]*zip.File) ([]*v1.ProbeUploadManifest_File, error) {
	conn, err := cmd.env.GRPCConnection(cmd.retryTimeout)
	if err != nil {
		return nil, errors.Wrap(err, "failed to establish a gRPC connection to Central")
	}

	probeUploadClient := v1.NewProbeUploadServiceClient(conn)

	req := &v1.GetExistingProbesRequest{}
	for probeFileName := range probeFilesInPackage {
		req.FilesToCheck = append(req.FilesToCheck, probeFileName)
	}

	ctx, cancel := context.WithTimeout(common.Context(), cmd.timeout)
	defer cancel()

	resp, err := probeUploadClient.GetExistingProbes(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query Central for existing probes")
	}
	return resp.GetExistingFiles(), nil
}

func readerFuncForZipEntry(entry *zip.File) func() io.Reader {
	return func() io.Reader {
		rc, err := entry.Open()
		if err != nil {
			return ioutils.ErrorReader(err)
		}
		return rc
	}
}

func buildUploadManifest(probeFilesInPackage map[string]*zip.File, existingFiles []*v1.ProbeUploadManifest_File, overwrite bool) (*v1.ProbeUploadManifest, io.ReadCloser, []*zip.File) {
	var nonOverwrittenFiles []*zip.File

	for _, existingFile := range existingFiles {
		pkgEntry := probeFilesInPackage[existingFile.GetName()]
		if pkgEntry == nil {
			continue
		}
		if existingFile.GetSize_() == int64(pkgEntry.UncompressedSize64) && existingFile.GetCrc32() == pkgEntry.CRC32 {
			delete(probeFilesInPackage, existingFile.GetName())
		} else if !overwrite {
			nonOverwrittenFiles = append(nonOverwrittenFiles, pkgEntry)
			delete(probeFilesInPackage, existingFile.GetName())
		}
	}

	mf := &v1.ProbeUploadManifest{}
	readerFuncs := make([]func() io.Reader, 0, len(probeFilesInPackage))
	for fileName, pkgEntry := range probeFilesInPackage {
		mf.Files = append(mf.Files, &v1.ProbeUploadManifest_File{
			Name:  fileName,
			Size_: int64(pkgEntry.UncompressedSize64),
			Crc32: pkgEntry.CRC32,
		})
		readerFuncs = append(readerFuncs, readerFuncForZipEntry(pkgEntry))
	}

	return mf, ioutils.ChainReadersLazy(readerFuncs...), nonOverwrittenFiles
}

func (cmd *collectorSPUploadCommand) doFileUpload(manifest *v1.ProbeUploadManifest, data io.Reader) error {
	totalSize, err := probeupload.AnalyzeManifest(manifest)
	if err != nil {
		return utils.ShouldErr(errors.Wrap(err, "generated invalid manifest"))
	}

	manifestBytes, err := proto.Marshal(manifest)
	if err != nil {
		return errors.Wrap(err, "failed to marshal manifest")
	}

	uploadData := io.MultiReader(bytes.NewReader(manifestBytes), data)

	manifestLen := len(manifestBytes)
	uploadDataSize := totalSize + int64(manifestLen)

	httpClient, err := cmd.env.HTTPClient(0)
	if err != nil {
		return errors.Wrap(err, "failed to instantiate an HTTP client")
	}

	req, err := httpClient.NewReq(http.MethodPost, "/api/extensions/probeupload", uploadData)
	if err != nil {
		return errors.Wrap(err, "failed to instantiate HTTP request")
	}
	req.ContentLength = uploadDataSize

	urlParams := make(url.Values)
	urlParams.Set("manifestLen", strconv.Itoa(manifestLen))
	req.URL.RawQuery = urlParams.Encode()

	cmd.env.Logger().InfofLn("Uploading %d files from support package ...\n", len(manifest.GetFiles()))
	resp, err := transfer.ViaHTTP(req, httpClient, time.Now(), cmd.timeout)
	if err != nil {
		return errors.Wrap(err, "HTTP transport error while uploading collector support files")
	}
	defer utils.IgnoreError(resp.Body.Close)

	if err := httputil.ResponseToError(resp); err != nil {
		return errors.Wrap(err, "server returned an error response")
	}

	cmd.env.Logger().InfofLn("Successfully uploaded %d files from support package.\n", len(manifest.GetFiles()))
	return nil
}

func (cmd *collectorSPUploadCommand) uploadFilesFromPackage() error {
	zipFile, err := zip.OpenReader(cmd.packageFile)
	if err != nil {
		return errors.Wrap(err, "opening support package file")
	}
	defer utils.IgnoreError(zipFile.Close)

	probeFiles, hasUnrecognized := analyzePackageFile(&zipFile.Reader)

	if hasUnrecognized {
		cmd.env.Logger().WarnfLn("The given support package contains unrecognized files. This may indicate data corruption.")
		cmd.env.Logger().WarnfLn("If you have obtained this support package from an official site, contact StackRox support.")
	}

	if len(probeFiles) == 0 {
		return errox.NotFound.New("the given support package contains no relevant files")
	}

	existingFiles, err := cmd.retrieveExistingProbeFiles(probeFiles)
	if err != nil {
		return err
	}

	manifest, data, nonOverwrittenFiles := buildUploadManifest(probeFiles, existingFiles, cmd.overwrite)
	defer utils.IgnoreError(data.Close)
	if len(manifest.GetFiles()) > 0 {
		if err := cmd.doFileUpload(manifest, data); err != nil {
			return err
		}
	} else {
		cmd.env.Logger().InfofLn("All relevant files from this support package are already present. Nothing to do.")
	}

	if len(nonOverwrittenFiles) > 0 {
		cmd.env.Logger().WarnfLn("there were %d file(s) present in the support package that were already present on the server, yet modified.", len(nonOverwrittenFiles))
		i := 0
		for _, omittedFile := range nonOverwrittenFiles {
			if i >= 2 {
				cmd.env.Logger().WarnfLn(" - and %d other(s)", len(nonOverwrittenFiles)-i)
				break
			}
			i++
			cmd.env.Logger().WarnfLn(" - %s", omittedFile.Name)
		}
		cmd.env.Logger().WarnfLn("Re-run this command with the --overwrite flag to overwrite these files on the server.")
	}

	return nil
}
