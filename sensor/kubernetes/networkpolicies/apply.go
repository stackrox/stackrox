package networkpolicies

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

func (h *commandHandler) createApplyTx(id string) *applyTx {
	return &applyTx{
		id:               id,
		networkingClient: h.networkingV1Client,
		timestamp:        time.Now().Format(time.RFC3339),
	}
}

func (h *commandHandler) dispatchApplyCommand(cmd *central.NetworkPoliciesCommand_Apply) (*central.NetworkPoliciesResponse_Payload, error) {
	undoMod, err := h.doApply(cmd)
	if err != nil {
		return nil, err
	}

	return &central.NetworkPoliciesResponse_Payload{
		Cmd: &central.NetworkPoliciesResponse_Payload_Apply{
			Apply: &central.NetworkPoliciesResponse_Apply{
				ApplyId:          cmd.GetApplyId(),
				UndoModification: undoMod,
			},
		},
	}, nil
}

func (h *commandHandler) doApply(cmd *central.NetworkPoliciesCommand_Apply) (*storage.NetworkPolicyModification, error) {
	policies, toDelete, err := parseModification(cmd.GetModification())
	if err != nil {
		return nil, fmt.Errorf("parsing network policy modification: %v", err)
	}

	if err := validateModification(policies, toDelete); err != nil {
		return nil, fmt.Errorf("invalid network policy modification: %v", err)
	}

	tx := h.createApplyTx(cmd.GetApplyId())

	if err := tx.Do(policies, toDelete); err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr == nil {
			return nil, fmt.Errorf("error applying network policies modification: %v. The old state has been restored", err)
		}
		return nil, fmt.Errorf("error applying network policies modification: %v. Additionally, there was an error rolling back partial modifications: %v", err, rollbackErr)
	}

	return tx.UndoModification(), nil
}
