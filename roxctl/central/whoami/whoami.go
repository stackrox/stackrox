package whoami

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

type centralWhoAmICommand struct {
	// Properties that are injected or constructed.
	env     environment.Environment
	timeout time.Duration
}

// Command defines the central command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	cbr := &cobra.Command{
		Use: "whoami",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			return makeCentralWhoAmICommand(cliEnvironment, c).whoami()
		}),
	}

	flags.AddTimeout(cbr)
	return cbr
}

func makeCentralWhoAmICommand(cliEnvironment environment.Environment, cbr *cobra.Command) *centralWhoAmICommand {
	return &centralWhoAmICommand{
		env:     cliEnvironment,
		timeout: flags.Timeout(cbr),
	}
}

func (cmd *centralWhoAmICommand) whoami() error {
	conn, err := cmd.env.GRPCConnection()
	if err != nil {
		return err
	}
	defer utils.IgnoreError(conn.Close)

	ctx, cancel := context.WithTimeout(context.Background(), cmd.timeout)
	defer cancel()

	auth, err := v1.NewAuthServiceClient(conn).GetAuthStatus(ctx, &v1.Empty{})
	if err != nil {
		return err
	}

	perms, err := v1.NewRoleServiceClient(conn).GetMyPermissions(ctx, &v1.Empty{})
	if err != nil {
		return err
	}

	roles, err := v1.NewRoleServiceClient(conn).GetRoles(ctx, &v1.Empty{})
	if err != nil {
		return err
	}

	// Lexicographically sort the set of resources we have (known) access to.
	resourceToAccess := perms.GetResourceToAccess()
	resources := make([]string, 0, len(resourceToAccess))
	for resource := range resourceToAccess {
		resources = append(resources, resource)
	}
	sort.Strings(resources)

	// Print user information.
	cmd.env.Logger().PrintfLn("User:")
	cmd.env.Logger().PrintfLn("  %s", auth.GetUserId())

	// Print resource access information
	cmd.printRoles(roles.GetRoles())
	cmd.env.Logger().PrintfLn("Access:")
	for _, resource := range resources {
		access := resourceToAccess[resource]
		cmd.env.Logger().PrintfLn("  %s %s", accessString(access), resource)
	}

	return nil
}

func (cmd *centralWhoAmICommand) printRoles(roles []*storage.Role) {
	cmd.env.Logger().PrintfLn("Roles:")
	sb := strings.Builder{}
	sb.WriteRune(' ')
	for i, r := range roles {
		sb.WriteString(r.GetName())
		if i != len(roles)-1 {
			sb.WriteString(", ")
		}
	}
	cmd.env.Logger().PrintfLn(sb.String())
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
