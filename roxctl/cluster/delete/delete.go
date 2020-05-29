package delete

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/util"
)

const (
	connectionTimeout = 5 * time.Second
)

// Command defines the deploy command tree
func Command() *cobra.Command {
	var name string
	c := &cobra.Command{
		Use:   "delete",
		Short: "Delete removes the Sensor from Central, but does not delete any orchestrator objects",
		Long:  "Delete removes the Sensor from Central, but does not delete any orchestrator objects",
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			if name == "" {
				return errors.New("--name is required")
			}
			return deleteCluster(name)
		}),
	}
	c.PersistentFlags().StringVar(&name, "name", "", "cluster name to delete")
	return c
}

func getClusters(svc v1.ClustersServiceClient) ([]*storage.Cluster, error) {
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()
	clusterResponse, err := svc.GetClusters(ctx, &v1.GetClustersRequest{})
	if err != nil {
		return nil, err
	}
	return clusterResponse.GetClusters(), nil
}

func deleteCluster(name string) error {
	conn, err := common.GetGRPCConnection()
	if err != nil {
		return err
	}
	service := v1.NewClustersServiceClient(conn)
	clusters, err := getClusters(service)
	if err != nil {
		return err
	}
	validClusters := make([]string, 0, len(clusters))
	var cluster *storage.Cluster
	for _, c := range clusters {
		validClusters = append(validClusters, c.GetName())
		if strings.EqualFold(c.GetName(), name) {
			cluster = c
			break
		}
	}
	if cluster == nil {
		return fmt.Errorf("cluster with name %q not found. Valid clusters are [ %s ]", name, strings.Join(validClusters, " | "))
	}
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()
	_, err = service.DeleteCluster(ctx, &v1.ResourceByID{Id: cluster.GetId()})
	if err != nil {
		return err
	}
	fmt.Printf("Successfully deleted cluster %q\n", name)
	return nil
}
