package service

import (
	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
)

func getMockK8sListAlert() *storage.ListAlert {
	return &storage.ListAlert{
		Id:             "k8s-event-mock-alert-1",
		LifecycleStage: storage.LifecycleStage_RUNTIME,
		Time:           ptypes.TimestampNow(),
		Policy: &storage.ListAlertPolicy{
			Id:         "8ab0f199-4904-4808-9461-3501da1d1b77",
			Name:       "Kubectl Exec into Pod",
			Severity:   storage.Severity_HIGH_SEVERITY,
			Categories: []string{"Kubernetes Events"},
		},
		Deployment: &storage.ListAlertDeployment{
			Id:          "dep-1",
			Name:        "dep-1",
			ClusterName: "cluster-1",
			Namespace:   "ns-1",
		},
		State: storage.ViolationState_ACTIVE,
	}
}

func getMockK8sAlert() *storage.Alert {
	return &storage.Alert{
		Id: "k8s-event-mock-alert-1",
		Violations: []*storage.Alert_Violation{
			{
				Message: "Kubernetes API received exec 'cmd' request into pod 'pod' container 'container'",
				MessageAttributes: &storage.Alert_Violation_KeyValueAttrs_{
					KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
						Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
							{Key: "pod", Value: "pod"},
							{Key: "container", Value: "container"},
							{Key: "command", Value: "cmd"},
						},
					},
				},
			},
			{
				Message: "This is another violation",
			},
		},
		Time: ptypes.TimestampNow(),
		Policy: &storage.Policy{
			Id:            "8ab0f199-4904-4808-9461-3501da1d1b77",
			Name:          "Kubectl Exec into Pod",
			Severity:      storage.Severity_HIGH_SEVERITY,
			Categories:    []string{"Kubernetes Events"},
			Rationale:     "'pods/exec' is non-standard approach for interacting with containers. Attackers with permissions could execute malicious code and compromise resources within a cluster",
			Remediation:   "Restrict RBAC access to the 'pods/exec' resource according to the Principle of Least Privilege. Limit such usage only to development, testing or debugging (non-production) activities",
			PolicyVersion: "1",
			PolicySections: []*storage.PolicySection{
				{
					PolicyGroups: []*storage.PolicyGroup{
						{
							FieldName:       "Kubernetes API Verb",
							BooleanOperator: storage.BooleanOperator_OR,
							Negate:          false,
							Values: []*storage.PolicyValue{
								{
									Value: "CREATE",
								},
							},
						},
						{
							FieldName:       "Kubernetes Resource",
							BooleanOperator: storage.BooleanOperator_OR,
							Negate:          false,
							Values: []*storage.PolicyValue{
								{
									Value: "PODS_EXEC",
								},
							},
						},
					},
				},
			},
		},
		Entity: &storage.Alert_Deployment_{
			Deployment: &storage.Alert_Deployment{
				Id:          "dep-1",
				Name:        "dep-1",
				ClusterName: "cluster-1",
				Namespace:   "ns-1",
			},
		},
	}
}
