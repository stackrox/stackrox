package whoami

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/productstreams"
	"github.com/stackrox/rox/pkg/version/versioncompatibility"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
	"google.golang.org/grpc"
)

type centralWhoAmICommand struct {
	// Properties that are injected or constructed.
	env          environment.Environment
	timeout      time.Duration
	retryTimeout time.Duration
}

// Command defines the central command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	cbr := &cobra.Command{
		Use:   "whoami",
		Short: "Display information about the current user and their authentication method",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			return makeCentralWhoAmICommand(cliEnvironment, c).whoami()
		}),
	}

	flags.AddTimeout(cbr)
	flags.AddRetryTimeout(cbr)
	return cbr
}

func makeCentralWhoAmICommand(cliEnvironment environment.Environment, cbr *cobra.Command) *centralWhoAmICommand {
	return &centralWhoAmICommand{
		env:          cliEnvironment,
		timeout:      flags.Timeout(cbr),
		retryTimeout: flags.RetryTimeout(cbr),
	}
}

func (cmd *centralWhoAmICommand) whoami() error {
	conn, err := cmd.env.GRPCConnection(common.WithRetryTimeout(cmd.retryTimeout))
	if err != nil {
		return errors.Wrap(err, "establishing GRPC connection to retrieve user role information")
	}
	defer utils.IgnoreError(conn.Close)

	ctx, cancel := context.WithTimeout(context.Background(), cmd.timeout)
	defer cancel()

	auth, err := v1.NewAuthServiceClient(conn).GetAuthStatus(ctx, &v1.Empty{})
	if err != nil {
		return errors.Wrap(err, "getting auth status")
	}

	perms, err := v1.NewRoleServiceClient(conn).GetMyPermissions(ctx, &v1.Empty{})
	if err != nil {
		return errors.Wrap(err, "getting user permissions")
	}

	// Lexicographically sort the set of resources we have (known) access to.
	resourceToAccess := perms.GetResourceToAccess()
	resources := make([]string, 0, len(resourceToAccess))
	for resource := range resourceToAccess {
		resources = append(resources, resource)
	}
	slices.Sort(resources)

	cmd.env.Logger().PrintfLn(`UserID:
	%s
User name:
	%s`, auth.GetUserId(), auth.GetUserInfo().GetFriendlyName())

	// Print the roles associated with the user.
	cmd.env.Logger().PrintfLn("Roles:")
	for _, role := range auth.GetUserInfo().GetRoles() {
		cmd.env.Logger().PrintfLn("\t- %s", role.GetName())
	}

	// Print resource access information.
	cmd.env.Logger().PrintfLn("Access:")
	for _, resource := range resources {
		access := resourceToAccess[resource]
		cmd.env.Logger().PrintfLn("\t%s %s", accessString(access), resource)
	}

	cmd.checkVersionCompatibility(ctx, conn)

	return nil
}

func (cmd *centralWhoAmICommand) checkVersionCompatibility(ctx context.Context, conn *grpc.ClientConn) {
	metadata, err := v1.NewMetadataServiceClient(conn).GetMetadata(ctx, &v1.Empty{})
	if err != nil {
		cmd.env.Logger().WarnfLn("getting metadata: %v", err)
		return
	}

	roxctlVersion := version.GetMainVersion()

	centralVersion := metadata.GetVersion()
	if centralVersion == "" {
		cmd.env.Logger().WarnfLn("Central did not report its version; skipping compatibility check")
		return
	}

	centralXY, err := productstreams.ParseXYFromVersionString(centralVersion)
	if err != nil {
		cmd.env.Logger().WarnfLn("parsing Central version %q: %v", centralVersion, err)
		return
	}

	compat, err := versioncompatibility.ClassifyVersion(centralXY)
	if err != nil {
		cmd.env.Logger().WarnfLn("classifying Central version %q: %v", centralVersion, err)
		return
	}
	if compat != versioncompatibility.IncompatibleAhead && compat != versioncompatibility.IncompatibleBehind {
		return
	}
	versionRange, err := versioncompatibility.CompatibleVersions()
	if err != nil {
		cmd.env.Logger().WarnfLn("generating compatible versions %q: %v", roxctlVersion, err)
		return
	}
	compatRange := formatVersionRange(versionRange)
	w := cmd.env.InputOutput().ErrOut()

	switch compat {
	case versioncompatibility.IncompatibleAhead:
		fmt.Fprintf(w, "Warning: Your roxctl %s is too old for this Central %s. "+
			"Correct functioning is not guaranteed. "+
			"Use roxctl version matching the Central version or at least such that the Central version is within the roxctl compatibility range.\n",
			roxctlVersion, centralVersion)
		fmt.Fprintf(w, "         roxctl: %s | Central: %s | Compatible Centrals: %s\n",
			roxctlVersion, centralVersion, compatRange)
	case versioncompatibility.IncompatibleBehind:
		fmt.Fprintf(w, "Warning: Your roxctl %s is too new for this Central %s. "+
			"Correct functioning is not guaranteed. "+
			"Use roxctl version matching the Central version or at least such that the Central version is within the roxctl compatibility range.\n",
			roxctlVersion, centralVersion)
		fmt.Fprintf(w, "         roxctl: %s | Central: %s | Compatible Centrals: %s\n",
			roxctlVersion, centralVersion, compatRange)
	}
}

func formatVersionRange(versions []productstreams.XYVersion) string {
	strs := make([]string, len(versions))
	for i, v := range versions {
		strs[i] = v.String()
	}
	return strings.Join(strs, ", ")
}

func accessString(access storage.Access) string {
	switch access {
	case storage.Access_READ_WRITE_ACCESS:
		return "rw"
	case storage.Access_READ_ACCESS:
		return "r-"
	default:
		return "--"
	}
}
