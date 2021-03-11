package check

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/retry"
	pkgUtils "github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/report"
	"github.com/stackrox/rox/roxctl/common/util"
)

// Command checks the image against image build lifecycle policies
func Command() *cobra.Command {
	const (
		jsonFlagName     = "json"
		jsonFailFlagName = "json-fail-on-policy-violations"
	)
	var (
		image                  string
		json                   bool
		failViolationsWithJSON bool
		retryDelay             int
		retryCount             int
		sendNotifications      bool
		policyCategories       []string
		printAllViolations     bool
	)
	c := &cobra.Command{
		Use: "check",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			return checkImageWithRetry(image, json, failViolationsWithJSON, sendNotifications, flags.Timeout(c), retryDelay, retryCount, policyCategories, printAllViolations)
		}),
		PreRun: func(c *cobra.Command, args []string) {
			jsonFlag := c.Flag(jsonFlagName)
			jsonFailFlag := c.Flag(jsonFailFlagName)
			if jsonFlag.Changed && !jsonFailFlag.Changed {
				fmt.Fprintf(os.Stderr, "Warning: the default value for --%s will change in a future release, you might want to specify it explicitly now.\n", jsonFailFlag.Name)
			} else if !jsonFlag.Changed && jsonFailFlag.Changed {
				fmt.Fprintf(os.Stderr, "Note: --%s has no effect when --%s is not specified.\n", jsonFailFlag.Name, jsonFlag.Name)
			}
		},
	}

	c.Flags().StringVarP(&image, "image", "i", "", "image name and reference. (e.g. nginx:latest or nginx@sha256:...)")
	pkgUtils.Must(c.MarkFlagRequired("image"))

	c.Flags().BoolVar(&json, jsonFlagName, false, "Output policy results as JSON")
	// TODO(ROX-6573): when changing the default in a future release, also remove the warning in PreRun.
	c.Flags().BoolVar(&failViolationsWithJSON, jsonFailFlagName, false,
		"Whether policy violations should cause the command to exit non-zero in JSON output mode too. "+
			"This flag only has effect when --json is also specified. "+
			"The default for this flag will change in a future release")
	c.Flags().IntVarP(&retryDelay, "retry-delay", "d", 3, "set time to wait between retries in seconds.")
	c.Flags().IntVarP(&retryCount, "retries", "r", 0, "number of retries before exiting as error.")
	c.Flags().BoolVar(&sendNotifications, "send-notifications", false,
		"whether to send notifications for violations (notifications will be sent to the notifiers "+
			"configured in each violated policy).")
	c.Flags().StringSliceVarP(&policyCategories, "categories", "c", nil, "optional comma separated list of policy categories to run.  Defaults to all policy categories.")
	c.Flags().BoolVar(&printAllViolations, "print-all-violations", false, "whether to print all violations per alert or truncate violations for readability")
	return c
}

func checkImageWithRetry(image string, json bool, failViolationsWithJSON bool, sendNotifications bool, timeout time.Duration, retryDelay int, retryCount int, policyCategories []string, printAllViolations bool) error {
	err := retry.WithRetry(func() error {
		return checkImage(image, json, failViolationsWithJSON, sendNotifications, timeout, policyCategories, printAllViolations)
	},
		retry.Tries(retryCount+1),
		retry.OnFailedAttempts(func(err error) {
			fmt.Fprintf(os.Stderr, "Checking image failed: %v. Retrying after %v seconds\n", err, retryDelay)
			time.Sleep(time.Duration(retryDelay) * time.Second)
		}))
	if err != nil {
		return err
	}
	return nil
}

func checkImage(image string, json bool, failViolationsWithJSON bool, sendNotifications bool, timeout time.Duration, policyCategories []string, printAllViolations bool) error {
	// Get the violated policies for the input data.
	req, err := buildRequest(image, sendNotifications, policyCategories)
	if err != nil {
		return err
	}
	alerts, err := sendRequestAndGetAlerts(req, timeout)
	if err != nil {
		return err
	}

	return reportCheckResults(image, json, failViolationsWithJSON, alerts, printAllViolations)
}

func reportCheckResults(image string, json bool, failViolationsWithJSON bool, alerts []*storage.Alert, printAllViolations bool) error {
	// If json mode was given, print results (as json) and either immediately return or check policy,
	// depending on a flag.
	if json {
		err := report.JSON(os.Stdout, alerts)
		if err != nil {
			return err
		}
		if failViolationsWithJSON {
			return checkPolicyFailures(alerts)
		}
		return nil
	}

	// Print results in human readable mode.
	if err := report.PrettyWithResourceName(os.Stdout, alerts, storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT, "Image", image, printAllViolations); err != nil {
		return err
	}

	return checkPolicyFailures(alerts)
}

func checkPolicyFailures(alerts []*storage.Alert) error {
	// Check if any of the violated policies have an enforcement action that
	// fails the CI build.
	for _, alert := range alerts {
		if report.EnforcementFailedBuild(storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT)(alert.GetPolicy()) {
			return errors.New("Violated a policy with CI enforcement set")
		}
	}
	return nil
}

// Get the alerts for the command line inputs.
func sendRequestAndGetAlerts(req *v1.BuildDetectionRequest, timeout time.Duration) ([]*storage.Alert, error) {
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
	response, err := service.DetectBuildTime(ctx, req)
	if err != nil {
		return nil, err
	}
	return response.GetAlerts(), nil
}

// Use inputs to generate an image name for request.
func buildRequest(image string, sendNotifications bool, policyCategories []string) (*v1.BuildDetectionRequest, error) {
	img, err := utils.GenerateImageFromString(image)
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse image '%s'", image)
	}
	return &v1.BuildDetectionRequest{
		Resource:          &v1.BuildDetectionRequest_Image{Image: img},
		SendNotifications: sendNotifications,
		PolicyCategories:  policyCategories,
	}, nil
}
