package networkpolicies

import (
	"time"

	"github.com/pkg/errors"
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
	ctx := h.ctx()

	policies, toDelete, err := parseModification(cmd.GetModification())
	if err != nil {
		return nil, errors.Wrap(err, "parsing network policy modification")
	}

	if err := validateModification(policies, toDelete); err != nil {
		return nil, errors.Wrap(err, "invalid network policy modification")
	}

	tx := h.createApplyTx(cmd.GetApplyId())

	if err := tx.Do(ctx, policies, toDelete); err != nil {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr == nil {
			return nil, errors.Wrap(err, "error applying network policies modification. The old state has been restored")
		}
		return nil, errors.Wrapf(rollbackErr, "error applying network policies modification: %v. Additionally, there was an error rolling back partial modifications", err)
	}

	return tx.UndoModification(), nil
}
