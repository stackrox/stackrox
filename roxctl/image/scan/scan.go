package scan

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/retry"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
	"google.golang.org/grpc"
)

// Command checks the image against image build lifecycle policies
func Command(cliEnvironment environment.Environment) *cobra.Command {
	imageScanCmd := &imageScanCommand{env: cliEnvironment}

	c := &cobra.Command{
		Use: "scan",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			if err := imageScanCmd.Construct(nil, c); err != nil {
				return err
			}

			if err := imageScanCmd.Validate(); err != nil {
				return err
			}

			return imageScanCmd.Scan()
		}),
	}

	c.Flags().StringVarP(&imageScanCmd.image, "image", "i", "", "image name and reference. (e.g. nginx:latest or nginx@sha256:...)")
	c.Flags().BoolVarP(&imageScanCmd.force, "force", "f", false, "the --force flag ignores Central's cache for the scan and forces a fresh re-pull from Scanner")
	c.Flags().BoolVarP(&imageScanCmd.includeSnoozed, "include-snoozed", "a", true, "the --include-snoozed flag returns both snoozed and unsnoozed CVEs if set to false")
	c.Flags().StringVarP(&imageScanCmd.format, "format", "", "json", "format of the output. Choose output format from json, csv, and pretty.")
	c.Flags().IntVarP(&imageScanCmd.retryDelay, "retry-delay", "d", 3, "set time to wait between retries in seconds")
	c.Flags().IntVarP(&imageScanCmd.retryCount, "retries", "r", 0, "Number of retries before exiting as error")
	return c
}

// imageScanCommand holds all configurations and metadata to execute an image scan
type imageScanCommand struct {
	image          string
	force          bool
	includeSnoozed bool
	format         string
	retryDelay     int
	retryCount     int
	timeout        time.Duration
	env            environment.Environment
	centralImgSvc  centralImageService
}

// centralImageService abstracts away the gRPC call to the image service within central
// this is especially useful for testing
type centralImageService interface {
	ScanImage(ctx context.Context, in *v1.ScanImageRequest, opts ...grpc.CallOption) (*storage.Image, error)
}

// Construct will enhance the struct with other values coming either from os.Args, other, global flags or environment variables
func (i *imageScanCommand) Construct(args []string, cmd *cobra.Command) error {
	i.timeout = flags.Timeout(cmd)

	return nil
}

// Validate will validate the injected values and check whether it's possible to execute the operation with the
// provided values
func (i *imageScanCommand) Validate() error {
	if i.image == "" {
		return errors.New("missing image name. please specify an image name via either --image or -i")
	}
	// TODO(dhaus): When creating the abstraction for the printer, this needs to be replaced.
	// 				The printer abstraction should be responsible to validate the format and
	//				select the correct printer accordingly
	if i.format != "json" && i.format != "csv" && i.format != "pretty" {
		return fmt.Errorf("invalid output format given: %q. You can only specify json, csv or pretty", i.format)
	}
	return nil
}

// Scan will execute the image scan with retry functionality
func (i *imageScanCommand) Scan() error {
	err := retry.WithRetry(func() error {
		return i.scanImage()
	},
		retry.Tries(i.retryCount+1),
		retry.OnFailedAttempts(func(err error) {
			fmt.Fprintf(i.env.InputOutput().ErrOut, "Scanning image failed: %v. Retrying after %v seconds\n", err, i.retryDelay)
			time.Sleep(time.Duration(i.retryDelay) * time.Second)
		}),
	)
	if err != nil {
		return err
	}
	return nil
}

func (i *imageScanCommand) scanImage() error {
	imageResult, err := i.getImageResultFromService()

	if err != nil {
		return err
	}

	return i.printImageResult(imageResult)
}

func (i *imageScanCommand) printImageResult(imageResult *storage.Image) error {
	switch i.format {
	case "csv":
		return PrintCSV(imageResult)
	case "pretty":
		PrintPretty(imageResult)
	default:
		marshaller := &jsonpb.Marshaler{
			Indent: "  ",
		}
		jsonResult, err := marshaller.MarshalToString(imageResult)
		if err != nil {
			return err
		}

		fmt.Fprintln(i.env.InputOutput().Out, jsonResult)
	}
	return nil
}

func (i *imageScanCommand) getImageResultFromService() (*storage.Image, error) {
	conn, err := i.env.GRPCConnection()
	if err != nil {
		return nil, err
	}
	defer utils.IgnoreError(conn.Close)

	svc := v1.NewImageServiceClient(conn)
	i.centralImgSvc = svc

	ctx, cancel := context.WithTimeout(pkgCommon.Context(), i.timeout)
	defer cancel()

	return i.centralImgSvc.ScanImage(ctx, &v1.ScanImageRequest{
		ImageName:      i.image,
		Force:          i.force,
		IncludeSnoozed: i.includeSnoozed,
	})
}
