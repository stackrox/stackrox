package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/cmd/common"
	"github.com/stackrox/rox/cmd/roxdetect/report"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/version"
	"golang.org/x/net/context"
)

const (
	// This is set very high, because typically the scan will need to be triggered as the image will be new
	// This means we must let the scanners do their thing otherwise we will miss the scans
	timeout = 10 * time.Minute
)

func versionCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "version",
		Short: "Prints out the version of roxdetect.",
		Long:  "Prints out the version of roxdetect.",
		Run: func(*cobra.Command, []string) {
			fmt.Printf("%s\n", version.GetMainVersion())
		},
	}
	return c
}

// CheckCommand checks the image against image build lifecycle policies
func CheckCommand() *cobra.Command {
	var (
		central string
		image   string
		json    bool
	)

	c := &cobra.Command{
		Use:   "roxdetect",
		Short: "Check images for build time policy violations.",
		Long:  "Check images for build time policy violations.",
		RunE: func(*cobra.Command, []string) error {
			if image == "" {
				return fmt.Errorf("image name must be set")
			}
			return checkImage(central, image, json)
		},
	}
	c.AddCommand(versionCommand())

	c.Flags().StringVarP(&central, "central", "c", "localhost:8443", "host and port endpoint where Central is located.")
	c.Flags().StringVarP(&image, "image", "i", "", "image name and reference. (e.g. nginx:latest or nginx@sha256:...)")
	c.Flags().BoolVar(&json, "json", false, "output policy results as json.")
	return c
}

func checkImage(central, image string, json bool) error {
	// Get the violated policies for the input data.
	violatedPolicies, err := getViolatedPolicies(central, image)
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
func getViolatedPolicies(central, image string) ([]*v1.Policy, error) {
	alerts, err := getAlerts(central, image)
	if err != nil {
		return nil, err
	}
	var policies []*v1.Policy
	for _, alert := range alerts {
		policies = append(policies, alert.GetPolicy())
	}
	return policies, nil
}

// Get the alerts for the command line inputs.
func getAlerts(central, imageStr string) ([]*v1.Alert, error) {
	// Attempt to construct the request first since it is the cheapest op.
	image, err := buildRequest(imageStr)
	if err != nil {
		return nil, err
	}
	// Create the connection to the central detection service.
	conn, err := common.GetGRPCConnection(central)
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
func buildRequest(image string) (*v1.Image, error) {
	img, err := utils.GenerateImageFromStringWithError(image)
	if err != nil {
		return nil, fmt.Errorf("could not parse image '%s': %s", image, err)
	}
	return img, nil
}

func main() {
	if err := CheckCommand().Execute(); err != nil {
		fmt.Printf("Error running roxdetect %v\n", err)
		os.Exit(1)
	}
}
