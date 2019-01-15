package check

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/image/check/report"
	"golang.org/x/net/context"
)

const timeout = 1 * time.Minute

var log = logging.LoggerForModule()

// Command checks the image against image build lifecycle policies
func Command() *cobra.Command {
	var (
		file string
		json bool
	)
	c := &cobra.Command{
		Use:   "check",
		Short: "Check images for build time policy violations.",
		Long:  "Check images for build time policy violations.",
		RunE: func(*cobra.Command, []string) error {
			if file == "" {
				return fmt.Errorf("--file must be set")
			}
			return checkDeployment(file, json)
		},
	}

	c.Flags().StringVarP(&file, "file", "f", "", "file to send to Central to evaluate policies against")
	c.Flags().BoolVar(&json, "json", false, "output policy results as json.")
	return c
}

func checkDeployment(file string, json bool) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	// Get the violated policies for the input data.
	violatedPolicies, err := getViolatedPolicies(string(data))
	if err != nil {
		return err
	}

	// If json mode was given, print results (as json) and immediately return.
	if json {
		return report.JSON(os.Stdout, violatedPolicies)
	}

	// Print results in human readable mode.
	if err = report.Pretty(os.Stdout, violatedPolicies); err != nil {
		return err
	}

	// Check if any of the violated policies have an enforcement action that
	// fails the CI build.
	for _, policy := range violatedPolicies {
		if report.EnforcementFailedBuild(policy) {
			return errors.New("Violated a policy with CI enforcement set")
		}
	}
	return nil
}

// Fetch the alerts for the inputs and convert them to a list of Policies that are violated.
func getViolatedPolicies(yaml string) ([]*storage.Policy, error) {
	alerts, err := getAlerts(yaml)
	if err != nil {
		return nil, err
	}

	var policies []*storage.Policy
	for _, alert := range alerts {
		policies = append(policies, alert.GetPolicy())
	}
	return policies, nil
}

// Get the alerts for the command line inputs.
func getAlerts(yaml string) ([]*storage.Alert, error) {
	// Create the connection to the central detection service.
	conn, err := common.GetGRPCConnection()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	service := v1.NewDetectionServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// Call detection and return the returned alerts.
	response, err := service.DetectDeployTimeFromYAML(ctx, &v1.DeployYAMLDetectionRequest{Yaml: yaml})
	if err != nil {
		return nil, err
	}

	log.Infof("Runs: %+v", response.GetRuns())

	return []*storage.Alert{}, nil
}
