package initbundles

import (
	"context"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
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

func applyRevokeInitBundles(ctx context.Context, cliEnvironment environment.Environment, svc v1.ClusterInitServiceClient, idsOrNames set.StringSet, force bool) error {
	resp, err := svc.GetInitBundles(ctx, &v1.Empty{})
	if err != nil {
		return err
	}

	impactedIDNameMap := map[string]string{}

	var revokeInitBundleIds []string
	for _, meta := range resp.Items {
		if idsOrNames.Remove(meta.GetId()) || idsOrNames.Remove(meta.GetName()) {
			revokeInitBundleIds = append(revokeInitBundleIds, meta.GetId())
			for _, impactedCluster := range meta.GetImpactedClusters() {
				impactedIDNameMap[impactedCluster.GetId()] = impactedCluster.GetName()
			}
		}
	}

	if len(idsOrNames) != 0 {
		return errox.NotFound.Newf("could not find init bundle(s) %s", strings.Join(idsOrNames.AsSlice(), ", "))
	}

	impactedClusterIds := make([]string, 0, len(impactedIDNameMap))
	for id := range impactedIDNameMap {
		impactedClusterIds = append(impactedClusterIds, id)
	}

	if !force {
		confirm, err := confirmImpactedClusterIds(impactedIDNameMap, cliEnvironment.InputOutput().Out(), cliEnvironment.InputOutput().In())
		if err != nil {
			return err
		}
		if !confirm {
			return nil
		}
	}

	revokeResp, err := svc.RevokeInitBundle(ctx, &v1.InitBundleRevokeRequest{Ids: revokeInitBundleIds, ConfirmImpactedClustersIds: impactedClusterIds})
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

func confirmImpactedClusterIds(impactedClusterIDNameMap map[string]string, out io.Writer, in io.Reader) (bool, error) {

	if len(impactedClusterIDNameMap) == 0 {
		return true, nil
	}

	impactedClusters := make([][2]string, 0, len(impactedClusterIDNameMap))
	for id, name := range impactedClusterIDNameMap {
		impactedClusters = append(impactedClusters, [2]string{id, name})
	}
	sort.Slice(impactedClusters, func(i, j int) bool {
		return impactedClusters[i][1] < impactedClusters[j][1]
	})

	_, _ = out.Write([]byte("Revoking init bundle(s) will impact the following cluster(s):\n"))

	t := tabwriter.NewWriter(out, 4, 8, 2, '\t', 0)
	_, _ = t.Write([]byte("Cluster ID\tCluster Name\n"))
	for i := 0; i < len(impactedClusters); i++ {
		_, _ = t.Write([]byte(impactedClusters[i][0] + "\t" + impactedClusters[i][1] + "\n"))
	}
	_ = t.Flush()

	_, _ = out.Write([]byte("Are you sure you want to revoke the init bundle(s)? [y/N] "))

	return flags.ReadUserYesNoConfirmation(in)
}

func printResponseResult(logger logger.Logger, resp *v1.InitBundleRevokeResponse) {
	for _, id := range resp.GetInitBundleRevokedIds() {
		logger.InfofLn("Revoked %q", id)
	}
	for _, revokeErr := range resp.GetInitBundleRevocationErrors() {
		logger.ErrfLn("Error revoking %q: %s", revokeErr.Id, revokeErr.Error)
	}
}

func revokeInitBundles(cliEnvironment environment.Environment, idsOrNames []string, force bool,
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
	if err = applyRevokeInitBundles(ctx, cliEnvironment, svc, idsOrNamesSet, force); err != nil {
		return err
	}
	return nil
}

// revokeCommand implements the command for revoking init bundles.
func revokeCommand(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "revoke <init bundle ID or name> [<init bundle ID or name> ...]",
		Short: "Revoke a cluster init bundle",
		Long:  "Revoke an init bundle for bootstrapping new StackRox secured clusters.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			force, err := cmd.Flags().GetBool("force")
			if err != nil {
				return errors.Wrap(err, "getting force flag")
			}
			return revokeInitBundles(cliEnvironment, args, force, flags.Timeout(cmd), flags.RetryTimeout(cmd))
		},
	}

	c.Flags().BoolP("force", "f", false, "Force revocation without confirmation.")

	return c
}
