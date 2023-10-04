package restore

import (
	"archive/zip"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/ioutils"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
)

const (
	checkCapsTimeout = 30 * time.Second
)

var (
	// ErrV2RestoreNotSupported is the error returned to indicate that the server does not support the new
	// backup/restore mechanism.
	ErrV2RestoreNotSupported = errox.InvariantViolation.New("server does not support V2 restore functionality")
)

type centralDbRestoreCommand struct {
	// Properties that are bound to cobra flags.
	file      string
	interrupt bool

	// Properties that are injected or constructed.
	env          environment.Environment
	timeout      time.Duration
	retryTimeout time.Duration
	confirm      func() error
}

// V2Command defines the new db restore command
func V2Command(cliEnvironment environment.Environment) *cobra.Command {
	centralDbRestoreCmd := &centralDbRestoreCommand{env: cliEnvironment}
	c := &cobra.Command{
		Use:   "restore <file>",
		Args:  cobra.ExactArgs(1),
		Short: "Restore the StackRox database from a previous backup.",
		Long:  "Restore the StackRox database from a backup (.zip file) that you created by using the `roxctl central db backup` command.",
		RunE: func(c *cobra.Command, args []string) error {
			if err := validate(c); err != nil {
				return err
			}
			if err := centralDbRestoreCmd.construct(c, args); err != nil {
				return err
			}
			if err := centralDbRestoreCmd.validate(); err != nil {
				return err
			}
			return centralDbRestoreCmd.restore(func(file *os.File, deadline time.Time) error {
				return centralDbRestoreCmd.restoreV2(file, deadline)
			})
		},
	}

	c.AddCommand(v2RestoreStatusCmd(cliEnvironment))
	c.AddCommand(v2RestoreCancelCommand(cliEnvironment))

	c.Flags().StringVar(&centralDbRestoreCmd.file, "file", "", "file to restore the DB from (deprecated; use positional argument)")
	c.Flags().BoolVar(&centralDbRestoreCmd.interrupt, "interrupt", false, "interrupt ongoing restore process (if any) to allow resuming")
	utils.Must(c.Flags().MarkDeprecated("file", "--file is deprecated; use the positional argument instead"))
	flags.AddForce(c)

	return c
}

func validate(cbr *cobra.Command) error {
	if file, _ := cbr.Flags().GetString("file"); file != "" {
		return errox.InvalidArgs.New("legacy --file flag must not be used in conjunction with a positional argument")
	}
	return nil
}

func (cmd *centralDbRestoreCommand) construct(cbr *cobra.Command, args []string) error {
	cmd.confirm = func() error {
		return flags.CheckConfirmation(cbr, cmd.env.Logger(), cmd.env.InputOutput())
	}
	cmd.timeout = flags.Timeout(cbr)
	cmd.retryTimeout = flags.RetryTimeout(cbr)
	if cmd.file == "" {
		cmd.file = args[0]
	}
	return nil
}

func (cmd *centralDbRestoreCommand) validate() error {
	if cmd.file == "" {
		return errox.InvalidArgs.New("file to restore from must be specified")
	}
	fi, err := os.Stat(cmd.file)
	if err != nil {
		if os.IsNotExist(err) {
			return errox.NotFound.Newf("file %q could not be found", cmd.file)
		}
		return errox.InvalidArgs.Newf("opening file %q", cmd.file)
	}

	if fi.IsDir() {
		return errox.InvalidArgs.Newf("expected a file not a directory for path %s", cmd.file)
	}

	return nil
}

func findManifestFile(fileName string, manifest *v1.DBExportManifest) (*v1.DBExportManifest_File, int, error) {
	for idx, mfFile := range manifest.GetFiles() {
		if mfFile.GetName() == fileName {
			return mfFile, idx, nil
		}
	}
	return nil, 0, errox.NotFound.Newf("file %s not found in manifest", fileName)
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
			return nil, errox.InvalidArgs.Newf("file %s is encoded as %v in ZIP file, but expected as %v per manifest", entry.Name, expectedCompressionType, manifestFile.GetEncoding())
		}

		if manifestFile.GetEncodedSize() != expectedEncodedSize {
			return nil, errox.InvalidArgs.Newf("file %s has an encoded length of %d in the ZIP file, but expected to be %d per manifest", entry.Name, expectedEncodedSize, manifestFile.GetEncodedSize())
		}

		if manifestFile.GetDecodedCrc32() != entry.CRC32 {
			return nil, errox.InvalidArgs.Newf("file %s has mismatching CRC32 checksum: %x in ZIP file versus %x in manifest", entry.Name, entry.CRC32, manifestFile.GetDecodedCrc32())
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
			return nil, errox.NotFound.Newf("file %s has no associated data reader", manifestFile.GetName())
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
func (cmd *centralDbRestoreCommand) tryRestoreV2(confirm func() error, file *os.File, deadline time.Time) error {
	restorer, err := cmd.newV2Restorer(confirm, deadline)
	if err != nil {
		return err
	}

	resp, err := restorer.Run(pkgCommon.Context(), file)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(resp.Body.Close)

	return httputil.ResponseToError(resp)
}

func (cmd *centralDbRestoreCommand) restoreV2(file *os.File, deadline time.Time) error {
	err := cmd.tryRestoreV2(cmd.confirm, file, deadline)
	if err == ErrV2RestoreNotSupported {
		err = cmd.restoreV1(file, deadline)
	}
	return err
}
