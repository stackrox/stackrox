package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

func RootCmd(ctx context.Context) *cobra.Command {
	cmd := cobra.Command{
		Use:          "agent",
		Short:        "Collects index reports for vulnerability scanning of virtual machines.",
		SilenceUsage: true,
	}
	cmd.SetContext(ctx)
	cmd.AddCommand(ServeCmd(ctx))
	return &cmd
}
