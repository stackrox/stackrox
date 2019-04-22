package check

import (
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/report"
	"golang.org/x/net/context"
)

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
		RunE: func(c *cobra.Command, _ []string) error {
			if image == "" {
				return fmt.Errorf("--image must be set")
			}
			return checkImage(image, json, flags.Timeout(c))
		},
	}

	c.Flags().StringVarP(&image, "image", "i", "", "image name and reference. (e.g. nginx:latest or nginx@sha256:...)")
	c.Flags().BoolVar(&json, "json", false, "output policy results as json.")
	return c
}

func checkImage(image string, json bool, timeout time.Duration) error {
	// Get the violated policies for the input data.
	alerts, err := getAlerts(image, timeout)
	if err != nil {
		return err
	}

	// If json mode was given, print results (as json) and immediately return.
	if json {
		return report.JSON(os.Stdout, alerts)
	}

	// Print results in human readable mode.
	if err = report.PrettyWithResourceName(os.Stdout, alerts, storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT, "Image", image); err != nil {
		return err
	}

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
func getAlerts(imageStr string, timeout time.Duration) ([]*storage.Alert, error) {
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
	defer func() {
		_ = conn.Close()
	}()
	service := v1.NewDetectionServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// Call detection and return the returned alerts.
	response, err := service.DetectBuildTime(ctx, &v1.BuildDetectionRequest{Resource: &v1.BuildDetectionRequest_Image{Image: image}})
	if err != nil {
		return nil, err
	}
	return response.GetAlerts(), nil
}

// Use inputs to generate an image name for request.
func buildRequest(image string) (*storage.Image, error) {
	img, err := utils.GenerateImageFromString(image)
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse image '%s'", image)
	}
	return img, nil
}
