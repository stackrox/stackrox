package complianceoperator

const (
	// Name is the default compliance operator name.
	Name = "compliance-operator"
)

// DefaultNodeRoles returns the backward-compatible default node roles used when
// a compliance scan configuration does not specify any.
// Keep in sync with the UI default in ui/.../Schedules/compliance.scanConfigs.utils.tsx.
func DefaultNodeRoles() []string {
	return []string{"master", "worker"}
}
