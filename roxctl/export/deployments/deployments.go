package deployments

import (
	"context"
	"fmt"
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
		Use:   "deployments",
		Short: "Commands related to exporting deployments from Central.",
	}
	flags.AddTimeoutWithDefault(c, 10*time.Minute)

	c.RunE = func(cmd *cobra.Command, args []string) error {
		conn, err := cliEnvironment.GRPCConnection()
		if err != nil {
			return errors.Wrap(err, "could not establish gRPC connection to central")
		}

		defer utils.IgnoreError(conn.Close)

		svc := v1.NewDeploymentServiceClient(conn)
		ctx, cancel := context.WithTimeout(pkgCommon.Context(), flags.Timeout(cmd))
		defer cancel()

		client, err := svc.Export(ctx, &v1.ExportDeploymentRequest{})
		if err != nil {
			return errors.Wrap(err, "could not initialize stream client")
		}

		marshaler := &jsonpb.Marshaler{}
		for {
			deployment, err := client.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				panic(err)
			}
			serialized, err := marshaler.MarshalToString(deployment)
			if err != nil {
				return err
			}
			fmt.Println(serialized)
		}
		return nil
	}
	return c
}
