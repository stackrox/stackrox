package scan

import (
	"context"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/gjson"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/printers"
	"github.com/stackrox/rox/pkg/retry"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/logger"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stackrox/rox/roxctl/common/util"
)

const (
	deprecationNote = "please use --output/-o to specify the output format. " +
		"NOTE: The new JSON / CSV format contains breaking changes, make sure you adapt to the new structure before migrating."
)

var (
	// JSON Path expressions to use for sarif report generation
	sarifJSONPathExpressions = map[string]string{
		printers.SarifRuleJSONPathExpressionKey: gjson.MultiPathExpression(
			`@text:{"printKeys":"false","customSeparator":"_"}`,
			gjson.Expression{
				Expression: "result.vulnerabilities.#.cveId",
			},
			gjson.Expression{
				Expression: "result.vulnerabilities.#.componentName",
			},
			gjson.Expression{
				Expression: "result.vulnerabilities.#.componentVersion",
			},
		),
		printers.SarifHelpJSONPathExpressionKey: gjson.MultiPathExpression(
			"@text",
			gjson.Expression{
				Key:        "Vulnerability",
				Expression: "result.vulnerabilities.#.cveId",
			},
			gjson.Expression{
				Key:        "Link",
				Expression: "result.vulnerabilities.#.cveInfo",
			},
			gjson.Expression{
				Key:        "Severity",
				Expression: "result.vulnerabilities.#.cveSeverity",
			},
			gjson.Expression{
				Key:        "Component",
				Expression: "result.vulnerabilities.#.componentName",
			},
			gjson.Expression{
				Key:        "Version",
				Expression: "result.vulnerabilities.#.componentVersion",
			},
			gjson.Expression{
				Key:        "Fixed Version",
				Expression: "result.vulnerabilities.#.componentFixedVersion",
			},
		),
		printers.SarifSeverityJSONPathExpressionKey: "result.vulnerabilities.#.cveSeverity",
		printers.SarifHelpLinkJSONPathExpressionKey: "result.vulnerabilities.#.cveInfo",
	}

	// supported output formats with default values
	supportedObjectPrinters = []printer.CustomPrinterFactory{
		printer.NewTabularPrinterFactoryWithAutoMerge(),
		printer.NewJSONPrinterFactory(false, false),
	}
)

// Command checks the image against image build lifecycle policies
func Command(cliEnvironment environment.Environment) *cobra.Command {
	imageScanCmd := &imageScanCommand{env: cliEnvironment}

	objectPrinterFactory, err := printer.NewObjectPrinterFactory("table",
		append(supportedObjectPrinters,
			printer.NewSarifPrinterFactory(printers.SarifVulnerabilityReport, sarifJSONPathExpressions, &imageScanCmd.image))...)
	// should not happen when using default values, must be a programming error
	utils.Must(err)
	// Set the Output Format to empty, so by default the new output format will not be used and the legacy one will be
	// preferred and used. Once the output format is set, it will take precedence over the legacy one specified
	// via the --format flag.
	objectPrinterFactory.OutputFormat = ""

	c := &cobra.Command{
		Use:   "scan",
		Short: "Scan the specified image, and return scan results",
		Long:  "Scan the specified image and return the fully enriched image. Optionally, force a rescan of the image. You must have write permissions for the `Image` resource.",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			if err := imageScanCmd.Construct(nil, c, objectPrinterFactory); err != nil {
				return err
			}

			if err := imageScanCmd.Validate(); err != nil {
				return err
			}

			return imageScanCmd.Scan()
		}),
	}

	objectPrinterFactory.AddFlags(c)

	c.Flags().StringVarP(&imageScanCmd.image, "image", "i", "", "Image name and reference. (e.g. nginx:latest or nginx@sha256:...).")
	c.Flags().BoolVarP(&imageScanCmd.force, "force", "f", false, "Bypass Central's cache for the image and force a new pull from the Scanner.")
	c.Flags().BoolVarP(&imageScanCmd.includeSnoozed, "include-snoozed", "a", false, "The --include-snoozed flag returns both snoozed and unsnoozed CVEs if set.")
	c.Flags().IntVarP(&imageScanCmd.retryDelay, "retry-delay", "d", 3, "Set time to wait between retries in seconds.")
	c.Flags().IntVarP(&imageScanCmd.retryCount, "retries", "r", 3, "Number of retries before exiting as error.")
	c.Flags().StringVar(&imageScanCmd.cluster, "cluster", "", "Cluster name or ID to delegate image scan to.")
	c.Flags().StringSliceVar(&imageScanCmd.severities, "severity", []string{
		lowCVESeverity.String(),
		moderateCVESeverity.String(),
		importantCVESeverity.String(),
		criticalCVESeverity.String(),
	}, "List of severities to include in the output. Use this to filter for specific severities.")
	c.Flags().BoolVarP(&imageScanCmd.failOnFinding, "fail", "", false, "Fail if vulnerabilities have been found.")

	// Deprecated flag
	// The error message will be prefixed by "command <command-name> has been deprecated".
	//
	// TODO(ROX-29120): This may NOT be removed until we find another place to put this or we replace this with another equivalent format.
	c.Flags().StringVarP(&imageScanCmd.format, "format", "", "json", "Format of the output. Choose output format from json and csv.")
	utils.Must(c.Flags().MarkDeprecated("format", deprecationNote))

	utils.Must(c.MarkFlagRequired("image"))
	return c
}

// imageScanCommand holds all configurations and metadata to execute an image scan
type imageScanCommand struct {
	// properties bound to cobra flags
	image          string
	force          bool
	includeSnoozed bool
	format         string
	retryDelay     int
	retryCount     int
	timeout        time.Duration
	cluster        string
	severities     []string
	failOnFinding  bool

	// injected or constructed values
	env                environment.Environment
	printer            printer.ObjectPrinter
	standardizedFormat bool
}

// Construct will enhance the struct with other values coming either from os.Args, other, global flags or environment variables
func (i *imageScanCommand) Construct(_ []string, cmd *cobra.Command, f *printer.ObjectPrinterFactory) error {
	i.timeout = flags.Timeout(cmd)

	if err := imageUtils.IsValidImageString(i.image); err != nil {
		return common.ErrInvalidCommandOption.CausedBy(err)
	}

	// There is a case where cobra is not printing the deprecation warning to stderr, when a deprecated flag is not
	// specified, but has default values. So, when --format is left with default values and --output is not specified,
	// we manually print the deprecation note. We do not need to do this when i.e. --format csv is used, because
	// then a deprecated flag will be explicitly used and cobra will take over the printing of the deprecation note.
	if !cmd.Flag("format").Changed && !cmd.Flag("output").Changed {
		i.env.Logger().WarnfLn("Flag --format has been deprecated, %s", deprecationNote)
	}
	// Only create the printer when the old, deprecated output format is not used
	// TODO(ROX-8303): This can be removed once the old output format is fully deprecated
	if f.OutputFormat != "" {
		p, err := f.CreatePrinter()
		if err != nil {
			return errors.Wrap(err, "could not create printer for image scan result")
		}
		i.printer = p
		i.standardizedFormat = f.IsStandardizedFormat()
	}

	return nil
}

// Validate will validate the injected values and check whether it's possible to execute the operation with the
// provided values
func (i *imageScanCommand) Validate() error {
	if i.image == "" {
		return errox.InvalidArgs.New("no image name specified via the -i or --image flag")
	}

	// Only verify the legacy output format if no printer is constructed, thus the new output format is not used
	if i.printer == nil {
		// TODO(ROX-8303): this can be removed once the old output format is fully deprecated
		if i.format != "" && i.format != "json" && i.format != "csv" {
			return errox.InvalidArgs.Newf("invalid output format %q used. You can "+
				"only specify json or csv", i.format)
		}
	}

	validSeverities := []string{
		lowCVESeverity.String(),
		moderateCVESeverity.String(),
		importantCVESeverity.String(),
		criticalCVESeverity.String(),
	}

	for _, severity := range i.severities {
		severity := strings.ToUpper(severity)
		if !slices.Contains(validSeverities, severity) {
			return errox.InvalidArgs.Newf("invalid severity %q used. Choose one of [%s]", severity,
				strings.Join(validSeverities, ", "))
		}
	}

	return nil
}

// Scan will execute the image scan with retry functionality
func (i *imageScanCommand) Scan() error {
	var failedAttempts int
	err := retry.WithRetry(func() error {
		return i.scanImage()
	},
		retry.Tries(i.retryCount+1),
		retry.OnlyRetryableErrors(),
		retry.OnFailedAttempts(func(err error) {
			failedAttempts++
			i.env.Logger().ErrfLn("Scanning image failed: %v. Retrying after %v seconds...", err, i.retryDelay)
			time.Sleep(time.Duration(i.retryDelay) * time.Second)
		}),
	)
	if err != nil {
		if failedAttempts > 0 {
			return errors.Wrapf(err, "image scan failed after %d retries", failedAttempts)
		}
		return errors.Wrap(err, "image scan failed")
	}
	return nil
}

// scanImage will retrieve scan results from central and print them afterwards
func (i *imageScanCommand) scanImage() error {
	imageResult, err := i.getImageResultFromService()
	if err != nil {
		return errors.Wrap(common.MakeRetryable(err), "retrieving image scan result")
	}

	return i.printImageResult(imageResult)
}

// getImageResultFromService will retrieve the scan results for the specified image from
// central's ImageService
func (i *imageScanCommand) getImageResultFromService() (*storage.Image, error) {
	conn, err := i.env.GRPCConnection()
	if err != nil {
		return nil, errors.Wrap(err, "could not establish gRPC connection to central")
	}
	defer utils.IgnoreError(conn.Close)

	svc := v1.NewImageServiceClient(conn)

	ctx, cancel := context.WithTimeout(pkgCommon.Context(), i.timeout)
	defer cancel()

	image, err := svc.ScanImage(ctx, &v1.ScanImageRequest{
		ImageName:      i.image,
		Force:          i.force,
		IncludeSnoozed: i.includeSnoozed,
		Cluster:        i.cluster,
	})
	return image, errors.Wrapf(err, "could not scan image: %q", i.image)
}

// printImageResult print the storage.ImageScan results, either in legacy output format or
// via a printer.ObjectPrinter
func (i *imageScanCommand) printImageResult(imageResult *storage.Image) error {
	if i.printer == nil {
		return legacyPrintFormat(imageResult, i.format, i.env.InputOutput().Out(), i.env.Logger())
	}

	cveSummary := newCVESummaryForPrinting(imageResult.GetScan(), i.severities)

	if !i.standardizedFormat {
		printCVESummary(i.image, cveSummary.Result.Summary, i.env.Logger())
	}

	if err := i.printer.Print(cveSummary, i.env.ColorWriter()); err != nil {
		return errors.Wrap(err, "could not print CVE summary")
	}

	if !i.standardizedFormat {
		printCVEWarning(cveSummary.CountVulnerabilities(), cveSummary.CountComponents(), i.env.Logger())
	}

	if cveCount := cveSummary.CountVulnerabilities(); i.failOnFinding && cveCount > 0 {
		return newErrVulnerabilityFound(cveCount)
	}
	return nil
}

// print summary of amount of CVEs found
func printCVESummary(image string, cveSummary map[string]int, out logger.Logger) {
	out.PrintfLn("Scan results for image: %s", image)
	out.PrintfLn("(%s: %d, %s: %d, %s: %d, %s: %d, %s: %d, %s: %d)\n",
		totalComponentsMapKey, cveSummary[totalComponentsMapKey],
		totalVulnerabilitiesMapKey, cveSummary[totalVulnerabilitiesMapKey],
		lowCVESeverity, cveSummary[lowCVESeverity.String()],
		moderateCVESeverity, cveSummary[moderateCVESeverity.String()],
		importantCVESeverity, cveSummary[importantCVESeverity.String()],
		criticalCVESeverity, cveSummary[criticalCVESeverity.String()])
}

// print warning with amount of CVEs found in components
func printCVEWarning(numOfVulns int, numOfComponents int, out logger.Logger) {
	if numOfVulns != 0 {
		out.WarnfLn("A total of %d unique vulnerabilities were found in %d components",
			numOfVulns, numOfComponents)
	}
}

// TODO(ROX-8303): remove this once we have fully deprecated the legacy output format
// print CVE scan result in legacy output format
func legacyPrintFormat(imageResult *storage.Image, format string, out io.Writer, logger logger.Logger) error {
	switch format {
	case "csv":
		return PrintCSV(imageResult, out)
	default:
		jsonResult, err := jsonutil.MarshalToString(imageResult)
		if err != nil {
			return errors.Wrap(err, "could not marshal image result")
		}

		logger.PrintfLn(jsonResult)
	}
	return nil
}
