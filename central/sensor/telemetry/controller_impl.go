package telemetry

import (
	"context"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	telemetryChanGCPeriod = 5 * time.Minute
)

type controller struct {
	stopSig concurrency.ReadOnlyErrorSignal

	returnChans      map[string]chan *central.TelemetryResponsePayload
	returnChansMutex sync.Mutex

	injector common.MessageInjector

	supportsCancellations bool
}

type telemetryCallback func(ctx concurrency.ErrorWaitable, chunk *central.TelemetryResponsePayload) error

func newController(capabilities set.Set[centralsensor.SensorCapability], injector common.MessageInjector, stopSig concurrency.ReadOnlyErrorSignal) *controller {
	ctrl := &controller{
		stopSig:               stopSig,
		returnChans:           make(map[string]chan *central.TelemetryResponsePayload),
		injector:              injector,
		supportsCancellations: capabilities.Contains(centralsensor.PullTelemetryDataCap),
	}
	go ctrl.pruneReturnChans()
	return ctrl
}

func (c *controller) streamingRequest(ctx context.Context, dataType central.PullTelemetryDataRequest_TelemetryDataType,
	cb telemetryCallback, since time.Time) error {
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	concurrency.CancelContextOnSignal(subCtx, cancel, c.stopSig)

	requestID := uuid.NewV4().String()

	var timeoutMs int64
	if deadline, ok := ctx.Deadline(); ok {
		timeoutMs = time.Until(deadline).Milliseconds()
		if timeoutMs <= 0 {
			return errors.New("deadline already expired")
		}
	}

	sinceTs, err := types.TimestampProto(since)
	if err != nil {
		return errors.Wrap(err, "could not convert since timestamp")
	}
	msg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_TelemetryDataRequest{
			TelemetryDataRequest: &central.PullTelemetryDataRequest{
				RequestId: requestID,
				DataType:  dataType,
				TimeoutMs: timeoutMs,
				Since:     sinceTs,
			},
		},
	}

	retC := make(chan *central.TelemetryResponsePayload, 1)
	concurrency.WithLock(&c.returnChansMutex, func() {
		c.returnChans[requestID] = retC
	})

	defer concurrency.WithLock(&c.returnChansMutex, func() {
		c.returnChans[requestID] = nil
	})

	if err := c.injector.InjectMessage(ctx, msg); err != nil {
		return errors.Wrap(err, "could not pull telemetry data")
	}

	hasEOS := false
	defer func() {
		// Check for c.supportsCancellations here as well, in order to avoid spawning a goroutine.
		if hasEOS || !c.supportsCancellations {
			return
		}

		go c.sendCancellation(requestID)
	}()

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
			hasEOS = true
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

func (c *controller) sendCancellation(requestID string) {
	if !c.supportsCancellations {
		return
	}

	cancelMsg := &central.MsgToSensor{
		Msg: &central.MsgToSensor_CancelPullTelemetryDataRequest{
			CancelPullTelemetryDataRequest: &central.CancelPullTelemetryDataRequest{
				RequestId: requestID,
			},
		},
	}

	// We don't care about any error - it can only be a context or stop error; the first is impossible because we're
	// using the background context, and in the second we're fine not sending a cancellation as the connection is going
	// away anyway.
	_ = c.injector.InjectMessage(context.Background(), cancelMsg)
}

func (c *controller) PullKubernetesInfo(ctx context.Context, cb KubernetesInfoChunkCallback, since time.Time) error {
	genericCB := func(ctx concurrency.ErrorWaitable, chunk *central.TelemetryResponsePayload) error {
		k8sInfo := chunk.GetKubernetesInfo()
		if k8sInfo == nil {
			utils.Should(errors.New("ignoring response in telemetry data stream with missing Kubernetes info payload"))
			return nil
		}

		return cb(ctx, k8sInfo)
	}
	return c.streamingRequest(ctx, central.PullTelemetryDataRequest_KUBERNETES_INFO, genericCB, since)
}

func (c *controller) PullMetrics(ctx context.Context, cb MetricsInfoChunkCallback) error {
	genericCB := func(ctx concurrency.ErrorWaitable, chunk *central.TelemetryResponsePayload) error {
		metricsInfo := chunk.GetMetricsInfo()
		if metricsInfo == nil {
			utils.Should(errors.New("ignoring response in telemetry data stream with missing metrics info payload"))
			return nil
		}

		return cb(ctx, metricsInfo)
	}
	return c.streamingRequest(ctx, central.PullTelemetryDataRequest_METRICS, genericCB, time.Now())
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
	return c.streamingRequest(ctx, central.PullTelemetryDataRequest_CLUSTER_INFO, genericCB, time.Now())
}

func (c *controller) ProcessTelemetryDataResponse(ctx context.Context, resp *central.PullTelemetryDataResponse) error {
	requestID := resp.GetRequestId()
	if resp.GetPayload() == nil {
		return utils.ShouldErr(errors.Errorf("received a telemetry response with an empty payload for requested ID %s", requestID))
	}

	retC, found := concurrency.WithLock2(&c.returnChansMutex, func() (chan *central.TelemetryResponsePayload, bool) {
		retC, found := c.returnChans[requestID]
		if !found {
			// Add the channel to the map to make sure log messages get throttled.
			c.returnChans[requestID] = nil
		}
		return retC, found
	})
	if retC == nil {
		if found {
			// If there is a nil entry, suppress error messages to avoid logspam.
			return nil
		}
		return errors.Errorf("could not dispatch response: no return channel registered for request id %s", requestID)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.stopSig.Done():
		return errors.Wrap(c.stopSig.Err(), "sensor connection stopped while waiting for network policies response")
	case retC <- resp.GetPayload():
		return nil
	}
}

func (c *controller) pruneReturnChans() {
	prevNilChans := set.NewStringSet()
	t := time.NewTicker(telemetryChanGCPeriod)
	defer t.Stop()

	for {
		select {
		case <-c.stopSig.Done():
			return
		case <-t.C:
		}

		// Go through all channels, and collect those that are nil. If we find a channel to be nil in two subsequent
		// iterations, that means it has been in this state for `telemetryChanGCPeriod` and now can be removed.
		newNilChans := set.NewStringSet()
		concurrency.WithLock(&c.returnChansMutex, func() {
			for id, retC := range c.returnChans {
				if retC != nil {
					continue
				}

				if prevNilChans.Contains(id) {
					delete(c.returnChans, id)
				} else {
					newNilChans.Add(id)
				}
			}
			prevNilChans = newNilChans
		})
	}
}
