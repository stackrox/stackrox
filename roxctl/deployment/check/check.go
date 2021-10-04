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
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/report"
	"github.com/stackrox/rox/roxctl/common/util"
)

var (
	log = logging.LoggerForModule()
)

// Command checks the deployment against deploy time system policies
func Command() *cobra.Command {
	var (
		file               string
		json               bool
		retryDelay         int
		retryCount         int
		policyCategories   []string
		printAllViolations bool
	)
	c := &cobra.Command{
		Use: "check",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			return checkDeploymentWithRetry(file, json, flags.Timeout(c), retryDelay, retryCount, policyCategories, printAllViolations)
		}),
	}

	c.Flags().StringVarP(&file, "file", "f", "", "yaml file to send to Central to evaluate policies against")
	c.Flags().BoolVar(&json, "json", false, "output policy results as json.")
	c.Flags().IntVarP(&retryDelay, "retry-delay", "d", 3, "set time to wait between retries in seconds")
	c.Flags().IntVarP(&retryCount, "retries", "r", 0, "Number of retries before exiting as error")
	c.Flags().StringSliceVarP(&policyCategories, "categories", "c", nil, "optional comma separated list of policy categories to run.  Defaults to all policy categories.")
	c.Flags().BoolVar(&printAllViolations, "print-all-violations", false, "whether to print all violations per alert or truncate violations for readability")
	utils.Should(c.MarkFlagRequired("file"))
	return c
}

func checkDeploymentWithRetry(file string, json bool, timeout time.Duration, retryDelay int, retryCount int, policyCategories []string, printAllViolations bool) error {
	err := retry.WithRetry(func() error {
		return checkDeployment(file, json, timeout, policyCategories, printAllViolations)
	},
		retry.Tries(retryCount+1),
		retry.OnFailedAttempts(func(err error) {
			fmt.Fprintf(os.Stderr, "Scanning image failed: %v. Retrying after %v seconds\n", err, retryDelay)
			time.Sleep(time.Duration(retryDelay) * time.Second)
		}))
	if err != nil {
		return err
	}
	return nil
}

func checkDeployment(file string, json bool, timeout time.Duration, policyCategories []string, printAllViolations bool) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	// Get the violated policies for the input data.
	alerts, err := getAlerts(string(data), timeout, policyCategories)
	if err != nil {
		return err
	}

	// If json mode was given, print results (as json) and immediately return.
	if json {
		return report.JSON(os.Stdout, alerts)
	}

	// Print results in human readable mode.
	if err = report.Pretty(os.Stdout, alerts, storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT, "Deployment", printAllViolations); err != nil {
		return err
	}

	// Check if any of the violated policies have an enforcement action that
	// fails the CI build.
	for _, alert := range alerts {
		if report.EnforcementFailedBuild(storage.EnforcementAction_SCALE_TO_ZERO_ENFORCEMENT)(alert.GetPolicy()) {
			return errors.New("Violated a policy with Deploy Time enforcement set")
		}
	}
	return nil
}

// Get the alerts for the command line inputs.
func getAlerts(yaml string, timeout time.Duration, policyCategories []string) ([]*storage.Alert, error) {
	// Create the connection to the central detection service.
	conn, err := common.GetGRPCConnection()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = conn.Close()
	}()
	service := v1.NewDetectionServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// Call detection and return the returned alerts.
	response, err := service.DetectDeployTimeFromYAML(ctx, &v1.DeployYAMLDetectionRequest{
		Yaml:             yaml,
		PolicyCategories: policyCategories,
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
