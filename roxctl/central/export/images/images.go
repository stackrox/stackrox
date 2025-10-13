package images

import (
	"context"
	"io"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/jsonutil"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"google.golang.org/grpc/codes"
)

// Command defines the central command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "images",
		Short: "(Technology Preview) Exports all images from Central",
		Long:  "Exports all images from Central." + common.TechPreviewLongText,
	}

	c.RunE = func(cmd *cobra.Command, args []string) error {
		// Override retriable codes to not included ResourceExhausted
		grpc_retry.DefaultRetriableCodes = []codes.Code{
			codes.Unavailable,
		}
		conn, err := cliEnvironment.GRPCConnection()
		if err != nil {
			return errors.Wrap(err, "could not establish gRPC connection to central")
		}
		defer utils.IgnoreError(conn.Close)

		svc := v1.NewImageServiceClient(conn)
		ctx, cancel := context.WithTimeout(pkgCommon.Context(), flags.Timeout(cmd))
		defer cancel()

		client, err := svc.ExportImages(ctx, &v1.ExportImageRequest{})
		if err != nil {
			return errors.Wrap(err, "could not initialize stream client")
		}

		for {
			image, err := client.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return errors.Wrap(err, "stream broken by unexpected error")
			}
			if err := jsonutil.Marshal(cliEnvironment.InputOutput().Out(), image); err != nil {
				return errors.Wrap(err, "unable to serialize image")
			}
		}
		return nil
	}
	return c
}
