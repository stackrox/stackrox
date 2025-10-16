package telemetry

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/protobuf/proto"
)

const (
	telemetryChanGCPeriod = 5 * time.Minute
	progressTimeout       = 5 * time.Minute
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

func (c *controller) streamingRequest(ctx context.Context, dataType central.PullTelemetryDataRequest_TelemetryDataType, cb telemetryCallback, opts PullKubernetesInfoOpts) error {
	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	concurrency.CancelContextOnSignal(subCtx, cancel, c.stopSig)

	requestID := uuid.NewV4().String()

	var timeoutMs int64
	if deadline, ok := subCtx.Deadline(); ok {
		timeoutMs = time.Until(deadline).Milliseconds()
		if timeoutMs <= 0 {
			return errors.New("deadline already expired")
		}
	}

	sinceTs, err := protocompat.ConvertTimeToTimestampOrError(opts.Since)
	if err != nil {
		return errors.Wrap(err, "could not convert Since timestamp")
	}
	ptdr := &central.PullTelemetryDataRequest{}
	ptdr.SetRequestId(requestID)
	ptdr.SetDataType(dataType)
	ptdr.SetTimeoutMs(timeoutMs)
	ptdr.SetSince(sinceTs)
	ptdr.SetWithComplianceOperator(opts.WithComplianceOperator)
	msg := &central.MsgToSensor{}
	msg.SetTelemetryDataRequest(proto.ValueOrDefault(ptdr))

	retC := make(chan *central.TelemetryResponsePayload, 1)
	concurrency.WithLock(&c.returnChansMutex, func() {
		c.returnChans[requestID] = retC
	})

	defer concurrency.WithLock(&c.returnChansMutex, func() {
		c.returnChans[requestID] = nil
	})

	if err := c.injector.InjectMessage(subCtx, msg); err != nil {
		return errors.Wrapf(err, "could not pull telemetry data for type %s", dataType.String())
	}

	hasEOS := false
	defer func() {
		// Check for c.supportsCancellations here as well, in order to avoid spawning a goroutine.
		if hasEOS || !c.supportsCancellations {
			return
		}

		go c.sendCancellation(requestID)
	}()

	// In case there's no progress regarding sensor sending telemetry response data,
	// we should stop waiting.
	progressTicker := time.NewTicker(progressTimeout)
	defer progressTicker.Stop()
	for {
		var resp *central.TelemetryResponsePayload
		select {
		case <-subCtx.Done():
			return errors.Wrap(subCtx.Err(), "context error")
		case <-c.stopSig.Done():
			return errors.Wrap(c.stopSig.Err(), "lost connection to sensor")
		case <-progressTicker.C:
			return errors.Errorf("sensor didn't sent any data in last %s", progressTimeout)
		case resp = <-retC:
			progressTicker.Reset(progressTimeout)
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

	cptdr := &central.CancelPullTelemetryDataRequest{}
	cptdr.SetRequestId(requestID)
	cancelMsg := &central.MsgToSensor{}
	cancelMsg.SetCancelPullTelemetryDataRequest(proto.ValueOrDefault(cptdr))

	// We don't care about any error - it can only be a context or stop error; the first is impossible because we're
	// using the background context, and in the second we're fine not sending a cancellation as the connection is going
	// away anyway.
	_ = c.injector.InjectMessage(context.Background(), cancelMsg)
}

type PullKubernetesInfoOpts struct {
	Since                  time.Time
	WithComplianceOperator bool
}

func (c *controller) PullKubernetesInfo(ctx context.Context, cb KubernetesInfoChunkCallback, opts PullKubernetesInfoOpts) error {
	genericCB := func(ctx concurrency.ErrorWaitable, chunk *central.TelemetryResponsePayload) error {
		k8sInfo := chunk.GetKubernetesInfo()
		if k8sInfo == nil {
			utils.Should(errors.New("ignoring response in telemetry data stream with missing Kubernetes info payload"))
			return nil
		}

		return cb(ctx, k8sInfo)
	}
	return c.streamingRequest(ctx, central.PullTelemetryDataRequest_KUBERNETES_INFO, genericCB, opts)
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
	return c.streamingRequest(ctx, central.PullTelemetryDataRequest_METRICS, genericCB, PullKubernetesInfoOpts{
		Since: time.Now(),
	})
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
	return c.streamingRequest(ctx, central.PullTelemetryDataRequest_CLUSTER_INFO, genericCB, PullKubernetesInfoOpts{
		Since: time.Now(),
	})
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
