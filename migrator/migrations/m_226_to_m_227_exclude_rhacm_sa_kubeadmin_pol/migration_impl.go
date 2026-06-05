package m226tom227

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	policyID = "18cbcb62-7d18-4a6c-b2ca-dd1242746943"
	rhacmSA  = "system:serviceaccount:open-cluster-management-agent-addon:config-policy-controller-sa"
)

var (
	log = logging.LoggerForModule()
)

func migrate(database *types.Databases) error {
	ctx := database.DBCtx

	var serialized []byte
	err := database.PostgresDB.QueryRow(ctx,
		"SELECT serialized FROM policies WHERE id = $1", policyID).Scan(&serialized)
	if err != nil {
		log.Warnf("policy %s not found, skipping migration: %v", policyID, err)
		return nil
	}

	policy := &storage.Policy{}
	if err := policy.UnmarshalVT(serialized); err != nil {
		return fmt.Errorf("unmarshal policy %s: %w", policyID, err)
	}

	group := findUserNameGroup(policy)
	if group == nil {
		log.Infof("policy %s has no negated 'Kubernetes User Name' group, skipping", policyID)
		return nil
	}

	if hasValue(group, rhacmSA) {
		log.Infof("policy %s already excludes RHACM SA, skipping", policyID)
		return nil
	}

	group.Values = append(group.Values, &storage.PolicyValue{Value: rhacmSA})

	updated, err := policy.MarshalVT()
	if err != nil {
		return fmt.Errorf("marshal policy %s: %w", policyID, err)
	}

	_, err = database.PostgresDB.Exec(ctx,
		"UPDATE policies SET serialized = $1 WHERE id = $2", updated, policyID)
	if err != nil {
		return fmt.Errorf("update policy %s: %w", policyID, err)
	}

	log.Infof("added RHACM SA exclusion to policy %s", policyID)
	return nil
}

func findUserNameGroup(policy *storage.Policy) *storage.PolicyGroup {
	for _, section := range policy.GetPolicySections() {
		for _, group := range section.GetPolicyGroups() {
			if group.GetFieldName() == "Kubernetes User Name" && group.GetNegate() {
				return group
			}
		}
	}
	return nil
}

func hasValue(group *storage.PolicyGroup, value string) bool {
	for _, v := range group.GetValues() {
		if v.GetValue() == value {
			return true
		}
	}
	return false
}
