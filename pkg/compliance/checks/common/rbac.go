package common

import (
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
)

// IsRBACConfiguredCorrectlyInterpretation is the text that explains how MasterAPIServerRBACConfigurationCommandLine works.
const IsRBACConfiguredCorrectlyInterpretation = `StackRox assesses how the Kubernetes API server is configured in your clusters.
For this control, StackRox checks that the legacy Application-Based Access Control (ABAC) authorizer is disabled and the more secure Role-Based Access Control (RBAC) authorizer is enabled.`

// MasterAPIServerRBACConfigurationCommandLine checks whether the master API server process has RBAC configured correctly
func MasterAPIServerRBACConfigurationCommandLine() *standards.CheckAndMetadata {
	checkAndMetadata := MasterAPIServerCommandLine("authorization-mode", "RBAC", "RBAC", Contains)
	checkAndMetadata.Metadata.InterpretationText = IsRBACConfiguredCorrectlyInterpretation
	return checkAndMetadata
}
