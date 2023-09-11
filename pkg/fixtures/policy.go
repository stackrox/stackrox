package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
)

var (
	booleanPolicy = &storage.Policy{
		Id:              "b3523d84-ac1a-4daa-a908-62d196c5a741",
		Name:            "Vulnerable Container",
		Categories:      []string{"Image Assurance", "Privileges Capabilities", "Container Configuration"},
		Description:     "Alert if the container contains vulnerabilities",
		Severity:        storage.Severity_LOW_SEVERITY,
		Rationale:       "This is the rationale",
		Remediation:     "This is the remediation",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_BUILD},
		Scope: []*storage.Scope{
			{
				Cluster:   "prod cluster",
				Namespace: "stackrox",
				Label: &storage.Scope_Label{
					Key:   "com.docker.stack.namespace",
					Value: "prevent",
				},
			},
		},
		PolicyVersion: "1.1",
		PolicySections: []*storage.PolicySection{
			{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Image Registry",
						Values: []*storage.PolicyValue{
							{
								Value: "docker.io",
							},
						},
					},
					{
						FieldName: "Image Remote",
						Values: []*storage.PolicyValue{
							{
								Value: "r/.*stackrox/nginx.*",
							},
						},
					},
					{
						FieldName: "Image Tag",
						Values: []*storage.PolicyValue{
							{
								Value: "1.10",
							},
						},
					},
					{
						FieldName: "Image Age",
						Values: []*storage.PolicyValue{
							{
								Value: "30",
							},
						},
					},
					{
						FieldName: "Dockerfile Line",
						Values: []*storage.PolicyValue{
							{
								Value: "VOLUME=/etc/*",
							},
						},
					},
					{
						FieldName: "CVE",
						Values: []*storage.PolicyValue{
							{
								Value: "CVE-1234",
							},
						},
					},
					{
						FieldName: "Image Component",
						Values: []*storage.PolicyValue{
							{
								Value: "berkeley*=.*",
							},
						},
					},
					{
						FieldName: "Image Scan Age",
						Values: []*storage.PolicyValue{
							{
								Value: "10",
							},
						},
					},
					{
						FieldName: "Environment Variable",
						Values: []*storage.PolicyValue{
							{
								Value: "UNSET=key=value",
							},
						},
					},
					{
						FieldName: "Volume Name",
						Values: []*storage.PolicyValue{
							{
								Value: "name",
							},
						},
					},
					{
						FieldName: "Volume Type",
						Values: []*storage.PolicyValue{
							{
								Value: "nfs",
							},
						},
					},
					{
						FieldName: "Volume Destination",
						Values: []*storage.PolicyValue{
							{
								Value: "/etc/network",
							},
						},
					},
					{
						FieldName: "Volume Source",
						Values: []*storage.PolicyValue{
							{
								Value: "10.0.0.1/export",
							},
						},
					},
					{
						FieldName: "Writable Mounted Volume",
						Values: []*storage.PolicyValue{
							{
								Value: "false",
							},
						},
					},
					{
						FieldName: "Port",
						Values: []*storage.PolicyValue{
							{
								Value: "8080",
							},
						},
					},
					{
						FieldName: "Protocol",
						Values: []*storage.PolicyValue{
							{
								Value: "tcp",
							},
						},
					},
					{
						FieldName: "Privileged",
						Values: []*storage.PolicyValue{
							{
								Value: "true",
							},
						},
					},
					{
						FieldName: "CVSS",
						Values: []*storage.PolicyValue{
							{
								Value: "\u003e= 5.000000",
							},
						},
					},
					{
						FieldName: "Drop Capabilities",
						Values: []*storage.PolicyValue{
							{
								Value: "DROP1",
							},
							{
								Value: "DROP2",
							},
						},
					},
					{
						FieldName: "Add Capabilities",
						Values: []*storage.PolicyValue{
							{
								Value: "ADD1",
							},
							{
								Value: "ADD2",
							},
						},
					},
				},
			},
		},
	}
)

// GetPolicy returns a Mock Policy
func GetPolicy() *storage.Policy {
	return booleanPolicy.Clone()
}

// GetPolicyWithMitre return mock Policy with MITRE ATT&CK
func GetPolicyWithMitre() *storage.Policy {
	policy := booleanPolicy.Clone()
	policy.MitreAttackVectors = []*storage.Policy_MitreAttackVectors{
		{
			Tactic:     "TA0001",
			Techniques: []string{"T1078", "T1078.001"},
		},
		{
			Tactic: "TA0003",
		},
	}
	return policy
}

// GetAuditLogEventSourcePolicy returns a Mock Policy with source set to Audit Log Event
func GetAuditLogEventSourcePolicy() *storage.Policy {
	p := booleanPolicy.Clone()
	p.EventSource = storage.EventSource_AUDIT_LOG_EVENT
	// Limit scope to things that are supported by audit log event source
	p.Scope = []*storage.Scope{
		{
			Cluster:   "prod cluster",
			Namespace: "stackrox",
		},
	}
	// Only runtime policies can have audit log event source
	p.LifecycleStages = []storage.LifecycleStage{storage.LifecycleStage_RUNTIME}
	// Switch the policy values to things related to kube events
	p.PolicySections[0].PolicyGroups = []*storage.PolicyGroup{
		{
			FieldName: "Kubernetes Resource",
			Values:    []*storage.PolicyValue{{Value: "SECRETS"}},
		},
		{
			FieldName: "Kubernetes API Verb",
			Values:    []*storage.PolicyValue{{Value: "CREATE"}},
		},
	}
	return p
}

// GetNetworkFlowPolicy returns a mock policy with criteria "Unexpected Network Flow Detected"
func GetNetworkFlowPolicy() *storage.Policy {
	return &storage.Policy{
		Id:                 fixtureconsts.NetworkPolicy1,
		Name:               "Unauthorized Network Flow",
		Description:        "This policy generates a violation for the network flows that fall outside baselines for which 'alert on anomalous violations' is set.",
		Rationale:          "The network baseline is a list of flows that are allowed, and once it is frozen, any flow outside that is a concern.",
		Remediation:        "Evaluate this network flow. If deemed to be okay, add it to the baseline. If not, investigate further as required.",
		Categories:         []string{"Anomalous Activity"},
		LifecycleStages:    []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
		Severity:           storage.Severity_HIGH_SEVERITY,
		SORTName:           "Unauthorized Network Flow",
		SORTLifecycleStage: "RUNTIME",
		PolicyVersion:      "1.1",
		PolicySections: []*storage.PolicySection{{
			PolicyGroups: []*storage.PolicyGroup{{
				FieldName: "Unexpected Network Flow Detected",
				Values: []*storage.PolicyValue{{
					Value: "true",
				}},
			}},
		}},
		EventSource: storage.EventSource_DEPLOYMENT_EVENT,
	}
}
