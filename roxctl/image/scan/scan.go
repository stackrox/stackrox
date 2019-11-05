package scan

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
)

// Command checks the image against image build lifecycle policies
func Command() *cobra.Command {
	var (
		image string
	)
	c := &cobra.Command{
		Use:   "scan",
		Short: "Scan an image and return the result.",
		Long:  "Scan an image and return the result.",
		RunE: func(c *cobra.Command, _ []string) error {
			if image == "" {
				return errors.New("--image must be set")
			}
			return scanImage(image, flags.Timeout(c))
		},
	}

	c.Flags().StringVarP(&image, "image", "i", "", "image name and reference. (e.g. nginx:latest or nginx@sha256:...)")
	return c
}

func scanImage(image string, timeout time.Duration) error {
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

	imageResult, err := service.ScanImage(ctx, &v1.ScanImageRequest{ImageName: image})
	if err != nil {
		return err
	}

	marshaler := &jsonpb.Marshaler{
		Indent: "  ",
	}
	result, err := marshaler.MarshalToString(imageResult)
	if err != nil {
		return err
	}
	fmt.Println(result)
	return nil
}
