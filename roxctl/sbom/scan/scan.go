package scan

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/printers"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stackrox/rox/roxctl/common/scan"
	"github.com/stackrox/rox/roxctl/common/util"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	sbomScanAPIPath = "/api/v1/sboms/scan"
)

var (
	validSeverities = scan.AllSeverities()
)

// Command detects vulnerabilities from SBOM contents.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	sbomScanCmd := &sbomScanCommand{env: cliEnvironment}

	objectPrinterFactory, err := printer.NewObjectPrinterFactory("",
		printer.NewTabularPrinterFactoryWithAutoMerge(),
		printer.NewJSONPrinterFactory(false, false),
		printer.NewSarifPrinterFactory(
			printers.SarifVulnerabilityReport,
			scan.SarifJSONPathExpressions,
			&sbomScanCmd.sbomFilePath),
	)
	// should not happen when using default values, must be a programming error.
	utils.Must(err)

	// Set the Output Format to empty, by default raw unformatted json will be printed.
	objectPrinterFactory.OutputFormat = ""

	c := &cobra.Command{
		Use:   "scan",
		Short: "[DEV PREVIEW] Scan the specified SBOM and return scan results",
		Long:  "[DEV PREVIEW] Scan the specified SBOM and return scan results. You must have write permissions to the `Image` resource. Currently supports SPDX 2.3 JSON documents with content types: [`application/spdx+json`, `text/spdx+json`].",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			if err := sbomScanCmd.Construct(nil, c, objectPrinterFactory); err != nil {
				return err
			}

			if err := sbomScanCmd.Validate(); err != nil {
				return err
			}

			return sbomScanCmd.ScanSBOM()
		}),
	}

	objectPrinterFactory.AddFlags(c)

	c.Flags().StringVarP(&sbomScanCmd.sbomFilePath, "file", "", "", "SBOM file to scan. Must be SPDX 2.3 JSON.")
	c.Flags().StringVarP(&sbomScanCmd.contentType, "content-type", "", "", "Set the content-type for the SBOM file, if unset will be auto-detected.")
	c.Flags().StringSliceVar(&sbomScanCmd.severities, "severity", validSeverities, "List of severities to include in the output. Use this to filter for specific severities.")
	c.Flags().BoolVarP(&sbomScanCmd.failOnFinding, "fail", "", false, "Fail if vulnerabilities have been found.")

	utils.Must(c.MarkFlagRequired("file"))

	return c
}

// sbomScanCommands holds all configurations and metadata to execute an SBOM scan.
type sbomScanCommand struct {
	sbomFilePath  string
	contentType   string
	severities    []string
	failOnFinding bool

	// injected or constructed values
	env                environment.Environment
	client             common.RoxctlHTTPClient
	printer            printer.ObjectPrinter
	standardizedFormat bool
	noOutputFormat     bool
}

// Construct will enhance the struct with other values coming either from os.Args, other, global flags or environment variables.
func (s *sbomScanCommand) Construct(_ []string, cmd *cobra.Command, f *printer.ObjectPrinterFactory) error {
	var err error

	if f.OutputFormat == "" {
		s.noOutputFormat = true
	} else {
		p, err := f.CreatePrinter()
		if err != nil {
			return errors.Wrap(err, "could not create printer for displaying sbom scan result")
		}
		s.printer = p
		s.standardizedFormat = f.IsStandardizedFormat()
	}

	s.client, err = s.env.HTTPClient(
		flags.Timeout(cmd),
		// Do not retry. Otherwise the http client's default retry count and delay make the scan
		// appear hung when timeout expires.
		common.WithRetryCount(0),
	)
	if err != nil {
		return errors.Wrap(err, "creating HTTP client")
	}

	return nil
}

// Validate will validate the injected values and check whether it's possible to execute the operation with the
// provided values.
func (s *sbomScanCommand) Validate() error {
	// Check if the SBOM file exists.
	if _, err := os.Stat(s.sbomFilePath); err != nil {
		if os.IsNotExist(err) {
			return errox.InvalidArgs.Newf("SBOM file does not exist: %q", s.sbomFilePath)
		}
		return errors.Wrapf(err, "checking SBOM file %q", s.sbomFilePath)
	}

	for _, severity := range s.severities {
		severity := strings.ToUpper(severity)
		if !slices.Contains(validSeverities, severity) {
			return errox.InvalidArgs.Newf("invalid severity %q used. Choose one of [%s]", severity,
				strings.Join(validSeverities, ", "))
		}
	}

	return nil
}

// Scan will execute the SBOM scan with retry functionality.
func (s *sbomScanCommand) ScanSBOM() error {
	// Open the SBOM file for reading.
	sbomFile, err := os.Open(s.sbomFilePath)
	if err != nil {
		return fmt.Errorf("opening SBOM file: %w", err)
	}
	defer utils.IgnoreError(sbomFile.Close)

	// Guess the media type.
	if s.contentType == "" {
		s.contentType, err = guessMediaType(sbomFile)
		if err != nil {
			return errors.Wrap(err, "auto detecting media type")
		}
	}

	// Make the scan request.
	req, err := s.client.NewReq(http.MethodPost, sbomScanAPIPath, sbomFile)
	if err != nil {
		return errors.Wrap(err, "creating SBOM scan request")
	}
	req.Header.Add("Content-Type", s.contentType)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("scanning SBOM: %w", err)
	}
	defer utils.IgnoreError(resp.Body.Close)

	// Verify the scan was successful.
	if resp.StatusCode != http.StatusOK {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrapf(err, "received unexpected status code %d. Additionally, there was an error reading the response", resp.StatusCode)
		}
		return errox.InvariantViolation.Newf("received unexpected status code %d. Response Body: %s", resp.StatusCode, string(data))
	}

	// Central returns a 200 response with Content-Type text/html for any unimplemented '/api/*' endpoints,
	// catch this and return an error.
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		return errors.Errorf("received unexpected Content-Type %q from Central, confirm Central version supports SBOM scanning at: %q", contentType, sbomScanAPIPath)
	}

	// Print the results.
	return s.printSBOMScanResults(resp.Body)
}

// guessMediaType will attempt to guess the media type of the SBOM file based on the first
// 4KB bytes. If it is unable to guess will return an error.
//
// The backend currently requires SPDX 2.3 JSON, which when detected will return
// media type `text/spdx+json`.
func guessMediaType(sbomFile *os.File) (string, error) {
	// Read 4KB of the file, should be enough to detect the SPDX metadata.
	buf := make([]byte, 4096)
	n, err := sbomFile.Read(buf)
	if err != nil && err != io.EOF {
		return "", errors.Wrap(err, "reading SBOM file")
	}

	// Reset file pointer to beginning so the file can be read again.
	if _, err := sbomFile.Seek(0, 0); err != nil {
		return "", errors.Wrap(err, "resetting file position")
	}

	content := string(buf[:n])

	// Skip UTF-8 BOM if present (0xEF, 0xBB, 0xBF).
	content = strings.TrimPrefix(content, "\xEF\xBB\xBF")

	// Quick check if content looks like JSON by checking if it starts with { or [.
	trimmed := strings.TrimLeft(content, " \t\n\r")
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		return "", errox.InvalidArgs.New("SBOM file does not appear to be valid JSON")
	}

	// Look for the spdxVersion field.
	if idx := strings.Index(content, `"spdxVersion"`); idx != -1 {
		remaining := content[idx+len(`"spdxVersion"`):]
		// Find the colon after the field name.
		colonIdx := strings.Index(remaining, ":")
		if colonIdx != -1 {
			// Get content after the colon.
			afterColon := remaining[colonIdx+1:]
			// Remove whitespace only.
			afterColon = strings.TrimLeft(afterColon, " \t\n\r")
			if strings.HasPrefix(afterColon, `"SPDX-2.3"`) {
				return "text/spdx+json", nil
			}
		}
	}

	return "", errox.InvalidArgs.New("unsupported SBOM format")
}

// printSBOMScanResults prints the SBOM results using the appropriate format
// specified via the command flags. The output format will resemble
// that for the `image scan` command.
func (s *sbomScanCommand) printSBOMScanResults(reader io.Reader) error {
	if s.noOutputFormat {
		// Write the raw contents by default.
		_, err := io.Copy(s.env.InputOutput().Out(), reader)
		if err != nil {
			return errors.Wrap(err, "reading response body")
		}
		return nil
	}

	// To re-use formatting logic from elsewhere in roxctl, we marshal the contents into
	// storage.Image which 'should have' overlapping field structure with v1.SBOMScanResponse.
	data, err := io.ReadAll(reader)
	if err != nil {
		return errors.Wrap(err, "reading response body")
	}

	image := &storage.Image{}
	err = protojson.Unmarshal(data, image)
	if err != nil {
		return errors.Wrap(err, "unmarshalling response")
	}

	cveSummary := scan.NewCVESummaryForPrinting(image.GetScan(), s.severities)
	if !s.standardizedFormat {
		s.env.Logger().PrintfLn("Scan results for SBOM: %s", s.sbomFilePath)
		scan.PrintCVESummary(cveSummary.Result.Summary, s.env.Logger())
	}

	if err := s.printer.Print(cveSummary, s.env.ColorWriter()); err != nil {
		return errors.Wrap(err, "could not print scan results")
	}

	if !s.standardizedFormat {
		scan.PrintCVEWarning(cveSummary.CountVulnerabilities(), cveSummary.CountComponents(), s.env.Logger())
	}

	if cveCount := cveSummary.CountVulnerabilities(); s.failOnFinding && cveCount > 0 {
		//nolint:wrapcheck // Preserving error message from scan package for consistent CLI output.
		return scan.NewErrVulnerabilityFound(cveCount)
	}

	return nil
}
