package reset

import (
	"context"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
)

// Command controls all of the functions being applied to a central-db
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "reset",
		Short: "Resets DB query statistics in the database",
		RunE: func(c *cobra.Command, args []string) error {
			conn, err := cliEnvironment.GRPCConnection()
			if err != nil {
				return errors.Wrap(err, "could not establish gRPC connection to central")
			}
			defer utils.IgnoreError(conn.Close)

			ctx, cancel := context.WithTimeout(context.Background(), flags.Timeout(c))
			defer cancel()

			svc := v1.NewDebugServiceClient(conn)
			_, err = svc.ResetDBStats(ctx, &v1.Empty{})
			if err != nil {
				return errors.Wrap(err, "could not reset pg_stat_statements")
			}
			return nil
		},
	}
	return c
}
