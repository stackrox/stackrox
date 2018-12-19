package sensornetworkflow

import (
	"fmt"

	"github.com/stackrox/rox/central/metrics"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type handler struct {
	clusterID string
	updater   flowStoreUpdater
	stream    Stream
}

func (h *handler) Run() error {
	for {
		update, err := h.stream.Recv()
		if err != nil {
			return fmt.Errorf("receiving message: %v", err)
		}
		if len(update.Updated) == 0 {
			return status.Errorf(codes.Internal, "received empty updated flows")
		}

		metrics.IncrementTotalNetworkFlowsReceivedCounter(h.clusterID, len(update.Updated))
		if err = h.updater.update(update.Updated, update.Time); err != nil {
			return status.Errorf(codes.Internal, err.Error())
		}
	}
}
