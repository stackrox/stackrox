package scan

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

// Command checks the image against image build lifecycle policies
func Command() *cobra.Command {
	var (
		image          string
		force          bool
		includeSnoozed bool
		format         string
		retryDelay     int
		retryCount     int
	)
	c := &cobra.Command{
		Use: "scan",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			if image == "" {
				return errors.New("--image must be set")
			}
			if format != "json" && format != "csv" && format != "pretty" {
				return errors.New("Unexpected format: " + format)

			}
			return scanImageWithRetry(image, force, includeSnoozed, format, flags.Timeout(c), retryDelay, retryCount)
		}),
	}

	c.Flags().StringVarP(&image, "image", "i", "", "image name and reference. (e.g. nginx:latest or nginx@sha256:...)")
	c.Flags().BoolVarP(&force, "force", "f", false, "the --force flag ignores Central's cache for the scan and forces a fresh re-pull from Scanner")
	c.Flags().BoolVarP(&includeSnoozed, "include-snoozed", "a", true, "the --include-snoozed flag returns both snoozed and unsnoozed CVEs if set to false")
	c.Flags().StringVarP(&format, "format", "", "json", "format of the output. Choose output format from json, csv, and pretty.")
	c.Flags().IntVarP(&retryDelay, "retry-delay", "d", 3, "set time to wait between retries in seconds")
	c.Flags().IntVarP(&retryCount, "retries", "r", 0, "Number of retries before exiting as error")
	return c
}

func scanImageWithRetry(image string, force, includeSnoozed bool, format string, timeout time.Duration, retryDelay int, retryCount int) error {
	err := retry.WithRetry(func() error {
		return scanImage(image, force, includeSnoozed, format, timeout)
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

func scanImage(image string, force, includeSnoozed bool, format string, timeout time.Duration) error {
	// Create the connection to the central detection service.
	conn, err := common.GetGRPCConnection()
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	service := v1.NewImageServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	imageResult, err := service.ScanImage(ctx, &v1.ScanImageRequest{
		ImageName:      image,
		Force:          force,
		IncludeSnoozed: includeSnoozed,
	})
	if err != nil {
		return err
	}
	switch format {
	case "csv":
		return PrintCSV(imageResult)
	case "pretty":
		PrintPretty(imageResult)
	default:
		// In json format.
		marshaler := &jsonpb.Marshaler{
			Indent: "  ",
		}
		jsonResult, err := marshaler.MarshalToString(imageResult)
		if err != nil {
			return err
		}
		fmt.Println(jsonResult)
	}
	return nil
}
