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
)

// Command defines the central command tree
func Command() *cobra.Command {
	c := &cobra.Command{
		Use:   "whoami",
		Short: "Authentication information",
		Long:  "Whoami prints information about the current authentication method",
		RunE: func(c *cobra.Command, _ []string) error {
			timeout := flags.Timeout(c)
			return whoami(timeout)
		},
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

	role, err := v1.NewRoleServiceClient(conn).GetMyPermissions(ctx, &v1.Empty{})
	if err != nil {
		return err
	}

	// Lexicographically sort the set of resources we have (known) access to.
	resourceToAccess := role.GetResourceToAccess()
	resources := make([]string, 0, len(resourceToAccess))
	for resource := range resourceToAccess {
		resources = append(resources, resource)
	}
	sort.Strings(resources)

	// Print user information.
	fmt.Println("User:")
	fmt.Printf("  %s\n", auth.GetUserId())

	// Print resource access information
	fmt.Println("Access:")
	fmt.Printf("  %s Global\n", accessString(role.GetGlobalAccess()))
	for _, resource := range resources {
		access := resourceToAccess[resource]
		fmt.Printf("  %s %s\n", accessString(access), resource)
	}

	return nil
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
