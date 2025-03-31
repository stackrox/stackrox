package downloaddb

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/utils"
	pkgVersion "github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
)

const (
	contentLengthHdrKey = "Content-Length"

	bundleFileNameFmt    = "%[1]s/scanner-vulns-%[1]s.zip"
	latestBundleFileName = "scanner-vuln-updates.zip"
)

type scannerDownloadDBCommand struct {
	// Properties that are bound to cobra flags.
	filename     string
	force        bool
	skipCentral  bool
	skipVariants bool
	timeout      time.Duration
	version      string

	// filenameValidated is set to true if filename is non-empty and has
	// already been validated, this ensures the same file isn't validated
	// repeatedly when processing version variants.
	filenameValidated bool

	// Properties that are injected or constructed.
	env environment.Environment
}

func (cmd *scannerDownloadDBCommand) construct(c *cobra.Command) {
	cmd.timeout = flags.Timeout(c)
}

func (cmd *scannerDownloadDBCommand) downloadDB() error {
	// Get the list of file names to attempt to download.
	bundleFileNames, err := cmd.buildBundleFileNames()
	if err != nil {
		return err
	}

	var errs []error
	for _, bundleFileName := range bundleFileNames {
		// Get the name of the output file and ensures it's valid.
		outFileName, err := cmd.buildAndValidateOutputFileName(bundleFileName)
		if err != nil {
			return fmt.Errorf("invalid output file %q: %w", bundleFileName, err)
		}

		// Get the URL from which to download the vulnerability db.
		url, err := cmd.buildDownloadURL(bundleFileName)
		if err != nil {
			errs = append(errs, fmt.Errorf("unable to build download URL for %q: %w", bundleFileName, err))
			continue
		}

		// Download the vulnerability database.
		err = cmd.downloadVulnDB(url, outFileName)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		cmd.env.Logger().PrintfLn("\nSuccessfully downloaded %q", outFileName)
		return nil
	}

	return errors.Join(errs...)
}

// buildBundleFileNames builds a list of filenames to attempt to download.
func (cmd *scannerDownloadDBCommand) buildBundleFileNames() ([]string, error) {
	version := cmd.detectVersion()

	priorToV4, err := pkgVersion.IsPriorToScannerV4(version)
	if err != nil {
		return nil, fmt.Errorf("invalid version %q: %w", version, err)
	}

	var bundleFileNames []string
	if priorToV4 {
		cmd.env.Logger().InfofLn("Version represents StackRox Scanner, downloading 'latest' bundle.")
		bundleFileNames = append(bundleFileNames, latestBundleFileName)
	} else if cmd.skipVariants {
		bundleFileNames = append(bundleFileNames, fmt.Sprintf(bundleFileNameFmt, version))
	} else {
		versionVariants, err := pkgVersion.Variants(version)
		if err != nil {
			return nil, fmt.Errorf("invalid version %q: %w", version, err)
		}
		for _, versionVariant := range versionVariants {
			bundleFileNames = append(bundleFileNames, fmt.Sprintf(bundleFileNameFmt, versionVariant))
		}
	}

	return bundleFileNames, nil
}

// detectVersion attempts to determine an appropriate base version to use.
func (cmd *scannerDownloadDBCommand) detectVersion() string {
	if cmd.version != "" {
		cmd.env.Logger().InfofLn("Using version from command line: %q", cmd.version)
		return cmd.version
	}

	if !cmd.skipCentral {
		if ver, err := cmd.versionFromCentral(); err == nil {
			cmd.env.Logger().InfofLn("Using version from Central: %q", ver)
			return ver
		}
	}

	ver := pkgVersion.GetMainVersion()
	cmd.env.Logger().InfofLn("Using version from roxctl binary: %q", ver)
	return ver
}

// versionFromCentral attempts to pull version from Central's metadata
// service.
func (cmd *scannerDownloadDBCommand) versionFromCentral() (string, error) {
	client, err := cmd.env.HTTPClient(cmd.timeout)
	if err != nil {
		cmd.env.Logger().WarnfLn("issue building central http client: %v", err)
		return "", err
	}

	resp, err := client.DoReqAndVerifyStatusCode("v1/metadata", http.MethodGet, http.StatusOK, nil)
	if err != nil {
		cmd.env.Logger().WarnfLn("error contacting central: %v", err)
		return "", err
	}
	defer utils.IgnoreError(resp.Body.Close)

	var metadata v1.Metadata
	if err := jsonutil.JSONReaderToProto(resp.Body, &metadata); err != nil {
		cmd.env.Logger().WarnfLn("error reading metadata from central: %v", err)
		return "", err
	}

	return metadata.GetVersion(), nil
}

// buildAndValidateOutputFileName returns a validated output file name where
// downloaded data should be stored.
func (cmd *scannerDownloadDBCommand) buildAndValidateOutputFileName(bundleFileName string) (string, error) {
	outFileName := bundleFileName

	if cmd.filename != "" {
		outFileName = cmd.filename

		if cmd.filenameValidated {
			return outFileName, nil
		}
		cmd.filenameValidated = true
	}

	if !cmd.force {
		// Throw an error if the file exists and force flag not set.
		if _, err := os.Stat(outFileName); err == nil {
			return "", fmt.Errorf("file %q already exists, to overwrite use `--force`", outFileName)
		}
	}

	return outFileName, nil
}

// buildDownloadURL returns the URL from which to download the vulnerability
// database from.
func (cmd *scannerDownloadDBCommand) buildDownloadURL(bundleFileName string) (string, error) {
	return url.JoinPath(env.ScannerDBDownloadBaseURL.Setting(), bundleFileName)
}

// httpClient builds a retryable http client for non-ACS requests (such as
// for downloading the vulnerability bundle from a public url).
func (cmd *scannerDownloadDBCommand) httpClient() *retryablehttp.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = env.ClientMaxRetries.IntegerSetting()
	retryClient.HTTPClient.Timeout = cmd.timeout
	retryClient.RetryWaitMin = 10 * time.Second

	return retryClient
}

// downloadVulnDB downloads the vulnerability database from url and stores it in
// the provided output file.
func (cmd *scannerDownloadDBCommand) downloadVulnDB(url string, outFileName string) error {
	resp, err := cmd.httpClient().Get(url)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading %q failed with status code %d: %s", url, resp.StatusCode, resp.Status)
	}

	err = os.MkdirAll(filepath.Dir(outFileName), 0700)
	if err != nil {
		return err
	}

	outFile, err := os.Create(outFileName)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(outFile.Close)

	var fileSize int64
	if fileSizeStr := resp.Header.Get(contentLengthHdrKey); fileSizeStr != "" {
		if fileSizeI, err := strconv.ParseInt(fileSizeStr, 10, 64); err == nil {
			fileSize = fileSizeI
		}
	}

	var size string
	if fileSize > 0 {
		size = fmt.Sprintf("(%d MiB)", fileSize/1024/1024)
	}

	cmd.env.Logger().InfofLn("Downloading %q %s", url, size)
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return err
	}

	if err := outFile.Close(); err != nil {
		return fmt.Errorf("could not close out file: %w", err)
	}

	return nil
}

// Command represents the command.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	scannerDownloadDBCmd := &scannerDownloadDBCommand{env: cliEnvironment}

	c := &cobra.Command{
		Use:   "download-db",
		Short: "Download the offline vulnerability database for StackRox Scanner and/or Scanner V4",
		Long: `Download the offline vulnerability database for StackRox Scanner and/or Scanner V4.

Download version specific offline vulnerability bundles. Will contact
Central to determine version if one is not specified, if communication fails
defaults to version embedded within roxctl.

By default will attempt to download the database for the determined version as
well as less specific variants. For example, given version "4.4.1-extra"
downloads will be attempted for the following version variants:
   4.4.1-extra
   4.4.1
   4.4

Use "--skip-variants" to only try the most specific version (i.e. "4.4.1-extra"
from the example above).`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			scannerDownloadDBCmd.construct(c)

			return scannerDownloadDBCmd.downloadDB()
		},
	}

	c.Flags().StringVar(&scannerDownloadDBCmd.version, "version", "", "Download a specific version (or version variant) of the vulnerability database (default: auto-detect).")
	c.Flags().StringVar(&scannerDownloadDBCmd.filename, "scanner-db-file", "", "Output file to save the vulnerability database to (default: remote filename).")
	c.Flags().BoolVar(&scannerDownloadDBCmd.force, "force", false, "Force overwriting the output file if it already exists.")
	c.Flags().BoolVar(&scannerDownloadDBCmd.skipCentral, "skip-central", false, "Do not contact Central when detecting version.")
	c.Flags().BoolVar(&scannerDownloadDBCmd.skipVariants, "skip-variants", false, "Do not attempt to process variants of the determined version.")
	flags.AddTimeoutWithDefault(c, 10*time.Minute)

	return c
}
