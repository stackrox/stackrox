package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/networkflow/datastore/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/timestamp"
)

var (
	networkGraphSAC = sac.ForResource(resources.NetworkGraph)
)

type flowDataStoreImpl struct {
	storage store.FlowStore
}

func (fds *flowDataStoreImpl) GetAllFlows(ctx context.Context, since *types.Timestamp) ([]*storage.NetworkFlow, types.Timestamp, error) {
	// Here check for global read permission. Therefore, if flow filtering is expected,
	// caller with lower access privileges need to elevate privileges and then apply filtering.
	if ok, err := networkGraphSAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, types.Timestamp{}, err
	}

	return fds.storage.GetAllFlows(since)
}

func (fds *flowDataStoreImpl) UpsertFlows(ctx context.Context, flows []*storage.NetworkFlow, lastUpdateTS timestamp.MicroTS) error {
	if ok, err := networkGraphSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return fds.storage.UpsertFlows(flows, lastUpdateTS)
}

func (fds *flowDataStoreImpl) RemoveFlowsForDeployment(ctx context.Context, id string) error {
	// This is reached only on write access to deployment,
	// therefore no need to fetch the deployment again for access check
	if ok, err := networkGraphSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return fds.storage.RemoveFlowsForDeployment(id)
}
