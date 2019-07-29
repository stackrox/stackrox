package restore

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/ioutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/v2backuprestore"
	"github.com/stackrox/rox/roxctl/central/db/transfer"
	"github.com/stackrox/rox/roxctl/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	checkCapsTimeout = 30 * time.Second
)

var (
	// ErrV2RestoreNotSupported is the error returned to indicate that the server does not support the new
	// backup/restore mechanism.
	ErrV2RestoreNotSupported = errors.New("server does not support V2 restore functionality")
)

func buildManifest(file *os.File, supportedCompressionTypes map[v1.DBExportManifest_EncodingType]struct{}) (*v1.DBExportManifest, []func() io.Reader, error) {
	stat, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}
	zipReader, err := zip.NewReader(file, stat.Size())
	if err != nil {
		return nil, nil, err
	}

	mf := &v1.DBExportManifest{}
	var readerFuncs []func() io.Reader

	for _, entry := range zipReader.File {
		if strings.HasSuffix(entry.Name, "/") {
			continue // ignore directories
		}

		manifestFile := &v1.DBExportManifest_File{
			Name:         entry.Name,
			DecodedSize:  int64(entry.UncompressedSize64),
			DecodedCrc32: entry.CRC32,
		}

		compressionType := v1.DBExportManifest_UNKNOWN
		switch entry.Method {
		case zip.Store:
			compressionType = v1.DBExportManifest_UNCOMPREESSED
		case zip.Deflate:
			compressionType = v1.DBExportManifest_DEFLATED
		}

		var readerFunc func() io.Reader
		if _, formatSupported := supportedCompressionTypes[compressionType]; formatSupported {
			manifestFile.Encoding = compressionType
			compressedLen := int64(entry.CompressedSize64)
			manifestFile.EncodedSize = compressedLen
			offset, err := entry.DataOffset()
			if err != nil {
				return nil, nil, errors.Wrapf(err, "could not determine data offset of file %s within ZIP", file.Name())
			}
			readerFunc = func() io.Reader {
				return io.NewSectionReader(file, offset, compressedLen)
			}
		} else {
			manifestFile.Encoding = v1.DBExportManifest_UNCOMPREESSED
			manifestFile.EncodedSize = manifestFile.DecodedSize
			readerFunc = func() io.Reader {
				reader, err := entry.Open()
				if err != nil {
					return ioutils.ErrorReader(err)
				}
				return reader
			}
		}

		mf.Files = append(mf.Files, manifestFile)
		readerFuncs = append(readerFuncs, readerFunc)
	}

	return mf, readerFuncs, nil
}

// tryRestoreV2 attempts to restore the database using the V2 backup/restore API. If the API is not supported by
// central, `ErrV2RestoreNotSupported` is returned. Otherwise, the error indicates whether the restore process was
// successful.
func tryRestoreV2(file *os.File, deadline time.Time) error {
	st, err := file.Stat()
	if err != nil {
		return errors.Wrap(err, "could not stat input file")
	}

	localFileInfo := &v1.DBRestoreRequestHeader_LocalFileInfo{
		Path:      file.Name(),
		BytesSize: st.Size(),
	}

	conn, err := common.GetGRPCConnection()
	if err != nil {
		return errors.Wrap(err, "could not establish gRPC connection to central")
	}

	ctx, cancel := context.WithTimeout(common.Context(), checkCapsTimeout)
	defer cancel()

	dbClient := v1.NewDBServiceClient(conn)
	caps, err := dbClient.GetExportCapabilities(ctx, &v1.Empty{})
	if err != nil {
		if status.Convert(err).Code() == codes.Unimplemented {
			return ErrV2RestoreNotSupported
		}
		return err
	}

	supportedCompressionTypes := make(map[v1.DBExportManifest_EncodingType]struct{}, len(caps.GetSupportedEncodings()))
	for _, ct := range caps.GetSupportedEncodings() {
		supportedCompressionTypes[ct] = struct{}{}
	}

	manifest, readerFuncs, err := buildManifest(file, supportedCompressionTypes)
	if err != nil {
		return errors.Wrap(err, "could not create manifest")
	}

	format, _, err := v2backuprestore.DetermineFormat(manifest, caps.GetFormats())
	if err != nil {
		return err
	}

	header := &v1.DBRestoreRequestHeader{
		FormatName: format.GetFormatName(),
		Manifest:   manifest,
		LocalFile:  localFileInfo,
	}

	headerBytes, err := proto.Marshal(header)
	if err != nil {
		return errors.Wrap(err, "could not marshal restore header")
	}

	headerReader := func() io.Reader { return bytes.NewReader(headerBytes) }

	allReaderFuncs := make([]func() io.Reader, 0, len(readerFuncs)+1)
	allReaderFuncs = append(allReaderFuncs, headerReader)
	allReaderFuncs = append(allReaderFuncs, readerFuncs...)

	bodyReader := ioutils.ChainReadersLazy(allReaderFuncs...)

	req, err := common.NewHTTPRequestWithAuth(http.MethodPost, "/db/v2/restore", bodyReader)
	if err != nil {
		return errors.Wrap(err, "could not create HTTP request")
	}
	queryParams := req.URL.Query()
	queryParams.Set("headerLength", strconv.Itoa(len(headerBytes)))
	req.URL.RawQuery = queryParams.Encode()

	req.ContentLength = v2backuprestore.RestoreBodySize(manifest) + int64(len(headerBytes))

	resp, err := transfer.ViaHTTP(req, common.GetHTTPClient(0), deadline, idleTimeout)
	if err != nil {
		return errors.Wrap(err, "could not make HTTP request")
	}
	defer utils.IgnoreError(resp.Body.Close)

	return httputil.ResponseToError(resp)
}
