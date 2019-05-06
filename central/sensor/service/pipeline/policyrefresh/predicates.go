package policyrefresh

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

type predicate struct {
	messagePred func(*central.MsgFromSensor) bool
	policyPred  func(*storage.Policy) bool
}

func predicatesForMessage(msg *central.MsgFromSensor) []*predicate {
	ret := make([]*predicate, 0)
	for _, predicate := range msgAndPolicyPredicates {
		if predicate.messagePred(msg) {
			ret = append(ret, predicate)
		}
	}
	return ret
}

var msgAndPolicyPredicates = []*predicate{
	// RBAC data requires RBAC affected policies are updated
	{
		messagePred: func(msg *central.MsgFromSensor) bool {
			return msg.GetEvent().GetBinding() != nil ||
				msg.GetEvent().GetRole() != nil ||
				msg.GetEvent().GetServiceAccount() != nil
		},
		policyPred: func(policy *storage.Policy) bool {
			return policy.GetFields().GetPermissionPolicy().GetPermissionLevel() != storage.PermissionLevel_UNSET
		},
	},
}
