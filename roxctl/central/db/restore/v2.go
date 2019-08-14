package restore

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/ioutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
)

const (
	checkCapsTimeout = 30 * time.Second
)

var (
	// ErrV2RestoreNotSupported is the error returned to indicate that the server does not support the new
	// backup/restore mechanism.
	ErrV2RestoreNotSupported = errors.New("server does not support V2 restore functionality")
)

// V2Command defines the new db restore command
func V2Command() *cobra.Command {
	var file string
	c := &cobra.Command{
		Use:   "restore <file>",
		Short: "Restore the Central DB from a local file.",
		Long:  "Restore the Central DB from a local file.",
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errors.Errorf("too many positional arguments (%d given)", len(args))
			}
			if len(args) == 1 {
				if file != "" {
					return errors.New("legacy --file flag must not be used in conjunction with a positional argument")
				}
				file = args[0]
			}
			if file == "" {
				if len(args) == 0 {
					return c.Usage()
				}
				return fmt.Errorf("file to restore from must be specified")
			}
			return restore(file, flags.Timeout(c), func(file *os.File, deadline time.Time) error {
				return restoreV2(c, file, deadline)
			})
		},
	}

	c.AddCommand(v2RestoreStatusCmd())
	c.AddCommand(v2RestoreCancelCommand())

	c.Flags().StringVar(&file, "file", "", "file to restore the DB from (deprecated; use positional argument)")
	c.Flags().Bool("interrupt", false, "interrupt ongoing restore process (if any) to allow resuming")

	return c
}

func findManifestFile(fileName string, manifest *v1.DBExportManifest) (*v1.DBExportManifest_File, int, error) {
	for idx, mfFile := range manifest.GetFiles() {
		if mfFile.GetName() == fileName {
			return mfFile, idx, nil
		}
	}
	return nil, 0, errors.Errorf("file %s not found in manifest", fileName)
}

func dataReadersForManifest(file *os.File, manifest *v1.DBExportManifest) ([]func() io.Reader, error) {
	st, err := file.Stat()
	if err != nil {
		return nil, errors.Wrapf(err, "could not stat %s", file.Name())
	}
	zipReader, err := zip.NewReader(file, st.Size())
	if err != nil {
		return nil, errors.Wrapf(err, "could not open file %s as ZIP file", file.Name())
	}

	readers := make([]func() io.Reader, len(manifest.GetFiles()))
	for _, entry := range zipReader.File {
		if strings.HasSuffix(entry.Name, "/") {
			continue // ignore directories
		}

		manifestFile, idx, err := findManifestFile(entry.Name, manifest)
		if err != nil {
			return nil, err
		}

		expectedCompressionType := v1.DBExportManifest_UNCOMPREESSED
		expectedEncodedSize := int64(entry.CompressedSize64)
		decode := false
		switch entry.Method {
		case zip.Store:
		case zip.Deflate:
			expectedCompressionType = v1.DBExportManifest_DEFLATED
		default:
			expectedEncodedSize = int64(entry.UncompressedSize64)
			decode = true
		}

		if manifestFile.GetEncoding() != expectedCompressionType {
			return nil, errors.Errorf("file %s is encoded as %v in ZIP file, but expected as %v per manifest", entry.Name, expectedCompressionType, manifestFile.GetEncoding())
		}

		if manifestFile.GetEncodedSize() != expectedEncodedSize {
			return nil, errors.Errorf("file %s has an encoded length of %d in the ZIP file, but expected to be %d per manifest", entry.Name, expectedEncodedSize, manifestFile.GetEncodedSize())
		}

		if manifestFile.GetDecodedCrc32() != entry.CRC32 {
			return nil, errors.Errorf("file %s has mismatching CRC32 checksum: %x in ZIP file versus %x in manifest", entry.Name, entry.CRC32, manifestFile.GetDecodedCrc32())
		}

		var readerFunc func() io.Reader
		if !decode {
			compressedLen := int64(entry.CompressedSize64)
			offset, err := entry.DataOffset()
			if err != nil {
				return nil, errors.Wrapf(err, "could not determine data offset of file %s within ZIP", entry.Name)
			}
			readerFunc = func() io.Reader {
				return io.NewSectionReader(file, offset, compressedLen)
			}
		} else {
			readerFunc = func() io.Reader {
				reader, err := entry.Open()
				if err != nil {
					return ioutils.ErrorReader(err)
				}
				return reader
			}
		}
		readers[idx] = readerFunc
	}

	for idx, manifestFile := range manifest.GetFiles() {
		if readers[idx] == nil {
			return nil, errors.Errorf("file %s has no associated data reader", manifestFile.GetName())
		}
	}

	return readers, nil
}

func assembleManifestFromZIP(file *os.File, supportedCompressionTypes map[v1.DBExportManifest_EncodingType]struct{}) (*v1.DBExportManifest, error) {
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	zipReader, err := zip.NewReader(file, stat.Size())
	if err != nil {
		return nil, err
	}

	mf := &v1.DBExportManifest{}

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

		if _, formatSupported := supportedCompressionTypes[compressionType]; formatSupported {
			manifestFile.Encoding = compressionType
			manifestFile.EncodedSize = int64(entry.CompressedSize64)
		} else {
			manifestFile.Encoding = v1.DBExportManifest_UNCOMPREESSED
			manifestFile.EncodedSize = manifestFile.DecodedSize
		}

		mf.Files = append(mf.Files, manifestFile)
	}

	return mf, nil
}

// tryRestoreV2 attempts to restore the database using the V2 backup/restore API. If the API is not supported by
// central, `ErrV2RestoreNotSupported` is returned. Otherwise, the error indicates whether the restore process was
// successful.
func tryRestoreV2(cmd *cobra.Command, file *os.File, deadline time.Time) error {
	restorer, err := newV2Restorer(cmd, deadline)
	if err != nil {
		return err
	}

	resp, err := restorer.Run(common.Context(), file)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(resp.Body.Close)

	return httputil.ResponseToError(resp)
}

func restoreV2(cmd *cobra.Command, file *os.File, deadline time.Time) error {
	err := tryRestoreV2(cmd, file, deadline)
	if err == ErrV2RestoreNotSupported {
		err = restoreV1(file, deadline)
	}
	return err
}
