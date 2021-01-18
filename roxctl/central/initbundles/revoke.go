package initbundles

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
)

func revokeSingleInitBundle(ctx context.Context, svc v1.ClusterInitServiceClient, id string) error {
	return errors.New("not implemented")
}

func revokeInitBundles(ids []string) error {
	ctx, cancel := context.WithTimeout(pkgCommon.Context(), contextTimeout)
	defer cancel()

	conn, err := common.GetGRPCConnection()
	if err != nil {
		return err
	}
	defer utils.IgnoreError(conn.Close)
	svc := v1.NewClusterInitServiceClient(conn)

	for _, id := range ids {
		err = revokeSingleInitBundle(ctx, svc, id)
		if err != nil {
			return err
		}
	}

	return nil
}

// revokeCommand implements the command for revoking init bundles.
func revokeCommand() *cobra.Command {
	c := &cobra.Command{
		Use:  "revoke <init bundle ID or name> [<init bundle ID or name> ...]",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return revokeInitBundles(args)
		},
	}

	return c
}
