package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/networkflow/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/timestamp"
)

type flowDataStoreImpl struct {
	storage store.FlowStore
}

func (fds *flowDataStoreImpl) GetAllFlows(_ context.Context, since *types.Timestamp) ([]*storage.NetworkFlow, types.Timestamp, error) {
	return fds.storage.GetAllFlows(since)
}

func (fds *flowDataStoreImpl) GetFlow(_ context.Context, props *storage.NetworkFlowProperties) (*storage.NetworkFlow, error) {
	return fds.storage.GetFlow(props)
}

func (fds *flowDataStoreImpl) UpsertFlows(_ context.Context, flows []*storage.NetworkFlow, lastUpdateTS timestamp.MicroTS) error {
	return fds.storage.UpsertFlows(flows, lastUpdateTS)
}

func (fds *flowDataStoreImpl) RemoveFlow(_ context.Context, props *storage.NetworkFlowProperties) error {
	return fds.storage.RemoveFlow(props)
}

func (fds *flowDataStoreImpl) RemoveFlowsForDeployment(_ context.Context, id string) error {
	return fds.storage.RemoveFlowsForDeployment(id)
}
