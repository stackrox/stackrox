package whoami

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

// Command defines the central command tree
func Command() *cobra.Command {
	c := &cobra.Command{
		Use: "whoami",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			timeout := flags.Timeout(c)
			return whoami(timeout)
		}),
	}

	flags.AddTimeout(c)
	return c
}

func whoami(timeout time.Duration) error {
	conn, err := common.GetGRPCConnection()
	if err != nil {
		return err
	}
	defer utils.IgnoreError(conn.Close)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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
	fmt.Println("User:")
	fmt.Printf("  %s\n", auth.GetUserId())

	// Print resource access information
	printRoles(roles.GetRoles())
	fmt.Println("Access:")
	for _, resource := range resources {
		access := resourceToAccess[resource]
		fmt.Printf("  %s %s\n", accessString(access), resource)
	}

	return nil
}

func printRoles(roles []*storage.Role) {
	fmt.Println("Roles:")
	fmt.Print(" ")
	for i, r := range roles {
		fmt.Print(r.GetName())
		if i != len(roles)-1 {
			fmt.Print(", ")
		}
	}
	fmt.Print("\n")
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
