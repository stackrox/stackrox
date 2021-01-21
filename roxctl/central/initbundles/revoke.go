package initbundles

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
)

func applyRevokeInitBundles(ctx context.Context, svc v1.ClusterInitServiceClient, idsOrNames set.StringSet) error {
	resp, err := svc.GetInitBundles(ctx, &v1.Empty{})
	if err != nil {
		return err
	}

	var revokeInitBundleIds []string
	for _, meta := range resp.Items {
		if idsOrNames.Remove(meta.GetId()) || idsOrNames.Remove(meta.GetName()) {
			revokeInitBundleIds = append(revokeInitBundleIds, meta.GetId())
		}
	}

	if len(idsOrNames) != 0 {
		return errors.Errorf("could not find init bundle(s) %s", strings.Join(idsOrNames.AsSlice(), ", "))
	}

	if _, err := svc.RevokeInitBundle(ctx, &v1.InitBundleRevokeRequest{Ids: revokeInitBundleIds}); err != nil {
		return errors.Wrap(err, "revoking init bundles")
	}

	fmt.Fprintf(os.Stdout, "Removed %d init bundle(s)", len(revokeInitBundleIds))
	return nil
}

func revokeInitBundles(idsOrNames []string) error {
	ctx, cancel := context.WithTimeout(pkgCommon.Context(), contextTimeout)
	defer cancel()

	conn, err := common.GetGRPCConnection()
	if err != nil {
		return err
	}
	defer utils.IgnoreError(conn.Close)
	svc := v1.NewClusterInitServiceClient(conn)

	idsOrNamesSet := set.NewStringSet(idsOrNames...)
	if err = applyRevokeInitBundles(ctx, svc, idsOrNamesSet); err != nil {
		return err
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
