package networkpolicies

import (
	"context"
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"k8s.io/client-go/kubernetes"
	networkingV1Client "k8s.io/client-go/kubernetes/typed/networking/v1"
)

var (
	log = logging.LoggerForModule()
)

type commandHandler struct {
	networkingV1Client networkingV1Client.NetworkingV1Interface

	commandsC  chan *central.NetworkPoliciesCommand
	responsesC chan *message.ExpiringMessage

	stopSig concurrency.Signal
}

// NewCommandHandler creates a new network policies command handler.
func NewCommandHandler(client kubernetes.Interface) common.SensorComponent {
	return newCommandHandler(client.NetworkingV1())
}

func newCommandHandler(networkingV1Client networkingV1Client.NetworkingV1Interface) *commandHandler {
	return &commandHandler{
		networkingV1Client: networkingV1Client,
		commandsC:          make(chan *central.NetworkPoliciesCommand),
		responsesC:         make(chan *message.ExpiringMessage),
		stopSig:            concurrency.NewSignal(),
	}
}

func (h *commandHandler) Start() error {
	go h.run()
	return nil
}

func (h *commandHandler) Stop(_ error) {
	h.stopSig.Signal()
}

func (h *commandHandler) Notify(common.SensorComponentEvent) {}

func (h *commandHandler) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (h *commandHandler) ResponsesC() <-chan *message.ExpiringMessage {
	return h.responsesC
}

func (h *commandHandler) run() {
	defer close(h.responsesC)

	for !h.stopSig.IsDone() {
		select {
		case cmd, ok := <-h.commandsC:
			if !ok {
				log.Panic("Command handler channel closed unexpectedly")
			}

			if !h.processCommand(cmd) {
				log.Errorf("Could not send response for network policies command %+v", cmd)
			}
		case <-h.stopSig.Done():
			return
		}
	}
}

func (h *commandHandler) ProcessMessage(msg *central.MsgToSensor) error {
	cmd := msg.GetNetworkPoliciesCommand()
	if cmd == nil {
		return nil
	}
	select {
	case h.commandsC <- cmd:
		return nil
	case <-h.stopSig.Done():
		return errors.Errorf("unable to apply network policies: %s", proto.MarshalTextString(cmd))
	}
}

func (h *commandHandler) sendResponse(resp *central.NetworkPoliciesResponse) bool {
	select {
	case h.responsesC <- message.New(&central.MsgFromSensor{
		Msg: &central.MsgFromSensor_NetworkPoliciesResponse{NetworkPoliciesResponse: resp},
	}):
		return true
	case <-h.stopSig.Done():
		return false
	}
}

func (h *commandHandler) processCommand(cmd *central.NetworkPoliciesCommand) bool {
	payload, err := h.dispatchCommand(cmd)
	if err != nil {
		payload = &central.NetworkPoliciesResponse_Payload{
			Cmd: &central.NetworkPoliciesResponse_Payload_Error{
				Error: &central.NetworkPoliciesResponse_Error{
					Message: err.Error(),
				},
			},
		}
	}

	resp := &central.NetworkPoliciesResponse{
		SeqId:   cmd.GetSeqId(),
		Payload: payload,
	}

	return h.sendResponse(resp)
}

func (h *commandHandler) dispatchCommand(cmd *central.NetworkPoliciesCommand) (*central.NetworkPoliciesResponse_Payload, error) {
	switch c := cmd.GetPayload().GetCmd().(type) {
	case *central.NetworkPoliciesCommand_Payload_Apply:
		return h.dispatchApplyCommand(c.Apply)
	default:
		return nil, fmt.Errorf("unknown network policy command of type %T", c)
	}
}

func (h *commandHandler) ctx() context.Context {
	return concurrency.AsContext(&h.stopSig)
}
