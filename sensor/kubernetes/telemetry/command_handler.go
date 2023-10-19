package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"runtime/pprof"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/k8sintrospect"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/prometheusutil"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/kubernetes/telemetry/gatherers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	clusterInfoChunkSize = 2 * (1 << 20) // Bytes per streaming chunk, 2MB chosen arbitrarily
	gatherTimeout        = 30 * time.Second

	maxK8sFileSize = 2 * (1 << 20) // maximum file size for Kubernetes files (YAMLs, logs)
)

var (
	log = logging.LoggerForModule()
)

type commandHandler struct {
	responsesC      chan *message.ExpiringMessage
	clusterGatherer *gatherers.ClusterGatherer

	stopSig concurrency.ErrorSignal

	pendingContextCancels      map[string]context.CancelFunc
	pendingContextCancelsMutex sync.Mutex
}

// NewCommandHandler creates a new network policies command handler.
func NewCommandHandler(client kubernetes.Interface, provider store.Provider) common.SensorComponent {
	return newCommandHandler(client, provider)
}

func newCommandHandler(k8sClient kubernetes.Interface, provider store.Provider) *commandHandler {
	return &commandHandler{
		responsesC:            make(chan *message.ExpiringMessage),
		clusterGatherer:       gatherers.NewClusterGatherer(k8sClient, provider.Deployments()),
		stopSig:               concurrency.NewErrorSignal(),
		pendingContextCancels: make(map[string]context.CancelFunc),
	}
}

func makeChunk(chunk []byte) *central.TelemetryResponsePayload {
	return &central.TelemetryResponsePayload{
		Payload: &central.TelemetryResponsePayload_ClusterInfo_{
			ClusterInfo: &central.TelemetryResponsePayload_ClusterInfo{
				Chunk: chunk,
			},
		},
	}
}

func (h *commandHandler) Start() error {
	return nil
}

func (h *commandHandler) Stop(err error) {
	if err == nil {
		err = errors.New("telemetry command handler was stopped")
	}
	h.stopSig.SignalWithError(err)
}

func (h *commandHandler) Notify(common.SensorComponentEvent) {}

func (h *commandHandler) ProcessMessage(msg *central.MsgToSensor) error {
	switch m := msg.GetMsg().(type) {
	case *central.MsgToSensor_TelemetryDataRequest:
		return h.processRequest(m.TelemetryDataRequest)
	case *central.MsgToSensor_CancelPullTelemetryDataRequest:
		return h.processCancelRequest(m.CancelPullTelemetryDataRequest)
	default:
		return nil
	}
}

func (h *commandHandler) processCancelRequest(req *central.CancelPullTelemetryDataRequest) error {
	requestID := req.GetRequestId()

	if requestID == "" {
		return errors.New("received invalid telemetry cancellation request with empty request ID")
	}

	h.pendingContextCancelsMutex.Lock()
	defer h.pendingContextCancelsMutex.Unlock()

	cancel := h.pendingContextCancels[requestID]
	if cancel != nil {
		log.Infof("Cancelling telemetry pull request %s upon request by central", requestID)
		cancel()
		delete(h.pendingContextCancels, requestID)
	}
	return nil
}

func (h *commandHandler) processRequest(req *central.PullTelemetryDataRequest) error {
	if req.GetRequestId() == "" {
		return errors.New("received invalid telemetry request with empty request ID")
	}
	go h.dispatchRequest(req)
	return nil
}

func (h *commandHandler) sendResponse(ctx concurrency.ErrorWaitable, resp *central.PullTelemetryDataResponse) error {
	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_TelemetryDataResponse{
			TelemetryDataResponse: resp,
		},
	}
	select {
	case h.responsesC <- message.New(msg):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *commandHandler) ResponsesC() <-chan *message.ExpiringMessage {
	return h.responsesC
}

func (h *commandHandler) dispatchRequest(req *central.PullTelemetryDataRequest) {
	requestID := req.GetRequestId()

	sendMsg := func(ctx concurrency.ErrorWaitable, payload *central.TelemetryResponsePayload) error {
		resp := &central.PullTelemetryDataResponse{
			RequestId: requestID,
			Payload:   payload,
		}
		return h.sendResponse(ctx, resp)
	}

	ctx := concurrency.AsContext(&h.stopSig)
	var cancel context.CancelFunc
	if req.GetTimeoutMs() > 0 {
		timeout := time.Duration(req.GetTimeoutMs()) * time.Millisecond
		ctx, cancel = context.WithTimeout(ctx, timeout)
		log.Infof("Received telemetry data request %s with a timeout of %v", requestID, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
		log.Infof("Received telemetry data request %s without a timeout", requestID)
	}

	// Store the context in order to be able to react to cancellations.
	concurrency.WithLock(&h.pendingContextCancelsMutex, func() {
		h.pendingContextCancels[requestID] = cancel
	})
	defer func() {
		cancel()
		concurrency.WithLock(&h.pendingContextCancelsMutex, func() {
			delete(h.pendingContextCancels, requestID)
		})
	}()

	time.Sleep(30 * time.Second)

	var err error
	switch req.GetDataType() {
	case central.PullTelemetryDataRequest_KUBERNETES_INFO:
		err = h.handleKubernetesInfoRequest(ctx, sendMsg, req.Since)
	case central.PullTelemetryDataRequest_CLUSTER_INFO:
		err = h.handleClusterInfoRequest(ctx, sendMsg)
	case central.PullTelemetryDataRequest_METRICS:
		err = h.handleMetricsInfoRequest(ctx, sendMsg)
	default:
		err = errors.Errorf("unknown telemetry data type %v", req.GetDataType())
	}

	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}

	eosPayload := &central.TelemetryResponsePayload{
		Payload: &central.TelemetryResponsePayload_EndOfStream_{
			EndOfStream: &central.TelemetryResponsePayload_EndOfStream{
				ErrorMessage: errMsg,
			},
		},
	}

	// Make sure we send the end-of-stream message even if the context is expired
	if err := sendMsg(&h.stopSig, eosPayload); err != nil {
		log.Errorf("Failed to send end of stream indicator for telemetry data request %s: %v", requestID, err)
	}
}

func createKubernetesPayload(file k8sintrospect.File) *central.TelemetryResponsePayload {
	contents := file.Contents
	if len(contents) > maxK8sFileSize {
		contents = contents[:maxK8sFileSize]
	}
	return &central.TelemetryResponsePayload{
		Payload: &central.TelemetryResponsePayload_KubernetesInfo_{
			KubernetesInfo: &central.TelemetryResponsePayload_KubernetesInfo{
				Files: []*central.TelemetryResponsePayload_KubernetesInfo_File{
					{
						Path:     file.Path,
						Contents: contents,
					},
				},
			},
		},
	}
}

func (h *commandHandler) handleKubernetesInfoRequest(ctx context.Context,
	sendMsgCb func(concurrency.ErrorWaitable, *central.TelemetryResponsePayload) error,
	since *types.Timestamp) error {
	restCfg, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "could not instantiate Kubernetes REST client config")
	}

	fileCb := func(ctx concurrency.ErrorWaitable, file k8sintrospect.File) error {
		return sendMsgCb(ctx, createKubernetesPayload(file))
	}

	sinceTs, err := types.TimestampFromProto(since)
	if err != nil {
		return errors.Wrap(err, "error parsing since timestamp")
	}
	return k8sintrospect.Collect(ctx, k8sintrospect.DefaultConfigWithSecrets(), restCfg, fileCb, sinceTs)
}

func (h *commandHandler) handleClusterInfoRequest(ctx context.Context, sendMsgCb func(concurrency.ErrorWaitable, *central.TelemetryResponsePayload) error) error {
	subCtx, cancel := context.WithTimeout(ctx, gatherTimeout)
	defer cancel()
	clusterInfo := h.clusterGatherer.Gather(subCtx)
	jsonBytes, err := json.Marshal(clusterInfo)
	if err != nil {
		return err
	}
	batchManager := batcher.New(len(jsonBytes), clusterInfoChunkSize)
	for {
		start, end, ok := batchManager.Next()
		if !ok {
			break
		}
		if err := sendMsgCb(subCtx, makeChunk(jsonBytes[start:end])); err != nil {
			return err
		}
	}
	return nil
}

func createMetricsPayload(file string, contents []byte) *central.TelemetryResponsePayload {
	return &central.TelemetryResponsePayload{
		Payload: &central.TelemetryResponsePayload_MetricsInfo{
			MetricsInfo: &central.TelemetryResponsePayload_KubernetesInfo{
				Files: []*central.TelemetryResponsePayload_KubernetesInfo_File{
					{
						Path:     file,
						Contents: contents,
					},
				},
			},
		},
	}
}

func (h *commandHandler) handleMetricsInfoRequest(ctx context.Context, sendMsgCb func(concurrency.ErrorWaitable, *central.TelemetryResponsePayload) error) error {
	subCtx, cancel := context.WithTimeout(ctx, gatherTimeout)
	defer cancel()

	fileCb := func(ctx concurrency.ErrorWaitable, file string, contents []byte) error {
		return sendMsgCb(ctx, createMetricsPayload(file, contents))
	}
	w := bytes.NewBuffer(nil)
	err := prometheusutil.ExportText(w)
	if err != nil {
		return err
	}
	if err := fileCb(subCtx, "metrics.prom", w.Bytes()); err != nil {
		return err
	}
	w = bytes.NewBuffer(nil)
	if err := pprof.WriteHeapProfile(w); err != nil {
		return err
	}
	if err := fileCb(subCtx, "heap.pb.gz", w.Bytes()); err != nil {
		return err
	}
	return nil
}

func (h *commandHandler) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.PullTelemetryDataCap, centralsensor.CancelTelemetryPullCap, centralsensor.PullMetricsCap}
}
