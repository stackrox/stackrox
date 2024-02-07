package downloaddb

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/roxctl/common/environment"
)

// TODO:
// - Verify adheres to style guide: https://github.com/stackrox/architecture-decision-records/blob/main/stackrox/ADR-0004-roxctl-subcommands-layout.md
// - Also verify in alignment with: https://github.com/stackrox/stackrox/blob/master/roxctl/README.md

const (
	contentLengthHdrKey = "Content-Length"

	// TODO: Pending decision to put vulns in subdir
	// bundleFileNameFmt = "/%[1]s/scanner-vulns-%[1]s.zip"
	bundleFileNameFmt = "scanner-vulns-%[1]s.zip"

	latestBundleFileName = "scanner-vuln-updates.zip"
)

type scannerDownloadDBCommand struct {
	// Properties that are bound to cobra flags.
	version     string
	force       bool
	skipCentral bool
	filename    string

	// filenameValidated is set to true if filename is non-empty and has
	// already been validated, this ensures the same file isn't validated
	// repeatedly.
	filenameValidated bool

	// Properties that are injected or constructed.
	env environment.Environment
}

func (cmd *scannerDownloadDBCommand) construct(_ *cobra.Command) {
}

func (cmd *scannerDownloadDBCommand) downloadDb() error {
	version := cmd.detectVersion()

	priorToV4, err := isPriorToScannerV4(version)
	if err != nil {
		return fmt.Errorf("invalid version %q: %w", version, err)
	}

	var bundleFileNames []string
	if priorToV4 {
		cmd.env.Logger().InfofLn("Version represents StackRox Scanner, downloading 'latest' bundle.")
		bundleFileNames = append(bundleFileNames, latestBundleFileName)
	} else {
		versionVariants := disectVersion(version)
		for _, versionVariant := range versionVariants {
			bundleFileNames = append(bundleFileNames, fmt.Sprintf(bundleFileNameFmt, versionVariant))
		}
	}

	var errs []error
	for _, bundleFileName := range bundleFileNames {
		// Get the name of the output file and ensures its valid.
		outFileName, err := cmd.buildAndValidateOutputFileName(bundleFileName)
		if err != nil {
			// If there was an error validating the output file, assume the file exists
			// and therefore was successfully created in the past. Do not continue
			// processing other variants.
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

// detectVersion attempts to determine an appropriate base version to use.
func (cmd *scannerDownloadDBCommand) detectVersion() string {
	if cmd.version != "" {
		cmd.env.Logger().InfofLn("Using the version from command line flag: %q", cmd.version)
		return cmd.version
	}

	if !cmd.skipCentral {
		if ver, err := cmd.versionFromCentral(); err == nil {
			cmd.env.Logger().InfofLn("Using version from Central: %q", ver)
			return ver
		}
	}

	ver := version.GetMainVersion()
	cmd.env.Logger().InfofLn("Using version from roxctl binary: %q", ver)
	return ver
}

func (cmd *scannerDownloadDBCommand) versionFromCentral() (string, error) {
	client, err := cmd.env.HTTPClient(5 * time.Second)
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
	if err := jsonpb.Unmarshal(resp.Body, &metadata); err != nil {
		cmd.env.Logger().WarnfLn("error reading metadata from central: %v", err)
		return "", err
	}

	if err := resp.Body.Close(); err != nil {
		return "", fmt.Errorf("could not close respond body: %w", err)
	}

	return metadata.GetVersion(), nil
}

func (cmd *scannerDownloadDBCommand) buildAndValidateOutputFileName(outFileName string) (string, error) {
	if cmd.filename != "" {
		outFileName = cmd.filename

		if cmd.filenameValidated {
			return outFileName, nil
		}
		cmd.filenameValidated = true
	}

	// Throw an error if the file exists and force flag not used
	if !cmd.force {
		if _, err := os.Stat(outFileName); err == nil {
			return "", fmt.Errorf("file %q already exists, to overwrite use `--force`", outFileName)
		}
	}

	return outFileName, nil
}

func (cmd *scannerDownloadDBCommand) buildDownloadURL(bundleFileName string) (string, error) {
	return url.JoinPath(env.ScannerDBDownloadBaseURL.Setting(), bundleFileName)
}

func (cmd *scannerDownloadDBCommand) downloadVulnDB(url string, outFileName string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(resp.Body.Close)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading %q failed with status code %d: %s", url, resp.StatusCode, resp.Status)
	}

	outFile, err := os.Create(outFileName)
	if err != nil {
		return err
	}
	defer utils.IgnoreError(outFile.Close)

	var fileSize int64
	if fileSizeStrs, ok := resp.Header[contentLengthHdrKey]; ok {
		fileSizeI, err := strconv.ParseInt(fileSizeStrs[0], 10, 64)
		if err == nil {
			fileSize = fileSizeI
		}
	}

	cmd.env.Logger().InfofLn("Downloading %q (%d MiB)", url, fileSize/1024/1024)
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return err
	}

	if err := outFile.Close(); err != nil {
		return fmt.Errorf("could not close out file: %w", err)
	}

	if err := resp.Body.Close(); err != nil {
		return fmt.Errorf("could not close response body: %w", err)
	}

	return nil
}

// disectVersion breaks a version into a series of version strings starting with
// the most specific to the least specific.
func disectVersion(version string) []string {
	res := []string{version}

	i := strings.LastIndex(version, "-")
	for i != -1 {
		res = append(res, version[:i])
		version = version[:i]
		i = strings.LastIndex(version, "-")
	}

	i = strings.LastIndex(version, ".")
	for i != -1 && strings.Count(version, ".") > 1 {
		res = append(res, version[:i])
		version = version[:i]
		i = strings.LastIndex(version, ".")
	}

	return res
}

// isPriorToScannerV4 returns true if version represents a version of ACS from prior to the
// introduction of Scanner V4. Will return an error if cannot determine result.
func isPriorToScannerV4(version string) (bool, error) {
	// 3.99.99 = Scanner V2
	// 4.3.99  = Scanner V2
	// 4.3.x   = Scanner V4
	// 4.4.*   = Scanner V4
	before, _, _ := strings.Cut(version, "-")
	parts := strings.Split(before, ".")

	if len(parts) < 2 || len(parts) > 3 {
		return false, fmt.Errorf("%q is not in X.Y[.Z] format", before)
	}

	x, err := strconv.Atoi(parts[0])
	if err != nil {
		return false, fmt.Errorf("x is not numeric: %q", version)
	}

	y, err := strconv.Atoi(parts[1])
	if err != nil {
		return false, fmt.Errorf("y is not numeric: %q", version)
	}

	var z string
	if len(parts) > 2 {
		z = parts[2]
	}

	if (x < 4 || y < 3) || (y == 3 && (z == "" || z != "x")) {
		return true, nil
	}

	return false, nil
}

// Command represents the command.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	scannerDownloadDBCmd := &scannerDownloadDBCommand{env: cliEnvironment}

	c := &cobra.Command{
		Use:   "download-db",
		Short: "Download the offline vulnerability database for StackRox Scanner and/or Scanner V4.",
		Long: `Download the offline vulnerability database for StackRox Scanner and/or Scanner V4.

Download version specific offline vulnerability bundles. Will contact
Central to determine version if one is not specified, if communication fails
defaults to version embedded within roxctl.`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			scannerDownloadDBCmd.construct(c)

			return scannerDownloadDBCmd.downloadDb()
		},
	}

	c.Flags().StringVar(&scannerDownloadDBCmd.version, "version", "", "Download a specific version of the vulnerability database (default: auto-detect)")
	c.Flags().StringVar(&scannerDownloadDBCmd.filename, "scanner-db-file", "", "Output file to save the vulnerability database to (default: remote filename)")
	c.Flags().BoolVar(&scannerDownloadDBCmd.force, "force", false, "Force overwriting output file if it already exists")
	c.Flags().BoolVar(&scannerDownloadDBCmd.skipCentral, "skip-central", false, "Do not contact Central when detecting version")

	return c
}
