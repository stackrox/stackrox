package networkpolicies

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
)

func (h *commandHandler) createApplyTx(id string) *applyTx {
	return &applyTx{
		id:               id,
		networkingClient: h.networkingV1Client,
		timestamp:        time.Now().Format(time.RFC3339),
	}
}

func (h *commandHandler) dispatchApplyCommand(cmd *central.NetworkPoliciesCommand_Apply) (*central.NetworkPoliciesResponse_Payload, error) {
	err := h.doApply(cmd)
	if err != nil {
		return nil, err
	}

	return &central.NetworkPoliciesResponse_Payload{
		Cmd: &central.NetworkPoliciesResponse_Payload_Apply{
			Apply: &central.NetworkPoliciesResponse_Apply{
				ApplyId: cmd.GetApplyId(),
			},
		},
	}, nil
}

func (h *commandHandler) doApply(cmd *central.NetworkPoliciesCommand_Apply) error {
	policies, toDelete, err := parseModification(cmd.GetModification())
	if err != nil {
		return fmt.Errorf("parsing network policy modification: %v", err)
	}

	if err := validateModification(policies, toDelete); err != nil {
		return fmt.Errorf("invalid network policy modification: %v", err)
	}

	tx := h.createApplyTx(cmd.GetApplyId())

	if err := tx.Do(policies, toDelete); err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr == nil {
			return fmt.Errorf("error applying network policies modification: %v. The old state has been restored", err)
		}
		return fmt.Errorf("error applying network policies modification: %v. Additionally, there was an error rolling back partial modifications: %v", err, rollbackErr)
	}
	return nil
}
