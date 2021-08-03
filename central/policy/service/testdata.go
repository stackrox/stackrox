package service

import (
	"github.com/stackrox/rox/generated/storage"
)

func injectMitreTestData(policy *storage.Policy) {
	policy.MitreAttackVectors = []*storage.Policy_MitreAttackVectors{
		{
			Tactic:     "TA0005",
			Techniques: []string{"T1562", "T1610"},
		},
		{
			Tactic:     "TA0006",
			Techniques: []string{"T1552"},
		},
	}
}
