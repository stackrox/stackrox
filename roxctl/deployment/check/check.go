package check

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/report"
)

var (
	log = logging.LoggerForModule()
)

// Command checks the deployment against deploy time system policies
func Command(cliEnvironment environment.Environment) *cobra.Command {
	deploymentCheckCmd := &deploymentCheckCommand{env: cliEnvironment}

	c := &cobra.Command{
		Use:  "check",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := deploymentCheckCmd.Construct(args, cmd); err != nil {
				return err
			}
			if err := deploymentCheckCmd.Validate(); err != nil {
				return err
			}

			return deploymentCheckCmd.Check()
		},
	}

	c.Flags().StringVarP(&deploymentCheckCmd.file, "file", "f", "", "yaml file to send to Central to evaluate policies against")
	c.Flags().BoolVar(&deploymentCheckCmd.json, "json", false, "output policy results as json.")
	c.Flags().IntVarP(&deploymentCheckCmd.retryDelay, "retry-delay", "d", 3, "set time to wait between retries in seconds")
	c.Flags().IntVarP(&deploymentCheckCmd.retryCount, "retries", "r", 0, "Number of retries before exiting as error")
	c.Flags().StringSliceVarP(&deploymentCheckCmd.policyCategories, "categories", "c", nil, "optional comma separated list of policy categories to run.  Defaults to all policy categories.")
	c.Flags().BoolVar(&deploymentCheckCmd.printAllViolations, "print-all-violations", false, "whether to print all violations per alert or truncate violations for readability")
	utils.Should(c.MarkFlagRequired("file"))
	return c
}

type deploymentCheckCommand struct {
	// properties bound to cobra flags
	file               string
	json               bool
	retryDelay         int
	retryCount         int
	policyCategories   []string
	printAllViolations bool
	timeout            time.Duration

	// injected or constructed values by Construct
	env environment.Environment
}

func (d *deploymentCheckCommand) Construct(args []string, cmd *cobra.Command) error {
	d.timeout = flags.Timeout(cmd)

	return nil
}

func (d *deploymentCheckCommand) Validate() error {
	return nil
}

func (d *deploymentCheckCommand) Check() error {
	err := retry.WithRetry(func() error {
		return d.checkDeployment()
	},
		retry.Tries(d.retryCount+1),
		retry.OnFailedAttempts(func(err error) {
			fmt.Fprintf(d.env.InputOutput().ErrOut, "Scanning image failed: %v. Retrying after %d seconds\n",
				err, d.retryDelay)
			time.Sleep(time.Duration(d.retryDelay) * time.Second)
		}))
	if err != nil {
		return err
	}
	return nil
}

func (d *deploymentCheckCommand) checkDeployment() error {
	deploymentFileContents, err := os.ReadFile(d.file)
	if err != nil {
		return err
	}

	alerts, err := d.getAlerts(string(deploymentFileContents))
	if err != nil {
		return err
	}

	return d.printResults(alerts)
}

func (d *deploymentCheckCommand) getAlerts(deploymentYaml string) ([]*storage.Alert, error) {
	conn, err := d.env.GRPCConnection()
	if err != nil {
		return nil, err
	}
	defer utils.IgnoreError(conn.Close)

	svc := v1.NewDetectionServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()

	response, err := svc.DetectDeployTimeFromYAML(ctx, &v1.DeployYAMLDetectionRequest{
		Yaml:             deploymentYaml,
		PolicyCategories: d.policyCategories,
	})
	if err != nil {
		return nil, err
	}

	var alerts []*storage.Alert
	for _, r := range response.GetRuns() {
		alerts = append(alerts, r.GetAlerts()...)
	}
	return alerts, nil
}

func (d *deploymentCheckCommand) printResults(alerts []*storage.Alert) error {
	if d.json {
		return report.JSON(d.env.InputOutput().Out, alerts)
	}

	if err := report.Pretty(d.env.InputOutput().Out, alerts, storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT,
		"Deployment", d.printAllViolations); err != nil {
		return err
	}

	return getFailingPolicies(alerts)
}

func getFailingPolicies(alerts []*storage.Alert) error {
	for _, alert := range alerts {
		if report.EnforcementFailedBuild(storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT)(alert.GetPolicy()) {
			return errors.New("Violated a policy with Deploy Time enforcement set")
		}
	}
	return nil
}
