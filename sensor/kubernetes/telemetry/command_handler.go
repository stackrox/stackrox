package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"runtime/pprof"
	"slices"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/k8sintrospect"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/prometheusutil"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stackrox/rox/sensor/kubernetes/telemetry/gatherers"
	"google.golang.org/protobuf/proto"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	clusterInfoChunkSize = 2 * (1 << 20) // Bytes per streaming chunk, 2MB chosen arbitrarily.

	maxK8sFileSize = 2 * (1 << 20) // maximum file size for Kubernetes files (YAMLs, logs).
)

var (
	log = logging.LoggerForModule()

	diagnosticBundleTimeout = env.DiagnosticDataCollectionTimeout.DurationSetting()
)

type commandHandler struct {
	responsesC      chan *message.ExpiringMessage
	clusterGatherer *gatherers.ClusterGatherer

	stopSig          concurrency.ErrorSignal
	centralReachable atomic.Bool

	pendingContextCancels      map[string]context.CancelFunc
	pendingContextCancelsMutex sync.Mutex
}

func (h *commandHandler) Name() string {
	return "telemetry.commandHandler"
}

// DiagnosticConfigurationFunc is a function that modifies the diagnostic configuration.
type DiagnosticConfigurationFunc func(request *central.PullTelemetryDataRequest, config k8sintrospect.Config) k8sintrospect.Config

var diagnosticConfigurationFuncs []DiagnosticConfigurationFunc

// RegisterDiagnosticConfigurationFunc registers a new function to modify the diagnostic configuration.
func RegisterDiagnosticConfigurationFunc(fn DiagnosticConfigurationFunc) {
	diagnosticConfigurationFuncs = append(diagnosticConfigurationFuncs, fn)
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
	tc := &central.TelemetryResponsePayload_ClusterInfo{}
	if chunk != nil {
		tc.SetChunk(chunk)
	}
	trp := &central.TelemetryResponsePayload{}
	trp.SetClusterInfo(proto.ValueOrDefault(tc))
	return trp
}

func (h *commandHandler) Start() error {
	return nil
}

func (h *commandHandler) Stop() {
	h.stopSig.Signal()
}

func (h *commandHandler) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e))
	switch e {
	case common.SensorComponentEventCentralReachable:
		h.centralReachable.Store(true)
	case common.SensorComponentEventOfflineMode:
		h.centralReachable.Store(false)
		h.cancelPendingRequests()
	}
}

func (h *commandHandler) Accepts(msg *central.MsgToSensor) bool {
	return msg.GetTelemetryDataRequest() != nil || msg.GetCancelPullTelemetryDataRequest() != nil
}

func (h *commandHandler) ProcessMessage(_ context.Context, msg *central.MsgToSensor) error {
	switch msg.WhichMsg() {
	case central.MsgToSensor_TelemetryDataRequest_case:
		return h.processRequest(msg.GetTelemetryDataRequest())
	case central.MsgToSensor_CancelPullTelemetryDataRequest_case:
		return h.processCancelRequest(msg.GetCancelPullTelemetryDataRequest())
	default:
		return nil
	}
}

func (h *commandHandler) processCancelRequest(req *central.CancelPullTelemetryDataRequest) error {
	requestID := req.GetRequestId()

	if requestID == "" {
		return errox.InvalidArgs.New("received invalid telemetry request with empty request ID")
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

// cancelPendingRequests cancels all pending requests currently executed by the command handler.
func (h *commandHandler) cancelPendingRequests() {
	h.pendingContextCancelsMutex.Lock()
	defer h.pendingContextCancelsMutex.Unlock()

	for reqID, cancel := range h.pendingContextCancels {
		if cancel != nil {
			log.Infof("Cancelling telemetry pull request %s due to Central connection interruption", reqID)
			delete(h.pendingContextCancels, reqID)
		}
	}
}

func (h *commandHandler) processRequest(req *central.PullTelemetryDataRequest) error {
	if req.GetRequestId() == "" {
		return errox.InvalidArgs.New("received invalid telemetry request with empty request ID")
	}
	go h.dispatchRequest(req)
	return nil
}

func (h *commandHandler) sendResponse(ctx concurrency.ErrorWaitable, resp *central.PullTelemetryDataResponse) error {
	if !h.centralReachable.Load() {
		log.Debugf("Sending telemetry response called while in offline mode, Telemetry response %s discarded",
			resp.GetRequestId())
		return nil
	}
	msg := &central.MsgFromSensor{}
	msg.SetTelemetryDataResponse(proto.ValueOrDefault(resp))
	select {
	case h.responsesC <- message.New(msg):
		return nil
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "sending pull telemetry data response")
	}
}

func (h *commandHandler) ResponsesC() <-chan *message.ExpiringMessage {
	return h.responsesC
}

func (h *commandHandler) dispatchRequest(req *central.PullTelemetryDataRequest) {
	requestID := req.GetRequestId()

	sendMsg := func(ctx concurrency.ErrorWaitable, payload *central.TelemetryResponsePayload) error {
		resp := &central.PullTelemetryDataResponse{}
		resp.SetRequestId(requestID)
		resp.SetPayload(payload)
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
	defer cancel()

	// Store the context in order to be able to react to cancellations.
	concurrency.WithLock(&h.pendingContextCancelsMutex, func() {
		h.pendingContextCancels[requestID] = cancel
	})
	defer func() {
		concurrency.WithLock(&h.pendingContextCancelsMutex, func() {
			delete(h.pendingContextCancels, requestID)
		})
	}()

	var err error
	switch req.GetDataType() {
	case central.PullTelemetryDataRequest_KUBERNETES_INFO:
		err = h.handleKubernetesInfoRequest(ctx, sendMsg, req)
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

	te := &central.TelemetryResponsePayload_EndOfStream{}
	te.SetErrorMessage(errMsg)
	eosPayload := &central.TelemetryResponsePayload{}
	eosPayload.SetEndOfStream(proto.ValueOrDefault(te))

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
	return central.TelemetryResponsePayload_builder{
		KubernetesInfo: central.TelemetryResponsePayload_KubernetesInfo_builder{
			Files: []*central.TelemetryResponsePayload_KubernetesInfo_File{
				central.TelemetryResponsePayload_KubernetesInfo_File_builder{
					Path:     file.Path,
					Contents: contents,
				}.Build(),
			},
		}.Build(),
	}.Build()
}

func (h *commandHandler) handleKubernetesInfoRequest(ctx context.Context,
	sendMsgCb func(concurrency.ErrorWaitable, *central.TelemetryResponsePayload) error,
	req *central.PullTelemetryDataRequest) error {
	subCtx, cancel := context.WithTimeout(ctx, diagnosticBundleTimeout)
	defer cancel()

	restCfg, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "could not instantiate Kubernetes REST client config")
	}

	fileCb := func(ctx concurrency.ErrorWaitable, file k8sintrospect.File) error {
		return sendMsgCb(ctx, createKubernetesPayload(file))
	}

	sinceTs, err := protocompat.ConvertTimestampToTimeOrError(req.GetSince())
	if err != nil {
		return errors.Wrap(err, "error parsing since timestamp")
	}

	cfg := k8sintrospect.DefaultConfigWithSecrets()
	for _, fn := range diagnosticConfigurationFuncs {
		cfg = fn(req, cfg)
	}

	err = k8sintrospect.Collect(subCtx, cfg, restCfg, fileCb, sinceTs)
	return errors.Wrap(err, "collecting k8s data")
}

func (h *commandHandler) handleClusterInfoRequest(ctx context.Context,
	sendMsgCb func(concurrency.ErrorWaitable, *central.TelemetryResponsePayload) error) error {
	subCtx, cancel := context.WithTimeout(ctx, diagnosticBundleTimeout)
	defer cancel()
	clusterInfo := h.clusterGatherer.Gather(subCtx)
	jsonBytes, err := json.Marshal(clusterInfo)
	if err != nil {
		return errors.Wrap(err, "marshalling cluster info")
	}
	for byteBatch := range slices.Chunk(jsonBytes, clusterInfoChunkSize) {
		if err := sendMsgCb(subCtx, makeChunk(byteBatch)); err != nil {
			return err
		}
	}
	return nil
}

func createMetricsPayload(file string, contents []byte) *central.TelemetryResponsePayload {
	return central.TelemetryResponsePayload_builder{
		MetricsInfo: central.TelemetryResponsePayload_KubernetesInfo_builder{
			Files: []*central.TelemetryResponsePayload_KubernetesInfo_File{
				central.TelemetryResponsePayload_KubernetesInfo_File_builder{
					Path:     file,
					Contents: contents,
				}.Build(),
			},
		}.Build(),
	}.Build()
}

func (h *commandHandler) handleMetricsInfoRequest(ctx context.Context,
	sendMsgCb func(concurrency.ErrorWaitable, *central.TelemetryResponsePayload) error) error {
	subCtx, cancel := context.WithTimeout(ctx, diagnosticBundleTimeout)
	defer cancel()

	fileCb := func(ctx concurrency.ErrorWaitable, file string, contents []byte) error {
		return sendMsgCb(ctx, createMetricsPayload(file, contents))
	}
	w := bytes.NewBuffer(nil)
	err := prometheusutil.ExportText(subCtx, w)
	if err != nil {
		return errors.Wrap(err, "exporting prometheus as text")
	}
	if err := fileCb(subCtx, "metrics.prom", w.Bytes()); err != nil {
		return err
	}
	w = bytes.NewBuffer(nil)
	if err := writeHeapProfile(subCtx, w); err != nil {
		return err
	}
	if err := fileCb(subCtx, "heap.pb.gz", w.Bytes()); err != nil {
		return err
	}
	return nil
}

func (h *commandHandler) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{
		centralsensor.PullTelemetryDataCap,
		centralsensor.CancelTelemetryPullCap,
		centralsensor.PullMetricsCap,
	}
}

// writeHeapProfile is a wrapper around pprof.WriteHeapProfile which respects context cancellation.
func writeHeapProfile(ctx context.Context, w io.Writer) error {
	var err error
	if ctxErr := concurrency.DoInWaitable(ctx, func() {
		err = pprof.WriteHeapProfile(w)
	}); ctxErr != nil {
		return errors.Wrap(ctxErr, "waiting on writing heap profile")
	}
	return errors.Wrap(err, "writing heap profile")
}
