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
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/roxctl/common/environment"
)

const (
	contentLengthHdrKey = "Content-Length"

	// TODO: Pending decision to put vulns in subdir
	// bundleFileNameFmt = "/%[1]s/scanner-vulns-%[1]s.zip"
	bundleFileNameFmt = "scanner-vulns-%[1]s.zip"

	latestBundleFileName = "scanner-vuln-updates.zip"
)

type scannerDownloadDBCommand struct {
	// Properties that are bound to cobra flags.
	version  string
	force    bool
	filename string

	// filenameValidated is set to true if filename is non-empty and has
	// already been validated, this ensures the same file isn't validated
	// repeatedly.
	filenameValidated bool

	// Properties that are injected or constructed.
	env environment.Environment
}

func (cmd *scannerDownloadDBCommand) downloadDb() error {
	version := cmd.detectVersion()

	// Get version variants.
	versionVariants, isScannerV2 := cmd.disectVersion(version)
	if versionVariants == nil && !isScannerV2 {
		return fmt.Errorf("unexpected error parsing version")
	}

	var bundleFileNames []string
	if isScannerV2 {
		cmd.env.Logger().InfofLn("Version represents StackRox Scanner, downloading 'latest' bundle.")
		bundleFileNames = append(bundleFileNames, latestBundleFileName)
	} else {
		for _, versionVariant := range versionVariants {
			bundleFileName := fmt.Sprintf(bundleFileNameFmt, versionVariant)
			bundleFileNames = append(bundleFileNames, bundleFileName)
		}
	}

	var errs []error
	for _, bundleFileName := range bundleFileNames {
		// Get the name of the output file and ensures its valid.
		outFileName, err := cmd.buildAndValidateOutputFile(bundleFileName)
		if err != nil {
			return fmt.Errorf("invalid output file %q: %v", bundleFileName, err)
		}

		// Get the URL from which to download the vulnerability db.
		url, err := cmd.buildDownloadURL(bundleFileName)
		if err != nil {
			errs = append(errs, fmt.Errorf("unable to build download URL for %q: %v", bundleFileName, err))
			continue
		}

		// Download the vulnerability database
		err = cmd.downloadVulnDB(url, outFileName)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		cmd.env.Logger().PrintfLn("\nSuccessfully downloaded database to %q", outFileName)
		return nil
	}

	return errors.Join(errs...)
}

func (cmd *scannerDownloadDBCommand) detectVersion() string {
	if cmd.version != "" {
		cmd.env.Logger().InfofLn("Using the version from command line flag: %q", cmd.version)
		return cmd.version
	}

	if ver, err := cmd.versionFromCentral(); err == nil {
		cmd.env.Logger().InfofLn("Using version from Central: %q", ver)
		return ver
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
	defer resp.Body.Close()

	var metadata v1.Metadata
	if err := jsonpb.Unmarshal(resp.Body, &metadata); err != nil {
		cmd.env.Logger().WarnfLn("error reading metadata from central: %v", err)
		return "", err
	}

	return metadata.GetVersion(), nil
}

func (cmd *scannerDownloadDBCommand) buildAndValidateOutputFile(outFileName string) (string, error) {
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
	url, err := url.JoinPath(env.ScannerDBDownloadBaseURL.Setting(), bundleFileName)
	if err != nil {
		return "", err
	}

	return url, nil
}
func (cmd *scannerDownloadDBCommand) downloadVulnDB(url string, outFileName string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading %q failed with status code %d: %s", url, resp.StatusCode, resp.Status)
	}

	outFile, err := os.Create(outFileName)
	if err != nil {
		return err
	}
	defer outFile.Close()

	var fileSize int64
	if fileSizeStrs, ok := resp.Header[contentLengthHdrKey]; ok {
		fileSizeI, err := strconv.ParseInt(fileSizeStrs[0], 10, 64)
		if err == nil {
			fileSize = fileSizeI
		}
	}

	cmd.env.Logger().InfofLn("Downloading %q (%d MiB)", url, fileSize/1024/1024)
	written, err := io.Copy(outFile, resp.Body)
	if err != nil {
		return err
	}

	// TODO: Replace with checksum / signature in the future
	if fileSize != 0 && written != fileSize {
		return fmt.Errorf("expected and received file sizes differ! (expected %d, got %d)", fileSize, written)
	}

	return nil
}

// disectVersion breaks a version into a series of version strings starting with
// the most specific to the least specific. True will be returned if the
// version should be ignored and Scanner V2 assumed.  If the return is
// (nil, false) the version is invalid.
func (cmd *scannerDownloadDBCommand) disectVersion(version string) ([]string, bool) {
	res := []string{version}

	i := strings.LastIndex(version, "-")
	for i != -1 {
		res = append(res, version[:i])

		version = version[:i]
		i = strings.LastIndex(version, "-")
	}

	// Version should be in X.Y.Z format at this point
	parts := strings.Split(version, ".")
	if len(parts) < 3 && len(res) > 1 {
		cmd.env.Logger().ErrfLn("Error dissecting version, does not adhere to X.Y.Z-* format: %q", version)
		return nil, false
	}

	x, err := strconv.Atoi(parts[0])
	if err != nil {
		cmd.env.Logger().ErrfLn("Error dissecting version, X part is not numeric: %q", version)
		return nil, false
	}

	y, err := strconv.Atoi(parts[1])
	if err != nil {
		cmd.env.Logger().ErrfLn("Error dissecting version, Y part is not numeric: %q", version)
		return nil, false
	}

	var z string
	if len(parts) > 2 {
		z = parts[2]
	}

	if isScannerV2(x, y, z) {
		return nil, true
	}

	// Skip append if Z is empty because X.Y would have already been added.
	if z != "" {
		res = append(res, fmt.Sprintf("%s.%s", parts[0], parts[1]))
	}
	return res, false
}

// isScannerV2 returns true if x, y, z represent a version from prior to the
// introduction of Scanner V4.
func isScannerV2(x int, y int, z string) bool {
	// Examples:
	//	 3.99.99 = Scanner V2
	//	 4.3.99  = Scanner V2
	//	 4.3.x   = Scanner V4
	//	 4.4.*   = Scanner V4

	if x < 4 || y < 3 {
		return true
	}

	if y == 3 && (z == "" || z != "x") {
		return true
	}

	return false
}

// Command represents the command.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	scannerDownloadDBCmd := &scannerDownloadDBCommand{env: cliEnvironment}

	c := &cobra.Command{
		Use:   "download-db",
		Short: "Download the offline vulnerability database for the StackRox Scanner and/or Scanner V4.",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			return scannerDownloadDBCmd.downloadDb()
		},
	}

	c.Flags().StringVarP(&scannerDownloadDBCmd.version, "version", "v", "", "Download a specific version of the vulnerability database (by default will auto-detect).")
	c.Flags().StringVar(&scannerDownloadDBCmd.filename, "scanner-db-file", "", "Output file to save the vulnerability database to.")
	c.Flags().BoolVar(&scannerDownloadDBCmd.force, "force", false, "Force overwriting output file if exists")

	return c
}
