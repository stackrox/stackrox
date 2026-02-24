package views

import "github.com/stackrox/rox/generated/storage"

// PolicyNameAndSeverity is a lightweight projection of alert data containing only
// the policy name and severity. Used by risk scoring to avoid deserializing full
// alert protobuf blobs when only these two fields are needed.
type PolicyNameAndSeverity struct {
	PolicyName string `db:"policy"`
	Severity   int    `db:"severity"`
}

// GetPolicyName returns the policy name.
func (p *PolicyNameAndSeverity) GetPolicyName() string {
	return p.PolicyName
}

// GetSeverity returns the severity as a storage.Severity enum value.
func (p *PolicyNameAndSeverity) GetSeverity() storage.Severity {
	return storage.Severity(p.Severity)
}
