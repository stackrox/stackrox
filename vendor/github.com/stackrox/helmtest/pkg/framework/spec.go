package framework

import (
	"github.com/itchyny/gojq"
)

// Test defines a helmtest test. A Test can be regarded as the equivalent of the *testing.T scope of a Go unit test.
// Tests are scoped, and a test may either define concrete expectations, or contain an arbitrary number of nested tests.
// See README.md in this directory for a more detailed explanation.
type Test struct {
	// Public section - fields settable via YAML

	Name string `json:"name,omitempty"`

	Values map[string]interface{} `json:"values,omitempty"`
	Set    map[string]interface{} `json:"set,omitempty"`

	Defs         string            `json:"defs,omitempty"`
	Release      *ReleaseSpec      `json:"release,omitempty"`
	Server       *ServerSpec       `json:"server,omitempty"`
	Capabilities *CapabilitiesSpec `json:"capabilities,omitempty"`

	Expect      string `json:"expect,omitempty"`
	ExpectError *bool  `json:"expectError,omitempty"`

	Tests []*Test `json:"tests,omitempty"`

	// Private section - the following fields are never set in the YAML, they are always populated by initialize.
	parent *Test

	funcDefs   []*gojq.FuncDef
	predicates []*gojq.Query
}

// ReleaseSpec specifies how the release options for Helm will be constructed.
type ReleaseSpec struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Revision  *int   `json:"revision,omitempty"`
	IsInstall *bool  `json:"isInstall,omitempty"`
	IsUpgrade *bool  `json:"isUpgrade,omitempty"`
}

// ServerSpec specifies how the model of the server will be constructed.
type ServerSpec struct {
	// AvailableSchemas are the names of schemas that are available on the server (i.e., that rendered objects must
	// pass validation against, but not necessarily discoverable via `.Capabilities.APIVersions`).
	AvailableSchemas []string `json:"availableSchemas,omitempty"`
	// VisibleSchemas are the names of schemas that are available on the server AND discoverable via
	// `.Capabilities.APIVersions`.
	VisibleSchemas []string `json:"visibleSchemas,omitempty"`

	// NoInherit indicates that server-side settings should *not* be inherited from the enclosing scope.
	NoInherit bool `json:"noInherit,omitempty"`
}

// CapabilitiesSpec represents the `Capabilities` in Helm.
type CapabilitiesSpec struct {
	// KubeVersion represents the kubernetes version which is discoverable via `.Capabilities.KubeVersion`.
	KubeVersion *KubeVersion `json:"kubeVersion,omitempty"`
}

// KubeVersion is the Kubernetes version.
type KubeVersion struct {
	Version string `json:"version,omitempty"` // i.e. v1.18
	Major   string `json:"major,omitempty"`   // i.e. 1
	Minor   string `json:"minor,omitempty"`   // i.e. 18
}
