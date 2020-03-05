package telemetry

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/k8sintrospect"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
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
	responsesC      chan *central.MsgFromSensor
	clusterGatherer *gatherers.ClusterGatherer

	stopSig concurrency.ErrorSignal
}

// NewCommandHandler creates a new network policies command handler.
func NewCommandHandler() common.SensorComponent {
	return newCommandHandler(client.MustCreateClientSet())
}

func newCommandHandler(k8sClient kubernetes.Interface) *commandHandler {
	return &commandHandler{
		responsesC:      make(chan *central.MsgFromSensor),
		clusterGatherer: gatherers.NewClusterGatherer(k8sClient, resources.DeploymentStoreSingleton()),
		stopSig:         concurrency.NewErrorSignal(),
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

func (h *commandHandler) ProcessMessage(msg *central.MsgToSensor) error {
	telemetryReq := msg.GetTelemetryDataRequest()
	if telemetryReq == nil {
		return nil
	}
	return h.processRequest(telemetryReq)
}

func (h *commandHandler) processRequest(req *central.PullTelemetryDataRequest) error {
	if req.GetRequestId() == "" {
		return errors.New("received invalid telemetry request with empty request ID")
	}
	go h.dispatchRequest(req)
	return nil
}

func (h *commandHandler) sendResponse(resp *central.PullTelemetryDataResponse) error {
	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_TelemetryDataResponse{
			TelemetryDataResponse: resp,
		},
	}
	select {
	case h.responsesC <- msg:
		return nil
	case <-h.stopSig.Done():
		return h.stopSig.Err()
	}
}

func (h *commandHandler) ResponsesC() <-chan *central.MsgFromSensor {
	return h.responsesC
}

func (h *commandHandler) dispatchRequest(req *central.PullTelemetryDataRequest) {
	requestID := req.GetRequestId()

	sendMsg := func(payload *central.TelemetryResponsePayload) error {
		resp := &central.PullTelemetryDataResponse{
			RequestId: requestID,
			Payload:   payload,
		}
		return h.sendResponse(resp)
	}

	var err error
	switch req.GetDataType() {
	case central.PullTelemetryDataRequest_KUBERNETES_INFO:
		err = h.handleKubernetesInfoRequest(sendMsg)
	case central.PullTelemetryDataRequest_CLUSTER_INFO:
		err = h.handleClusterInfoRequest(sendMsg)
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

	if err := sendMsg(eosPayload); err != nil {
		log.Errorf("Failed to send end of stream indicator for telemetry data request %s: %v", requestID, err)
	}
}

func (h *commandHandler) handleKubernetesInfoRequest(sendMsgCb func(*central.TelemetryResponsePayload) error) error {
	restCfg, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "could not instantiate Kubernetes REST client config")
	}

	fileCb := func(_ concurrency.ErrorWaitable, file k8sintrospect.File) error {
		contents := file.Contents
		if len(contents) > maxK8sFileSize {
			contents = contents[:maxK8sFileSize]
		}
		payload := &central.TelemetryResponsePayload{
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
		return sendMsgCb(payload)
	}

	return k8sintrospect.Collect(&h.stopSig, k8sintrospect.DefaultConfig, restCfg, fileCb)
}

func (h *commandHandler) handleClusterInfoRequest(sendMsgCb func(*central.TelemetryResponsePayload) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), gatherTimeout)
	defer cancel()
	clusterInfo := h.clusterGatherer.Gather(ctx)
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
		if err := sendMsgCb(makeChunk(jsonBytes[start:end])); err != nil {
			return err
		}
	}
	return nil
}

func (h *commandHandler) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.PullTelemetryDataCap}
}
