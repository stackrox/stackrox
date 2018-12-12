package check

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/image/check/report"
	"golang.org/x/net/context"
)

// This is set very high, because typically the scan will need to be triggered as the image will be new
// This means we must let the scanners do their thing otherwise we will miss the scans
// TODO(cgorman) We need a flag currently that says --wait-for-image timeout or something like that because Clair does scanning inline
// but other scanners do not
const timeout = 10 * time.Minute

// Command checks the image against image build lifecycle policies
func Command() *cobra.Command {
	var (
		image string
		json  bool
	)
	c := &cobra.Command{
		Use:   "check",
		Short: "Check images for build time policy violations.",
		Long:  "Check images for build time policy violations.",
		RunE: func(*cobra.Command, []string) error {
			if image == "" {
				return fmt.Errorf("--image must be set")
			}
			return checkImage(image, json)
		},
	}

	c.Flags().StringVarP(&image, "image", "i", "", "image name and reference. (e.g. nginx:latest or nginx@sha256:...)")
	c.Flags().BoolVar(&json, "json", false, "output policy results as json.")
	return c
}

func checkImage(image string, json bool) error {
	// Get the violated policies for the input data.
	violatedPolicies, err := getViolatedPolicies(image)
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
func getViolatedPolicies(image string) ([]*storage.Policy, error) {
	alerts, err := getAlerts(image)
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
func getAlerts(imageStr string) ([]*v1.Alert, error) {
	// Attempt to construct the request first since it is the cheapest op.
	image, err := buildRequest(imageStr)
	if err != nil {
		return nil, err
	}

	// Create the connection to the central detection service.
	conn, err := common.GetGRPCConnection()
	if err != nil {
		return nil, err
	}
	service := v1.NewDetectionServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// Call detection and return the returned alerts.
	response, err := service.DetectBuildTime(ctx, image)
	if err != nil {
		return nil, err
	}
	return response.GetAlerts(), nil
}

// Use inputs to generate an image name for request.
func buildRequest(image string) (*storage.Image, error) {
	img, err := utils.GenerateImageFromStringWithError(image)
	if err != nil {
		return nil, fmt.Errorf("could not parse image '%s': %s", image, err)
	}
	return img, nil
}
