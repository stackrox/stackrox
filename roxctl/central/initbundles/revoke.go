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

	revokeResp, err := svc.RevokeInitBundle(ctx, &v1.InitBundleRevokeRequest{Ids: revokeInitBundleIds})
	if err != nil {
		return errors.Wrap(err, "revoking init bundles")
	}
	printResponseResult(revokeResp)

	if len(revokeResp.GetInitBundleRevocationErrors()) == 0 {
		fmt.Fprintf(os.Stdout, "Revoked %d init bundle(s)\n", len(revokeInitBundleIds))
	} else {
		fmt.Fprintf(os.Stdout, "Failed. Revoked %d of %d init bundle(s)\n", len(revokeResp.GetInitBundleRevokedIds()), len(revokeInitBundleIds))
	}
	return nil
}

func printResponseResult(resp *v1.InitBundleRevokeResponse) {
	for _, id := range resp.GetInitBundleRevokedIds() {
		fmt.Fprintf(os.Stdout, "Revoked %q\n", id)
	}
	for _, revokeErr := range resp.GetInitBundleRevocationErrors() {
		fmt.Fprintf(os.Stderr, "Error revoking %q: %s\n", revokeErr.Id, revokeErr.Error)
	}
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
