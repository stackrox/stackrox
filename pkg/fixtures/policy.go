package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
)

var (
	booleanPolicy = storage.Policy_builder{
		Id:              "b3523d84-ac1a-4daa-a908-62d196c5a741",
		Name:            "Vulnerable Container",
		Categories:      []string{"Image Assurance", "Privileges Capabilities", "Container Configuration"},
		Description:     "Alert if the container contains vulnerabilities",
		Severity:        storage.Severity_LOW_SEVERITY,
		Rationale:       "This is the rationale",
		Remediation:     "This is the remediation",
		LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_BUILD},
		Scope: []*storage.Scope{
			storage.Scope_builder{
				Cluster:   "prod cluster",
				Namespace: "stackrox",
				Label: storage.Scope_Label_builder{
					Key:   "com.docker.stack.namespace",
					Value: "prevent",
				}.Build(),
			}.Build(),
		},
		PolicyVersion: "1.1",
		PolicySections: []*storage.PolicySection{
			storage.PolicySection_builder{
				PolicyGroups: []*storage.PolicyGroup{
					storage.PolicyGroup_builder{
						FieldName: "Image Registry",
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "docker.io",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "Image Remote",
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "r/.*stackrox/nginx.*",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "Image Tag",
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "1.10",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "Image Age",
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "30",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "Dockerfile Line",
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "VOLUME=/etc/*",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "CVE",
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "CVE-1234",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "Image Component",
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "berkeley*=.*",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "Image Scan Age",
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "10",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "Environment Variable",
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "UNSET=key=value",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "Volume Name",
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "name",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "Volume Type",
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "nfs",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "Volume Destination",
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "/etc/network",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "Volume Source",
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "10.0.0.1/export",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "Writable Mounted Volume",
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "false",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "Port",
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "8080",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "Protocol",
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "tcp",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "Privileged",
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "true",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "CVSS",
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "\u003e= 5.000000",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "Drop Capabilities",
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "DROP1",
							}.Build(),
							storage.PolicyValue_builder{
								Value: "DROP2",
							}.Build(),
						},
					}.Build(),
					storage.PolicyGroup_builder{
						FieldName: "Add Capabilities",
						Values: []*storage.PolicyValue{
							storage.PolicyValue_builder{
								Value: "ADD1",
							}.Build(),
							storage.PolicyValue_builder{
								Value: "ADD2",
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
		},
	}.Build()
)

// GetPolicy returns a Mock Policy
func GetPolicy() *storage.Policy {
	return booleanPolicy.CloneVT()
}

// GetPolicyWithMitre return mock Policy with MITRE ATT&CK
func GetPolicyWithMitre() *storage.Policy {
	policy := booleanPolicy.CloneVT()
	pm := &storage.Policy_MitreAttackVectors{}
	pm.SetTactic("TA0001")
	pm.SetTechniques([]string{"T1078", "T1078.001"})
	pm2 := &storage.Policy_MitreAttackVectors{}
	pm2.SetTactic("TA0003")
	policy.SetMitreAttackVectors([]*storage.Policy_MitreAttackVectors{
		pm,
		pm2,
	})
	return policy
}

// GetAuditLogEventSourcePolicy returns a Mock Policy with source set to Audit Log Event
func GetAuditLogEventSourcePolicy() *storage.Policy {
	p := booleanPolicy.CloneVT()
	p.SetEventSource(storage.EventSource_AUDIT_LOG_EVENT)
	// Limit scope to things that are supported by audit log event source
	scope := &storage.Scope{}
	scope.SetCluster("prod cluster")
	scope.SetNamespace("stackrox")
	p.SetScope([]*storage.Scope{
		scope,
	})
	// Only runtime policies can have audit log event source
	p.SetLifecycleStages([]storage.LifecycleStage{storage.LifecycleStage_RUNTIME})
	// Switch the policy values to things related to kube events
	p.GetPolicySections()[0].SetPolicyGroups([]*storage.PolicyGroup{
		storage.PolicyGroup_builder{
			FieldName: "Kubernetes Resource",
			Values:    []*storage.PolicyValue{storage.PolicyValue_builder{Value: "SECRETS"}.Build()},
		}.Build(),
		storage.PolicyGroup_builder{
			FieldName: "Kubernetes API Verb",
			Values:    []*storage.PolicyValue{storage.PolicyValue_builder{Value: "CREATE"}.Build()},
		}.Build(),
	})
	return p
}

// GetNetworkFlowPolicy returns a mock policy with criteria "Unexpected Network Flow Detected"
func GetNetworkFlowPolicy() *storage.Policy {
	return storage.Policy_builder{
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
		PolicySections: []*storage.PolicySection{storage.PolicySection_builder{
			PolicyGroups: []*storage.PolicyGroup{storage.PolicyGroup_builder{
				FieldName: "Unexpected Network Flow Detected",
				Values: []*storage.PolicyValue{storage.PolicyValue_builder{
					Value: "true",
				}.Build()},
			}.Build()},
		}.Build()},
		EventSource: storage.EventSource_DEPLOYMENT_EVENT,
	}.Build()
}
