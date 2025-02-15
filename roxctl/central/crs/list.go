package crs

import (
	"context"
	"fmt"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/protocompat"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

func listCRSs(cliEnvironment environment.Environment, timeout time.Duration, retryTimeout time.Duration) error {
	ctx, cancel := context.WithTimeout(pkgCommon.Context(), timeout)
	defer cancel()

	conn, err := cliEnvironment.GRPCConnection(common.WithRetryTimeout(retryTimeout))
	if err != nil {
		return err
	}
	defer utils.IgnoreError(conn.Close)
	svc := v1.NewClusterInitServiceClient(conn)

	rsp, err := svc.GetCRSs(ctx, &v1.Empty{})
	if err != nil {
		return errors.Wrap(err, "getting all CRSs")
	}

	crsMetas := rsp.GetItems()
	sort.Slice(crsMetas, func(i, j int) bool { return crsMetas[i].GetName() < crsMetas[j].GetName() })

	tabWriter := tabwriter.NewWriter(cliEnvironment.InputOutput().Out(), 4, 8, 2, '\t', 0)
	fmt.Fprintln(tabWriter, "Name\tCreated at\tExpires at\tCreated by\tID")
	fmt.Fprintln(tabWriter, "====\t==========\t==========\t==========\t==")

	for _, crsMeta := range crsMetas {
		name := crsMeta.GetName()
		if name == "" {
			name = "(empty)"
		}
		fmt.Fprintf(tabWriter, "%s\t%s\t%s\t%s\t%s\n",
			name,
			protocompat.ConvertTimestampToString(crsMeta.GetCreatedAt(), time.RFC3339),
			protocompat.ConvertTimestampToString(crsMeta.GetExpiresAt(), time.RFC3339),
			getPrettyUser(crsMeta.GetCreatedBy()),
			crsMeta.GetId(),
		)
	}
	return errors.Wrap(tabWriter.Flush(), "flushing tabular output")
}

// listCommand implements the command for listing CRSs.
func listCommand(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List Cluster Registration Secrets",
		Long:  "List all previously generated Cluster Registration Secrets (CRSs) for bootstrapping new Secured Clusters.",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			return listCRSs(cliEnvironment, flags.Timeout(c), flags.RetryTimeout(c))
		}),
	}
	return c
}
