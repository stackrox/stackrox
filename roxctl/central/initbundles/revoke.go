package initbundles

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
)

func applyRevokeInitBundles(ctx context.Context, cliEnvironment common.Environment, svc v1.ClusterInitServiceClient, idsOrNames set.StringSet) error {
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
	printResponseResult(cliEnvironment.Logger(), revokeResp)

	if len(revokeResp.GetInitBundleRevocationErrors()) == 0 {
		cliEnvironment.Logger().InfofLn("Revoked %d init bundle(s)", len(revokeInitBundleIds))
	} else {
		cliEnvironment.Logger().ErrfLn("Failed. Revoked %d of %d init bundle(s)", len(revokeResp.GetInitBundleRevokedIds()), len(revokeInitBundleIds))
	}
	return nil
}

func printResponseResult(logger common.Logger, resp *v1.InitBundleRevokeResponse) {
	for _, id := range resp.GetInitBundleRevokedIds() {
		logger.InfofLn("Revoked %q", id)
	}
	for _, revokeErr := range resp.GetInitBundleRevocationErrors() {
		logger.ErrfLn("Error revoking %q: %s", revokeErr.Id, revokeErr.Error)
	}
}

func revokeInitBundles(cliEnvironment common.Environment, idsOrNames []string) error {
	ctx, cancel := context.WithTimeout(pkgCommon.Context(), contextTimeout)
	defer cancel()

	conn, err := cliEnvironment.GRPCConnection()
	if err != nil {
		return err
	}
	defer utils.IgnoreError(conn.Close)
	svc := v1.NewClusterInitServiceClient(conn)

	idsOrNamesSet := set.NewStringSet(idsOrNames...)
	if err = applyRevokeInitBundles(ctx, cliEnvironment, svc, idsOrNamesSet); err != nil {
		return err
	}
	return nil
}

// revokeCommand implements the command for revoking init bundles.
func revokeCommand(cliEnvironment common.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:  "revoke <init bundle ID or name> [<init bundle ID or name> ...]",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return revokeInitBundles(cliEnvironment, args)
		},
	}

	return c
}
