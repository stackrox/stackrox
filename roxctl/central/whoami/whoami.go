package whoami

import (
	"context"
	"slices"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
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

	cmd.checkVersionSkew(ctx, conn)

	return nil
}

const maxVersionSkew = 3

func (cmd *centralWhoAmICommand) checkVersionSkew(ctx context.Context, conn *grpc.ClientConn) {
	metadata, err := v1.NewMetadataServiceClient(conn).GetMetadata(ctx, &v1.Empty{})
	if err != nil {
		cmd.env.Logger().WarnfLn("Could not fetch Central metadata for version skew check: %v", err)
		return
	}

	centralVersion := metadata.GetVersion()
	if centralVersion == "" {
		return
	}

	localBumps, err := version.EmbeddedMajorBumps()
	if err != nil {
		cmd.env.Logger().WarnfLn("Could not load embedded version bump data: %v", err)
		return
	}

	remoteBumps := bumpsFromProto(metadata.GetKnownMajorBumps())
	merged := version.MergeBumps(localBumps, remoteBumps)

	result := version.CheckSkew(version.GetMainVersion(), centralVersion, maxVersionSkew, merged)
	switch result.Status {
	case version.SkewWarning:
		cmd.env.Logger().WarnfLn("WARNING: %s", result.Message)
	case version.SkewOK:
		cmd.env.Logger().InfofLn("Version check: %s", result.Message)
	}
}

func bumpsFromProto(pb []*v1.MajorVersionBump) []version.MajorBump {
	out := make([]version.MajorBump, 0, len(pb))
	for _, b := range pb {
		from, err := version.ParseXY(b.GetFromVersion())
		if err != nil {
			continue
		}
		to, err := version.ParseXY(b.GetToVersion())
		if err != nil {
			continue
		}
		out = append(out, version.MajorBump{From: from, To: to})
	}
	return out
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
