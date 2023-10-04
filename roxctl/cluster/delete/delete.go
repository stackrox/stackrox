package delete

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
)

// Command removes a Sensor from Central without deleting any orchestrator objects.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	clusterDeleteCmd := &clusterDeleteCommand{env: cliEnvironment}

	cbr := &cobra.Command{
		Use:   "delete",
		Short: "Remove a Sensor from Central.",
		Long:  "Remove a Sensor from Central, without deleting any orchestrator objects.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := clusterDeleteCmd.Construct(args, cmd); err != nil {
				return err
			}
			if err := clusterDeleteCmd.Validate(); err != nil {
				return err
			}

			return clusterDeleteCmd.Delete()
		},
	}
	cbr.PersistentFlags().StringVar(&clusterDeleteCmd.name, "name", "", "cluster name to delete")
	return cbr
}

type clusterDeleteCommand struct {
	// Properties that are bound to cobra flags.
	name string

	// Properties that are injected or constructed.
	env          environment.Environment
	timeout      time.Duration
	retryTimeout time.Duration
}

func (cmd *clusterDeleteCommand) Construct(_ []string, cbr *cobra.Command) error {
	cmd.timeout = flags.Timeout(cbr)
	cmd.retryTimeout = flags.RetryTimeout(cbr)
	return nil
}

func (cmd *clusterDeleteCommand) Validate() error {
	if cmd.name == "" {
		return common.ErrInvalidCommandOption
	}
	return nil
}

func (cmd *clusterDeleteCommand) Delete() error {
	conn, err := cmd.env.GRPCConnection(cmd.retryTimeout)
	if err != nil {
		return errors.Wrap(err, "could not establish gRPC connection to central")
	}
	service := v1.NewClustersServiceClient(conn)
	clusters, err := cmd.getClusters(service)
	if err != nil {
		return err
	}

	validClusters := make([]string, 0, len(clusters))
	var cluster *storage.Cluster
	for _, cl := range clusters {
		validClusters = append(validClusters, cl.GetName())
		if strings.EqualFold(cl.GetName(), cmd.name) {
			cluster = cl
			break
		}
	}
	if cluster == nil {
		cmd.env.Logger().ErrfLn("Cluster with name %q not found. Valid clusters are [ %s ]", cmd.name, strings.Join(validClusters, " | "))
		return errox.NotFound
	}

	ctx, cancel := context.WithTimeout(context.Background(), cmd.timeout)
	defer cancel()
	_, err = service.DeleteCluster(ctx, &v1.ResourceByID{Id: cluster.GetId()})
	if err != nil {
		return errors.Wrapf(err, "could not delete cluster: %q", cluster.GetId())
	}

	cmd.env.Logger().PrintfLn("Successfully deleted cluster %q\n", cmd.name)
	return nil
}

func (cmd *clusterDeleteCommand) getClusters(svc v1.ClustersServiceClient) ([]*storage.Cluster, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmd.timeout)
	defer cancel()
	clusterResponse, err := svc.GetClusters(ctx, &v1.GetClustersRequest{})
	if err != nil {
		return nil, errors.Wrap(err, "could not get clusters")
	}
	return clusterResponse.GetClusters(), nil
}
