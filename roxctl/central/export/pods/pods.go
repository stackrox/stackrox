package pods

import (
	"context"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/jsonutil"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
)

// Command defines the central command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "pods",
		Short: "(Technology Preview) Exports all pods from Central",
		Long:  "Exports all pods from Central." + common.TechPreviewLongText,
	}

	c.RunE = func(cmd *cobra.Command, args []string) error {
		conn, err := cliEnvironment.GRPCConnection()
		if err != nil {
			return errors.Wrap(err, "could not establish gRPC connection to central")
		}
		defer utils.IgnoreError(conn.Close)

		svc := v1.NewPodServiceClient(conn)
		ctx, cancel := context.WithTimeout(pkgCommon.Context(), flags.Timeout(cmd))
		defer cancel()

		client, err := svc.ExportPods(ctx, &v1.ExportPodRequest{})
		if err != nil {
			return errors.Wrap(err, "could not initialize stream client")
		}
		for {
			pod, err := client.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return errors.Wrap(err, "stream broken by unexpected error")
			}
			if err := jsonutil.Marshal(cliEnvironment.InputOutput().Out(), pod); err != nil {
				return errors.Wrap(err, "unable to serialize pod")
			}
		}
		return nil
	}
	return c
}
