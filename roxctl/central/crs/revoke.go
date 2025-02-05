package crs

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errox"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/logger"
)

func applyRevokeCRSs(ctx context.Context, cliEnvironment environment.Environment, svc v1.ClusterInitServiceClient, idsOrNames set.StringSet) error {
	resp, err := svc.GetCRSs(ctx, &v1.Empty{})
	if err != nil {
		return err
	}

	var revokeIds []string
	idNames := make(map[string]string)

	for _, meta := range resp.Items {
		idNames[meta.GetId()] = meta.GetName()
		if idsOrNames.Remove(meta.GetId()) || idsOrNames.Remove(meta.GetName()) {
			revokeIds = append(revokeIds, meta.GetId())
		}
	}

	if len(idsOrNames) != 0 {
		return errox.NotFound.Newf("could not find CRS(s) %s", strings.Join(idsOrNames.AsSlice(), ", "))
	}

	revokeResp, err := svc.RevokeCRS(ctx, &v1.CRSRevokeRequest{Ids: revokeIds})
	if err != nil {
		return errors.Wrap(err, "revoking CRSs")
	}
	printResponseResult(cliEnvironment.Logger(), idNames, revokeIds, revokeResp)

	return nil
}

func printResponseResult(logger logger.Logger, idNames map[string]string, revokeIds []string, resp *v1.CRSRevokeResponse) {
	for _, id := range resp.GetRevokedIds() {
		logger.InfofLn("Revoked %s (%q)", id, idNames[id])
	}
	for _, revokeErr := range resp.GetCrsRevocationErrors() {
		id := revokeErr.GetId()
		logger.ErrfLn("Error revoking %s (%q): %s", id, idNames[id], revokeErr.Error)
	}

	if len(resp.GetCrsRevocationErrors()) == 0 {
		logger.InfofLn("Revoked %d CRS(s)", len(revokeIds))
	} else {
		logger.ErrfLn("Failed. Revoked %d of %d CRS(s)", len(resp.GetRevokedIds()), len(revokeIds))
	}
}

func revokeCRSs(cliEnvironment environment.Environment, idsOrNames []string,
	timeout time.Duration, retryTimeout time.Duration,
) error {
	ctx, cancel := context.WithTimeout(pkgCommon.Context(), timeout)
	defer cancel()

	conn, err := cliEnvironment.GRPCConnection(common.WithRetryTimeout(retryTimeout))
	if err != nil {
		return err
	}
	defer utils.IgnoreError(conn.Close)
	svc := v1.NewClusterInitServiceClient(conn)

	idsOrNamesSet := set.NewStringSet(idsOrNames...)
	return applyRevokeCRSs(ctx, cliEnvironment, svc, idsOrNamesSet)
}

// revokeCommand implements the command for revoking CRSs.
func revokeCommand(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "revoke <CRS ID or name> [<CRS ID or name> ...]",
		Short: "Revoke a Cluster Registration Secret",
		Long:  "Revoke a Cluster Registration Secret (CRS) for bootstrapping new Secured Clusters.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return revokeCRSs(cliEnvironment, args, flags.Timeout(cmd), flags.RetryTimeout(cmd))
		},
	}

	return c
}
