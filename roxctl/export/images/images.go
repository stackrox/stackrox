package images

import (
	"context"
	"io"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
)

// Command defines the central command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "images",
		Short: "Commands related to exporting images from Central.",
	}
	flags.AddTimeoutWithDefault(c, 10*time.Minute)

	c.RunE = func(cmd *cobra.Command, args []string) error {
		conn, err := cliEnvironment.GRPCConnection()
		if err != nil {
			return errors.Wrap(err, "could not establish gRPC connection to central")
		}

		defer utils.IgnoreError(conn.Close)

		svc := v1.NewImageServiceClient(conn)
		ctx, cancel := context.WithTimeout(pkgCommon.Context(), flags.Timeout(cmd))
		defer cancel()

		client, err := svc.Export(ctx, &v1.ExportImageRequest{})
		if err != nil {
			return errors.Wrap(err, "could not initialize stream client")
		}

		marshaler := &jsonpb.Marshaler{}
		for {
			image, err := client.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return errors.Wrap(err, "stream broken by unexpected error")
			}
			if err := marshaler.Marshal(cliEnvironment.InputOutput().Out(), image); err != nil {
				return errors.Wrap(err, "unable to serialize image")
			}
		}
		return nil
	}
	return c
}
