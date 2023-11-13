package util

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errox"
	pkgCommon "github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
)

// ResolveClusterID returns the cluster ID corresponding to the given id or name,
// or an error if no matching cluster was found.
func ResolveClusterID(idOrName string, timeout time.Duration, retryTimeout time.Duration, env environment.Environment) (string, error) {
	if _, err := uuid.FromString(idOrName); err == nil {
		return idOrName, nil
	}

	conn, err := env.GRPCConnection(common.WithRetryTimeout(retryTimeout))
	if err != nil {
		return "", err
	}

	service := v1.NewClustersServiceClient(conn)

	ctx, cancel := context.WithTimeout(pkgCommon.Context(), timeout)
	defer cancel()

	clusters, err := service.GetClusters(ctx, &v1.GetClustersRequest{
		Query: fmt.Sprintf("%s:%s", search.Cluster, idOrName),
	})
	if err != nil {
		return "", err
	}

	for _, cluster := range clusters.GetClusters() {
		if cluster.GetName() == idOrName {
			return cluster.GetId(), nil
		}
	}
	return "", errox.NotFound.Newf("no cluster with name %q found", idOrName)
}
