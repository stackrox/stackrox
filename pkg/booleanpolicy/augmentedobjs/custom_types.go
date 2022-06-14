package augmentedobjs

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
)

// This block enumerates custom tags.
const (
	ComponentAndVersionCustomTag       = "Component And Version"
	ContainerNameCustomTag             = "Container Name"
	DockerfileLineCustomTag            = "Dockerfile Line"
	EnvironmentVarCustomTag            = "Environment Variable"
	ImageScanCustomTag                 = "Image Scan"
	ImageSignatureVerifiedCustomTag    = "Image Signature Verified By"
	HasIngressPolicyCustomTag          = "Has Ingress Network Policy"
	HasEgressPolicyCustomTag           = "Has Egress Network Policy"
	NotInNetworkBaselineCustomTag      = "Not In Network Baseline"
	NotInProcessBaselineCustomTag      = "Not In Baseline"
	KubernetesAPIVerbCustomTag         = "Kubernetes API Verb"
	KubernetesResourceCustomTag        = "Kubernetes Resource"
	KubernetesResourceNameCustomTag    = "Kubernetes Resource Name"
	KubernetesUserNameCustomTag        = "Kubernetes User Name"
	KubernetesUserGroupsCustomTag      = "Kubernetes User Groups"
	KubernetesSourceIPAddressCustomTag = "Source IP Address"
	KubernetesUserAgentCustomTag       = "User Agent"
	KubernetesIsImpersonatedCustomTag  = "Is Impersonated User"

	RuntimeClassCustomTag = "Runtime Class"
)

type dockerfileLine struct {
	Line string `search:"Dockerfile Line"`
}

type componentAndVersion struct {
	ComponentAndVersion string `search:"Component And Version"`
}

type baselineResult struct {
	NotInBaseline bool `search:"Not In Baseline"`
}

type impersonatedEventResult struct {
	IsImpersonatedUser bool `search:"Is Impersonated User"`
}

// NetworkPoliciesApplied holds the intermediate information about Network Policy presence in a cluster.
type NetworkPoliciesApplied struct {
	HasIngressNetworkPolicy bool `policy:"Has Ingress Network Policy"`
	HasEgressNetworkPolicy  bool `policy:"Has Egress Network Policy"`
	Policies                map[string]*storage.NetworkPolicy
}

// NetworkFlowDetails captures information about a particular network flow.
// Used with MatchAgainstDeploymentAndNetworkFlow to validate network flows
// Note that as of now only the field NotInNetworkBaseline is captured as a
// required field for network flow runtime checks. Please update printer.go
// if other fields are included in the future
type NetworkFlowDetails struct {
	SrcEntityName        string
	SrcEntityType        storage.NetworkEntityInfo_Type
	DstEntityName        string
	DstEntityType        storage.NetworkEntityInfo_Type
	DstPort              uint32
	L4Protocol           storage.L4Protocol
	NotInNetworkBaseline bool `policy:"Not In Network Baseline"`
	LastSeenTimestamp    *types.Timestamp
	// will only be populated if src is deployment
	SrcDeploymentNamespace string
	// will only be populated if dst is deployment
	DstDeploymentNamespace string
	// will only be populated if src is deployment
	SrcDeploymentType string
	// will only be populated if dst is deployment
	DstDeploymentType string
}

type envVar struct {
	EnvVar string `search:"Environment Variable"`
}

type imageSignatureVerification struct {
	VerifierIDs []string `search:"Image Signature Verified By"`
}
