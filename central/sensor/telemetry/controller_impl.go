package telemetry

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
)

type controller struct {
	stopSig concurrency.ReadOnlyErrorSignal

	returnChans      map[string]chan *central.TelemetryResponsePayload
	returnChansMutex sync.Mutex

	injector common.MessageInjector
}

type telemetryCallback func(ctx concurrency.ErrorWaitable, chunk *central.TelemetryResponsePayload) error

func newController(injector common.MessageInjector, stopSig concurrency.ReadOnlyErrorSignal) *controller {
	return &controller{
		stopSig:     stopSig,
		returnChans: make(map[string]chan *central.TelemetryResponsePayload),
		injector:    injector,
	}
}

func (c *controller) streamingRequest(ctx context.Context, dataType central.PullTelemetryDataRequest_TelemetryDataType, cb telemetryCallback) error {
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	concurrency.CancelContextOnSignal(subCtx, cancel, c.stopSig)

	requestID := uuid.NewV4().String()

	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_TelemetryDataRequest{
			TelemetryDataRequest: &central.PullTelemetryDataRequest{
				RequestId: requestID,
				DataType:  dataType,
			},
		},
	}

	retC := make(chan *central.TelemetryResponsePayload, 1)
	concurrency.WithLock(&c.returnChansMutex, func() {
		c.returnChans[requestID] = retC
	})

	defer concurrency.WithLock(&c.returnChansMutex, func() {
		delete(c.returnChans, requestID)
	})

	if err := c.injector.InjectMessage(ctx, msg); err != nil {
		return errors.Wrap(err, "could not pull telemetry data")
	}

	for {
		var resp *central.TelemetryResponsePayload
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "context error")
		case <-c.stopSig.Done():
			return errors.Wrap(c.stopSig.Err(), "lost connection to sensor")
		case resp = <-retC:
		}

		if eos := resp.GetEndOfStream(); eos != nil {
			if eos.GetErrorMessage() != "" {
				return errors.New(eos.GetErrorMessage())
			}
			return nil
		}

		if err := cb(subCtx, resp); err != nil {
			return err
		}
	}
}

func (c *controller) PullKubernetesInfo(ctx context.Context, cb KubernetesInfoChunkCallback) error {
	genericCB := func(ctx concurrency.ErrorWaitable, chunk *central.TelemetryResponsePayload) error {
		k8sInfo := chunk.GetKubernetesInfo()
		if k8sInfo == nil {
			utils.Should(errors.New("ignoring response in telemetry data stream with missing Kubernetes info payload"))
			return nil
		}

		return cb(ctx, k8sInfo)
	}
	return c.streamingRequest(ctx, central.PullTelemetryDataRequest_KUBERNETES_INFO, genericCB)
}

func (c *controller) PullClusterInfo(ctx context.Context, cb ClusterInfoCallback) error {
	genericCB := func(ctx concurrency.ErrorWaitable, chunk *central.TelemetryResponsePayload) error {
		clusterInfo := chunk.GetClusterInfo()
		if clusterInfo == nil {
			utils.Should(errors.New("ignoring response in telemetry data stream with missing Cluster info payload"))
			return nil
		}

		return cb(ctx, clusterInfo)
	}
	return c.streamingRequest(ctx, central.PullTelemetryDataRequest_CLUSTER_INFO, genericCB)
}

func (c *controller) ProcessTelemetryDataResponse(resp *central.PullTelemetryDataResponse) error {
	requestID := resp.GetRequestId()
	if resp.GetPayload() == nil {
		return utils.Should(errors.Errorf("received a telemetry response with an empty payload for requested ID %s", requestID))
	}

	var retC chan *central.TelemetryResponsePayload
	concurrency.WithLock(&c.returnChansMutex, func() {
		retC = c.returnChans[requestID]
	})
	if retC == nil {
		return errors.Errorf("could not dispatch response: no return channel registered for request id %s", requestID)
	}

	select {
	case <-c.stopSig.Done():
		return errors.Wrap(c.stopSig.Err(), "sensor connection stopped while waiting for network policies response")
	case retC <- resp.GetPayload():
		return nil
	}
}
