package check

import (
	"context"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/gjson"
	"github.com/stackrox/rox/pkg/printers"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/logger"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stackrox/rox/roxctl/common/report"
	"github.com/stackrox/rox/roxctl/summaries/policy"
)

const (
	// Default JSON path expression which retrieves the data from the policy.PolicyJSONResult
	defaultDeploymentCheckJSONPathExpression = "{" +
		"results.#.violatedPolicies.#.name," +
		"results.#.violatedPolicies.#.severity," +
		"results.#.violatedPolicies.#.failingCheck.@boolReplace:{\"true\":\"X\",\"false\":\"-\"}," +
		"results.#.metadata.additionalInfo.name," +
		"results.#.violatedPolicies.#.description," +
		"results.#.violatedPolicies.#.violation.@list," +
		"results.#.violatedPolicies.#.remediation}"
)

var (
	// Default headers to use when printing tabular output
	defaultDeploymentCheckHeaders = []string{
		"POLICY", "SEVERITY", "BREAKS DEPLOY", "DEPLOYMENT", "DESCRIPTION", "VIOLATION", "REMEDIATION",
	}
	defaultJunitJSONPathExpressions = map[string]string{
		printers.JUnitTestCasesExpressionKey:            "results.#.violatedPolicies.#.name",
		printers.JUnitFailedTestCasesExpressionKey:      "results.#.violatedPolicies.#(failingCheck==~true)#.name",
		printers.JUnitSkippedTestCasesExpressionKey:     "results.#.violatedPolicies.#(failingCheck==~false)#.name",
		printers.JUnitFailedTestCaseErrMsgExpressionKey: "results.#.violatedPolicies.#(failingCheck==~true)#.violation.@list",
	}

	sarifJSONPathExpressions = map[string]string{
		printers.SarifRuleJSONPathExpressionKey: "results.#.violatedPolicies.#.name",
		printers.SarifHelpJSONPathExpressionKey: gjson.MultiPathExpression(
			"@text",
			gjson.Expression{
				Key:        "Policy",
				Expression: "results.#.violatedPolicies.#.name",
			},
			gjson.Expression{
				Key:        "Severity",
				Expression: "results.#.violatedPolicies.#.severity",
			},
			gjson.Expression{
				Key:        "Violations",
				Expression: "results.#.violatedPolicies.#.violation.@list",
			},
			gjson.Expression{
				Key:        "Remediation",
				Expression: "results.#.violatedPolicies.#.remediation",
			},
		),
		printers.SarifSeverityJSONPathExpressionKey: "results.#.violatedPolicies.#.severity",
	}

	// supported output formats with default values
	supportedObjectPrinters = []printer.CustomPrinterFactory{
		printer.NewTabularPrinterFactory(defaultDeploymentCheckHeaders, defaultDeploymentCheckJSONPathExpression),
		printer.NewJSONPrinterFactory(false, false),
		printer.NewJUnitPrinterFactory("deployment-check", defaultJunitJSONPathExpressions),
	}
)

// Command checks the deployment against deploy time system policies
func Command(cliEnvironment environment.Environment) *cobra.Command {
	deploymentCheckCmd := &deploymentCheckCommand{env: cliEnvironment}

	objectPrinterFactory, err := printer.NewObjectPrinterFactory("table", append(supportedObjectPrinters,
		printer.NewSarifPrinterFactory(printers.SarifPolicyReport, sarifJSONPathExpressions, &deploymentCheckCmd.file))...)
	// this error should never occur, it would only occur if default values are invalid
	utils.Must(err)

	c := &cobra.Command{
		Use:   "check",
		Short: "Check deployments for deploy time policy violations.",
		Long:  "Check deployments for deploy time policy violations, and exit with an non-zero code if at least one of the violated policies has deploy time enforcement turned on.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := deploymentCheckCmd.Construct(args, cmd, objectPrinterFactory); err != nil {
				return err
			}
			if err := deploymentCheckCmd.Validate(); err != nil {
				return err
			}

			return deploymentCheckCmd.Check()
		},
	}

	// Add all printer related flags
	objectPrinterFactory.AddFlags(c)

	c.Flags().StringSliceVarP(&deploymentCheckCmd.files, "files", "f", []string{}, "yaml file to send to Central to evaluate policies against")
	c.Flags().StringVarP(&deploymentCheckCmd.file, "file", "f", "", "yaml file to send to Central to evaluate policies against")
	c.Flags().BoolVar(&deploymentCheckCmd.json, "json", false, "output policy results as json.")
	c.Flags().IntVarP(&deploymentCheckCmd.retryDelay, "retry-delay", "d", 3, "set time to wait between retries in seconds")
	c.Flags().IntVarP(&deploymentCheckCmd.retryCount, "retries", "r", 3, "Number of retries before exiting as error")
	c.Flags().StringSliceVarP(&deploymentCheckCmd.policyCategories, "categories", "c", nil, "optional comma separated list of policy categories to run.  Defaults to all policy categories.")
	c.Flags().BoolVar(&deploymentCheckCmd.printAllViolations, "print-all-violations", false, "whether to print all violations per alert or truncate violations for readability")
	c.Flags().BoolVar(&deploymentCheckCmd.force, "force", false, "bypass Central's cache for images and force a new pull from the Scanner")
	utils.Must(c.MarkFlagRequired("file"))
	c.Flags().StringVar(&deploymentCheckCmd.cluster, "cluster", "", "cluster name or ID to use as context for evaluation")

	// mark legacy output format specific flags as deprecated
	utils.Must(c.Flags().MarkDeprecated("json", "use the new output format which also offers JSON. NOTE: "+
		"The new output format's structure has changed in a non-backward compatible way."))
	utils.Must(c.Flags().MarkDeprecated("print-all-violations", "use the new output format where all "+
		"violations are printed by default. This flag will only be relevant in combination with the --json flag"))

	return c
}

type deploymentCheckCommand struct {
	// properties bound to cobra flags
	files              []string
	file               string
	json               bool
	retryDelay         int
	retryCount         int
	policyCategories   []string
	printAllViolations bool
	timeout            time.Duration
	force              bool
	cluster            string

	// injected or constructed values by Construct
	env                environment.Environment
	printer            printer.ObjectPrinter
	standardizedFormat bool
}

func (d *deploymentCheckCommand) Construct(_ []string, cmd *cobra.Command, f *printer.ObjectPrinterFactory) error {
	d.timeout = flags.Timeout(cmd)

	// Only create a printer if legacy json output format is not used
	// TODO(ROX-8303): Remove this once we have fully deprecated the old output format
	if !d.json {
		p, err := f.CreatePrinter()
		if err != nil {
			return errors.Wrap(err, "could not create printer for deployment check results")
		}
		d.printer = p
		d.standardizedFormat = f.IsStandardizedFormat()
	}

	return nil
}

func (d *deploymentCheckCommand) Validate() error {
	for _, file := range d.files {
		if _, err := os.Open(file); err != nil {
			return common.ErrInvalidCommandOption.CausedBy(err)
		}
	}

	if _, err := os.Open(d.file); err != nil {
		return common.ErrInvalidCommandOption.CausedBy(err)
	}

	return nil
}

func (d *deploymentCheckCommand) Check() error {
	err := retry.WithRetry(func() error {
		return d.checkDeployment()
	},
		retry.Tries(d.retryCount+1),
		retry.OnlyRetryableErrors(),
		retry.OnFailedAttempts(func(err error) {
			d.env.Logger().ErrfLn("Checking deployment failed: %v. Retrying after %d seconds...",
				err, d.retryDelay)
			time.Sleep(time.Duration(d.retryDelay) * time.Second)
		}))
	if err != nil {
		return errors.Wrapf(err, "checking deployment failed after %d retries", d.retryCount)
	}
	return nil
}

func (d *deploymentCheckCommand) checkDeployment() error {
	var deploymentFileContents []byte
	if d.file != "" {
		var err error
		deploymentFileContents, err = os.ReadFile(d.file)
		if err != nil {
			return errors.Wrapf(err, "could not read deployment file: %q", d.file)
		}
	} else {
		for _, file := range d.files {
			fileContents, err := os.ReadFile(file)
			if err != nil {
				return errors.Wrapf(err, "could not read deployment file: %q", file)
			}
			if len(deploymentFileContents) > 0 {
				deploymentFileContents = append(deploymentFileContents, "\n---\n"...)
			}
			deploymentFileContents = append(deploymentFileContents, fileContents...)
		}
	}

	alerts, ignoredObjRefs, err := d.getAlertsAndIgnoredObjectRefs(string(deploymentFileContents))
	if err != nil {
		return errors.Wrap(retry.MakeRetryable(err), "retrieving alerts from central")
	}

	return d.printResults(alerts, ignoredObjRefs)
}

func (d *deploymentCheckCommand) getAlertsAndIgnoredObjectRefs(deploymentYaml string) ([]*storage.Alert, []string, error) {
	conn, err := d.env.GRPCConnection()
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not establish gRPC connection to central")
	}
	defer utils.IgnoreError(conn.Close)

	svc := v1.NewDetectionServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()

	response, err := svc.DetectDeployTimeFromYAML(ctx, &v1.DeployYAMLDetectionRequest{
		Yaml:             deploymentYaml,
		PolicyCategories: d.policyCategories,
		Cluster:          d.cluster,
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, "could not check deploy-time alerts")
	}

	var alerts []*storage.Alert
	for _, r := range response.GetRuns() {
		alerts = append(alerts, r.GetAlerts()...)
	}
	return alerts, response.GetIgnoredObjectRefs(), nil
}

func (d *deploymentCheckCommand) printResults(alerts []*storage.Alert, ignoredObjectRefs []string) error {
	// Print all ignored objects whose schema was not registered, i.e. CRDs. We don't need to take standardizedFormat
	// into account since we will print to os.StdErr by default. We shall do this at the beginning, since we also
	// want this to be visible to the old output format.
	for _, ignoredObjRef := range ignoredObjectRefs {
		d.env.Logger().InfofLn("Ignored object %q as its schema was not registered.", ignoredObjRef)
	}

	if d.json {
		return errors.Wrap(report.JSON(d.env.InputOutput().Out(), alerts), "could not print JSON report")
	}

	// TODO: Need to refactor this to include additional summary info for non-standardized formats
	// as well as multiple results for each deployment
	policySummary := policy.NewPolicySummaryForPrinting(alerts, storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT)

	if !d.standardizedFormat {
		printDeploymentPolicySummary(policySummary.Summary, d.env.Logger(), policySummary.GetResultNames()...)
	}

	if err := d.printer.Print(policySummary, d.env.ColorWriter()); err != nil {
		return errors.Wrap(err, "could not print policy summary")
	}

	amountBreakingPolicies := policySummary.GetTotalAmountOfBreakingPolicies()

	if !d.standardizedFormat {
		printAdditionalWarnsAndErrs(policySummary.Summary[policy.TotalPolicyAmountKey], amountBreakingPolicies,
			policySummary.Results, d.env.Logger())
	}

	if amountBreakingPolicies != 0 {
		return errors.Wrap(policy.NewErrBreakingPolicies(amountBreakingPolicies), "breaking policies found")
	}
	return nil
}

func printDeploymentPolicySummary(numOfPolicyViolations map[string]int, out logger.Logger, deployments ...string) {
	out.PrintfLn("Policy check results for deployments: %v", deployments)
	out.PrintfLn("(%s: %d, %s: %d, %s: %d, %s: %d, %s: %d)\n",
		policy.TotalPolicyAmountKey, numOfPolicyViolations[policy.TotalPolicyAmountKey],
		policy.LowSeverity, numOfPolicyViolations[policy.LowSeverity.String()],
		policy.MediumSeverity, numOfPolicyViolations[policy.MediumSeverity.String()],
		policy.HighSeverity, numOfPolicyViolations[policy.HighSeverity.String()],
		policy.CriticalSeverity, numOfPolicyViolations[policy.CriticalSeverity.String()])
}

func printAdditionalWarnsAndErrs(amountViolatedPolicies, amountBreakingPolicies int, results []policy.EntityResult,
	out logger.Logger,
) {
	if amountViolatedPolicies == 0 {
		return
	}

	out.WarnfLn("A total of %d policies have been violated", amountViolatedPolicies)

	if amountBreakingPolicies == 0 {
		return
	}

	out.ErrfLn("%s", policy.NewErrBreakingPolicies(amountBreakingPolicies))

	for _, res := range results {
		for _, breakingPolicy := range res.GetBreakingPolicies() {
			out.ErrfLn("Policy %q within Deployment %q - Possible remediation: %q",
				breakingPolicy.Name, res.Metadata.GetName(), breakingPolicy.Remediation)
		}
	}
}
