package networkpolicies

import (
	"fmt"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/networkpolicies"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	networkingV1Client "k8s.io/client-go/kubernetes/typed/networking/v1"
)

var (
	log = logging.LoggerForModule()
)

type commandHandler struct {
	networkingV1Client networkingV1Client.NetworkingV1Interface

	commandsC  chan *central.NetworkPoliciesCommand
	responsesC chan *central.NetworkPoliciesResponse

	stopSig concurrency.Signal
}

// NewCommandHandler creates a new network policies command handler.
func NewCommandHandler() networkpolicies.CommandHandler {
	return newCommandHandler(client.MustCreateClientSet().NetworkingV1())
}

func newCommandHandler(networkingV1Client networkingV1Client.NetworkingV1Interface) *commandHandler {
	return &commandHandler{
		networkingV1Client: networkingV1Client,
		commandsC:          make(chan *central.NetworkPoliciesCommand),
		responsesC:         make(chan *central.NetworkPoliciesResponse),
		stopSig:            concurrency.NewSignal(),
	}
}

func (h *commandHandler) Start() {
	go h.run()
}

func (h *commandHandler) Stop() {
	h.stopSig.Signal()
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

func (h *commandHandler) SendCommand(cmd *central.NetworkPoliciesCommand) bool {
	select {
	case h.commandsC <- cmd:
		return true
	case <-h.stopSig.Done():
		return false
	}
}

func (h *commandHandler) sendResponse(resp *central.NetworkPoliciesResponse) bool {
	select {
	case h.responsesC <- resp:
		return true
	case <-h.stopSig.Done():
		return false
	}
}

func (h *commandHandler) Responses() <-chan *central.NetworkPoliciesResponse {
	return h.responsesC
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
