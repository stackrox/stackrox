//go:build sql_integration

package splunk

// This file contains tests for /violations endpoint (mostly).

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/violationmessages/printer"
	"github.com/stackrox/rox/pkg/httputil/mock"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"k8s.io/client-go/util/jsonpath"
)

var (
	// In case later we'd need to make adjustments or take different samples, the following structs were dumped from a
	// live system by simply using "github.com/mitranim/repr" module and printing them with `repr.Println(alert)` call.
	// Next, I removed some redundant `&` operators that compiler did not like, adjusted enums to use symbols (such as
	// storage.Severity_HIGH_SEVERITY) instead of integer values (e.g. 3) and made timestamps look human-friendly by
	// means of makeTimestamp() calls.

	deployAlert = storage.Alert{
		Id: "f56ffae8-adf9-4983-8e56-e260f1ab3dc9",
		Policy: &storage.Policy{
			Id:          "2db9a279-2aec-4618-a85d-7f1bdf4911b1",
			Name:        "90-Day Image Age",
			Description: "Alert on deployments with images that haven't been updated in 90 days",
			Rationale:   "Base images are updated frequently with bug fixes and vulnerability patches. Image age exceeding 90 days may indicate a higher risk of vulnerabilities existing in the image.",
			Remediation: "Rebuild your image, push a new minor version (with a new immutable tag), and update your service to use it",
			Categories: []string{
				"DevOps Best Practices",
				"Security Best Practices",
			},
			LifecycleStages: []storage.LifecycleStage{storage.LifecycleStage_BUILD, storage.LifecycleStage_DEPLOY},
			Exclusions: []*storage.Exclusion{{
				Name: "Don't alert on kube-system namespace",
				Deployment: &storage.Exclusion_Deployment{
					Scope: &storage.Scope{
						Namespace: "kube-system",
					},
				},
			}, {
				Name: "Don't alert on istio-system namespace",
				Deployment: &storage.Exclusion_Deployment{
					Scope: &storage.Scope{
						Namespace: "istio-system",
					},
				},
			}},
			Severity:           storage.Severity_LOW_SEVERITY,
			SORTName:           "90-Day Image Age",
			SORTLifecycleStage: "BUILD,DEPLOY",
			PolicyVersion:      "1.1",
			PolicySections: []*storage.PolicySection{{
				PolicyGroups: []*storage.PolicyGroup{{
					FieldName: "Image Age",
					Values: []*storage.PolicyValue{{
						Value: "90",
					}},
				}},
			}},
		},
		Entity: &storage.Alert_Deployment_{
			Deployment: &storage.Alert_Deployment{
				Id:          "565bdd7a-eb3e-4367-9b73-87a9bcb8f4e7",
				Name:        "monitoring",
				Type:        "Deployment",
				Namespace:   "stackrox",
				NamespaceId: "dff6a17e-f246-4dc0-98b3-c70ee59c3cea",
				Labels: map[string]string{
					"app":                          "monitoring",
					"app.kubernetes.io/managed-by": "Helm",
					"app.kubernetes.io/name":       "stackrox",
				},
				ClusterId:   "098e0e05-a96b-43ca-95af-3ef72cd32828",
				ClusterName: "remote",
				Containers: []*storage.Alert_Deployment_Container{{
					Image: &storage.ContainerImage{
						Id: "sha256:488ce940267b9b7e281779845d45c6aef36774ed4ca54b2aef67104bf70dee23",
						Name: &storage.ImageName{
							Registry: "docker.io",
							Remote:   "stackrox/monitoring",
							Tag:      "1.0.0",
							FullName: "docker.io/stackrox/monitoring:1.0.0",
						},
					},
					Name: "grafana",
				}, {
					Image: &storage.ContainerImage{
						Id: "sha256:488ce940267b9b7e281779845d45c6aef36774ed4ca54b2aef67104bf70dee23",
						Name: &storage.ImageName{
							Registry: "docker.io",
							Remote:   "stackrox/monitoring",
							Tag:      "1.0.0",
							FullName: "docker.io/stackrox/monitoring:1.0.0",
						},
					},
					Name: "influxdb",
				}, {
					Image: &storage.ContainerImage{
						Id: "sha256:488ce940267b9b7e281779845d45c6aef36774ed4ca54b2aef67104bf70dee23",
						Name: &storage.ImageName{
							Registry: "docker.io",
							Remote:   "stackrox/monitoring",
							Tag:      "1.0.0",
							FullName: "docker.io/stackrox/monitoring:1.0.0",
						},
					},
					Name: "telegraf-proxy",
				}},
				Annotations: map[string]string{
					"owner":                          "stackrox",
					"email":                          "support@stackrox.com",
					"meta.helm.sh/release-name":      "stackrox-monitoring",
					"meta.helm.sh/release-namespace": "stackrox",
				},
			},
		},
		Violations: []*storage.Alert_Violation{{
			Message: "Container 'grafana' has image created at 2020-09-28 17:03:00 (UTC)",
		}, {
			Message: "Container 'influxdb' has image created at 2020-09-28 17:03:00 (UTC)",
		}, {
			Message: "Container 'telegraf-proxy' has image created at 2020-09-28 17:03:00 (UTC)",
		}},
		Time:          makeTimestamp("2021-02-01T16:09:02.193352817Z"),
		FirstOccurred: makeTimestamp("2021-02-01T16:09:02.128791072Z"),
	}

	processAlert = storage.Alert{
		Id: "f2d0efaa-2c54-402c-aeed-5b88ed5ccb8a",
		Policy: &storage.Policy{
			Id:          "f0bacecd-87be-4f51-89a5-8f86ad523620",
			Name:        "nmap Execution",
			Description: "Alerts when the nmap process launches in a container during run time",
			Rationale:   "Nmap can be used to probe a running container's network to enumerate open ports and perform other actions such as OS version detection and launching over-the-network scripted attacks",
			Remediation: "Consider removing package managers during the build process that could be used to download such software. Check that exposed ports don't allow for remote code execution",
			Categories: []string{
				"Network Tools",
			},
			LifecycleStages:    []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
			Severity:           storage.Severity_HIGH_SEVERITY,
			SORTName:           "nmap Execution",
			SORTLifecycleStage: "RUNTIME",
			PolicyVersion:      "1.1",
			PolicySections: []*storage.PolicySection{{
				PolicyGroups: []*storage.PolicyGroup{{
					FieldName: "Process Name",
					Values: []*storage.PolicyValue{{
						Value: "nmap",
					}},
				}},
			}},
			EventSource: storage.EventSource_DEPLOYMENT_EVENT,
		},
		LifecycleStage: storage.LifecycleStage_RUNTIME,
		Entity: &storage.Alert_Deployment_{
			Deployment: &storage.Alert_Deployment{
				Id:          "0f709d63-f2cc-4825-a984-b9cfd25b02cd",
				Name:        "debian-test",
				Type:        "Pod",
				Namespace:   "stackrox",
				NamespaceId: "dff6a17e-f246-4dc0-98b3-c70ee59c3cea",
				Labels: map[string]string{
					"run": "debian-test",
				},
				ClusterId:   "098e0e05-a96b-43ca-95af-3ef72cd32828",
				ClusterName: "remote",
				Containers: []*storage.Alert_Deployment_Container{{
					Image: &storage.ContainerImage{
						Id: "sha256:b16f66714660c4b3ea14d273ad8c35079b81b35d65d1e206072d226c7ff78299",
						Name: &storage.ImageName{
							Registry: "docker.io",
							Remote:   "library/debian",
							Tag:      "latest",
							FullName: "docker.io/library/debian:latest",
						},
					},
					Name: "debian-test",
				}},
				Annotations: map[string]string{
					"cni.projectcalico.org/podIP": "10.65.48.8/32",
				},
			},
		},
		ProcessViolation: &storage.Alert_ProcessViolation{
			Message: "Binary '/usr/bin/nmap' executed with arguments '-v -A localhost' under user ID 0",
			Processes: []*storage.ProcessIndicator{{
				Id:            "8472f6e2-53d2-4ddf-ad59-ecc43a8d98d2",
				DeploymentId:  "0f709d63-f2cc-4825-a984-b9cfd25b02cd",
				ContainerName: "debian-test",
				PodId:         "debian-test",
				PodUid:        "e20f2691-1371-588f-ae6d-bd0ef24af78b",
				Signal: &storage.ProcessSignal{
					Id:           "2569b112-64b1-11eb-9541-f65aedf20953",
					ContainerId:  "111bf6d5e461",
					Time:         makeTimestamp("2021-02-01T17:18:49.421852357Z"),
					Name:         "nmap",
					Args:         "-v -A localhost",
					ExecFilePath: "/usr/bin/nmap",
					Pid:          64307,
					LineageInfo: []*storage.ProcessSignal_LineageInfo{{
						ParentExecFilePath: "/bin/bash",
					}},
				},
				Namespace:          "stackrox",
				ContainerStartTime: makeTimestamp("2021-02-01T16:17:32Z"),
			}, {
				Id:            "cfc994d5-11bd-4471-a82e-b1735ad94e06",
				DeploymentId:  "0f709d63-f2cc-4825-a984-b9cfd25b02cd",
				ContainerName: "debian-test",
				PodId:         "debian-test",
				PodUid:        "e20f2691-1371-588f-ae6d-bd0ef24af78b",
				Signal: &storage.ProcessSignal{
					Id:           "8c84d098-64b1-11eb-9541-f65aedf20953",
					ContainerId:  "111bf6d5e461",
					Time:         makeTimestamp("2021-02-01T17:15:56.457252Z"),
					Name:         "nmap",
					Args:         "-v -A localhost",
					ExecFilePath: "/usr/bin/nmap",
					Pid:          65923,
					LineageInfo: []*storage.ProcessSignal_LineageInfo{{
						ParentExecFilePath: "/bin/bash",
					}},
				},
				Namespace:          "stackrox",
				ContainerStartTime: makeTimestamp("2021-02-01T16:17:32Z"),
			}},
		},
		Time:          makeTimestamp("2021-02-01T17:18:49.439085673Z"),
		FirstOccurred: makeTimestamp("2021-02-01T17:15:56.474524288Z"),
	}

	k8sAlert = storage.Alert{
		Id: "90e0feed-662c-4593-b414-e55d1eaff017",
		Policy: &storage.Policy{
			Id:          "8ab0f199-4904-4808-9461-3501da1d1b77",
			Name:        "Kubernetes Actions: Exec into Pod",
			Description: "Alerts when Kubernetes API receives request to execute command in container",
			Rationale:   "'pods/exec' is non-standard approach for interacting with containers. Attackers with permissions could execute malicious code and compromise resources within a cluster",
			Remediation: "Restrict RBAC access to the 'pods/exec' resource according to the Principle of Least Privilege. Limit such usage only to development, testing or debugging (non-production) activities",
			Categories: []string{
				"Kubernetes Events",
			},
			LifecycleStages:    []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
			Severity:           storage.Severity_HIGH_SEVERITY,
			SORTName:           "Kubernetes Actions: Exec into Pod",
			SORTLifecycleStage: "RUNTIME",
			PolicyVersion:      "1.1",
			PolicySections: []*storage.PolicySection{{
				PolicyGroups: []*storage.PolicyGroup{{
					FieldName: "Kubernetes Resource",
					Values: []*storage.PolicyValue{{
						Value: "PODS_EXEC",
					}},
				}},
			}},
			EventSource: storage.EventSource_DEPLOYMENT_EVENT,
		},
		LifecycleStage: storage.LifecycleStage_RUNTIME,
		Entity: &storage.Alert_Deployment_{
			Deployment: &storage.Alert_Deployment{
				Id:          "587556aa-5885-4a6e-8389-9d4e1c36e42a",
				Name:        "central",
				Type:        "Deployment",
				Namespace:   "stackrox",
				NamespaceId: "75868f7c-5949-4de3-bfc0-579f80148d45",
				Labels: map[string]string{
					"app":                          "central",
					"app.kubernetes.io/component":  "central",
					"app.kubernetes.io/instance":   "stackrox-central-services",
					"app.kubernetes.io/managed-by": "Helm",
					"app.kubernetes.io/name":       "stackrox",
					"app.kubernetes.io/part-of":    "stackrox-central-services",
					"app.kubernetes.io/version":    "3.0.55.x-118-gec7dc725f2-dirty",
					"helm.sh/chart":                "stackrox-central-services-55.0.118-gec7dc725f2-dirty",
				},
				ClusterId:   "943451bd-54c8-437d-98fa-820f5b9ad431",
				ClusterName: "remote",
				Containers: []*storage.Alert_Deployment_Container{{
					Image: &storage.ContainerImage{
						Id: "sha256:d4c1df40d209978307551e4b0a000067105e07578b66dfc8a4929f59dce86368",
						Name: &storage.ImageName{
							Registry: "docker.io",
							Remote:   "stackrox/main",
							Tag:      "3.0.55.x-118-gec7dc725f2-dirty",
							FullName: "docker.io/stackrox/main:3.0.55.x-118-gec7dc725f2-dirty",
						},
						NotPullable: true,
					},
					Name: "central",
				}},
				Annotations: map[string]string{
					"meta.helm.sh/release-name":      "stackrox-central-services",
					"meta.helm.sh/release-namespace": "stackrox",
					"owner":                          "stackrox",
					"email":                          "support@stackrox.com",
				},
			},
		},
		Violations: []*storage.Alert_Violation{{
			Message: "Kubernetes API received exec '/go/bin/dlv --headless --listen=:40000 --api-version=2 --accept-multiclient attach 1 --continue' request into pod 'central-6c8f4d4d8d-9hxpt' container 'central'",
			MessageAttributes: &storage.Alert_Violation_KeyValueAttrs_{
				KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
					Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{{
						Key:   "pod",
						Value: "central-6c8f4d4d8d-9hxpt",
					}, {
						Key:   "container",
						Value: "central",
					}, {
						Key:   "commands",
						Value: "/go/bin/dlv --headless --listen=:40000 --api-version=2 --accept-multiclient attach 1 --continue",
					}},
				},
			},
			Type: storage.Alert_Violation_K8S_EVENT,
			Time: makeTimestamp("2021-02-15T19:04:36.843302212Z"),
		}, {
			Message: "Kubernetes API received exec '/bin/sh -c [ -e /proc/sys/kernel/yama/ptrace_scope ] && cat /proc/sys/kernel/yama/ptrace_scope || echo 0' request into pod 'central-6c8f4d4d8d-9hxpt' container 'central'",
			MessageAttributes: &storage.Alert_Violation_KeyValueAttrs_{
				KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
					Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{{
						Key:   "pod",
						Value: "central-6c8f4d4d8d-9hxpt",
					}, {
						Key:   "container",
						Value: "central",
					}, {
						Key:   "commands",
						Value: "/bin/sh -c [ -e /proc/sys/kernel/yama/ptrace_scope ] && cat /proc/sys/kernel/yama/ptrace_scope || echo 0",
					}},
				},
			},
			Type: storage.Alert_Violation_K8S_EVENT,
			Time: makeTimestamp("2021-02-15T19:04:36.659410153Z"),
		}, {
			// Port forward violation is triggered by a different policy and will be in a separate Alert, but for the
			// sake of these tests it is sufficient to keep it also here.
			Message: "Kubernetes API received port forward request to pod 'central-84cbdb7869-4psdr'",
			MessageAttributes: &storage.Alert_Violation_KeyValueAttrs_{
				KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
					Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{{
						Key:   "pod",
						Value: "central-84cbdb7869-4psdr",
					}},
				},
			},
			Type: storage.Alert_Violation_K8S_EVENT,
			Time: makeTimestamp("2021-02-15T19:04:36.712345678Z"),
		}},
		Time:          makeTimestamp("2021-02-15T19:04:36.843516328Z"),
		FirstOccurred: makeTimestamp("2021-02-15T19:04:36.662294945Z"),
	}

	networkAlert = storage.Alert{
		Id: "86a55daa-de0d-4649-a7a9-ad71eeebfb6a",
		Policy: &storage.Policy{
			Id:          "1b74ffdd-8e67-444c-9814-1c23863c8ccb",
			Name:        "Unauthorized Network Flow",
			Description: "This policy generates a violation for the network flows that fall outside baselines for which 'alert on anomalous violations' is set.",
			Rationale:   "The network baseline is a list of flows that are allowed, and once it is frozen, any flow outside that is a concern.",
			Remediation: "Evaluate this network flow. If deemed to be okay, add it to the baseline. If not, investigate further as required.",
			Categories: []string{
				"Anomalous Activity",
			},
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
		},
		LifecycleStage: storage.LifecycleStage_RUNTIME,
		Entity: &storage.Alert_Deployment_{
			Deployment: &storage.Alert_Deployment{
				Id:          "b09dd238-9131-4e05-af89-727e37cd31f1",
				Name:        "central",
				Type:        "Deployment",
				Namespace:   "stackrox",
				NamespaceId: "e537bed5-1f30-4425-9757-0e0056fffedf",
				Labels: map[string]string{
					"app":                          "central",
					"app.kubernetes.io/component":  "central",
					"app.kubernetes.io/instance":   "stackrox-central-services",
					"app.kubernetes.io/managed-by": "Helm",
					"app.kubernetes.io/name":       "stackrox",
					"app.kubernetes.io/part-of":    "stackrox-central-services",
					"app.kubernetes.io/version":    "3.0.56.x-67-g847d2628a2",
					"helm.sh/chart":                "stackrox-central-services-56.0.67-g847d2628a2",
				},
				ClusterId:   "9e2755af-c2ba-4249-b4f2-f11a01694c71",
				ClusterName: "remote",
				Containers: []*storage.Alert_Deployment_Container{{
					Image: &storage.ContainerImage{
						Id: "sha256:09fcd52410a9b3ebb25fd932cc8269336ff2290cb8113e4513458020261267a0",
						Name: &storage.ImageName{
							Registry: "docker.io",
							Remote:   "stackrox/main",
							Tag:      "3.0.56.x-89-gc8e50289a2",
							FullName: "docker.io/stackrox/main:3.0.56.x-89-gc8e50289a2",
						},
					},
					Name: "central",
				}},
				Annotations: map[string]string{
					"email":                          "support@stackrox.com",
					"meta.helm.sh/release-name":      "stackrox-central-services",
					"meta.helm.sh/release-namespace": "stackrox",
					"owner":                          "stackrox",
				},
			},
		},
		Violations: []*storage.Alert_Violation{
			{
				Message: "Unexpected network flow found in deployment. Source name: 'central'. Destination name: 'External Entities'. Destination port: '9'. Protocol: 'L4_PROTOCOL_UDP'.",
				MessageAttributes: &storage.Alert_Violation_NetworkFlowInfo_{
					NetworkFlowInfo: &storage.Alert_Violation_NetworkFlowInfo{
						Protocol: storage.L4Protocol_L4_PROTOCOL_UDP,
						Source: &storage.Alert_Violation_NetworkFlowInfo_Entity{
							Name:                "central",
							EntityType:          storage.NetworkEntityInfo_DEPLOYMENT,
							DeploymentNamespace: "stackrox",
							DeploymentType:      "Deployment",
						},
						Destination: &storage.Alert_Violation_NetworkFlowInfo_Entity{
							Name:                "External Entities",
							EntityType:          storage.NetworkEntityInfo_INTERNET,
							DeploymentNamespace: "internet",
							Port:                9,
						},
					},
				},
				Type: storage.Alert_Violation_NETWORK_FLOW,
				Time: makeTimestamp("2021-03-21T21:50:46.600080752Z"),
			},
			{
				Message: "Unexpected network flow found in deployment. Source name: 'central'. Destination name: 'scanner'. Destination port: '8080'. Protocol: 'L4_PROTOCOL_TCP'.",
				MessageAttributes: &storage.Alert_Violation_NetworkFlowInfo_{
					NetworkFlowInfo: &storage.Alert_Violation_NetworkFlowInfo{
						Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
						Source: &storage.Alert_Violation_NetworkFlowInfo_Entity{
							Name:                "central",
							EntityType:          storage.NetworkEntityInfo_DEPLOYMENT,
							DeploymentNamespace: "stackrox",
							DeploymentType:      "Deployment",
						},
						Destination: &storage.Alert_Violation_NetworkFlowInfo_Entity{
							Name:                "scanner",
							EntityType:          storage.NetworkEntityInfo_DEPLOYMENT,
							DeploymentNamespace: "stackrox",
							DeploymentType:      "Deployment",
							Port:                8080,
						},
					},
				},
				Type: storage.Alert_Violation_NETWORK_FLOW,
				Time: makeTimestamp("2021-03-21T21:50:46.741573591Z"),
			},
		},
		Time:          makeTimestamp("2021-03-21T21:50:46.741586331Z"),
		FirstOccurred: makeTimestamp("2021-03-21T21:50:46.210811055Z"),
	}

	resourceAlert = storage.Alert{
		Id: "9f3cb534-5374-44f2-b661-83a8eec8dbdb",
		Policy: &storage.Policy{
			Id:          "18cbcb62-7d18-4a6c-b2ca-dd1242746943",
			Name:        "OpenShift: Kubeadmin Secret Accessed",
			Description: "Alert when the kubeadmin secret is accessed",
			Rationale:   "Kubeadmin is the default administrative user for OpenShift and can be used to obtain full administrative access to the cluster. Investigating if this was accessed for valid business purposes can help organizations to control the use of administrative privileges",
			Remediation: "Audit the access carefully to ensure that this secret is only accessed for valid business purposes.",
			Categories: []string{
				"Anomalous Activity",
				"Kubernetes Events",
			},
			LifecycleStages:    []storage.LifecycleStage{storage.LifecycleStage_RUNTIME},
			Severity:           storage.Severity_HIGH_SEVERITY,
			SORTName:           "OpenShift: Kubeadmin Secret Accessed",
			SORTLifecycleStage: "RUNTIME",
			PolicyVersion:      "1.1",
			PolicySections: []*storage.PolicySection{{
				PolicyGroups: []*storage.PolicyGroup{
					{
						FieldName: "Kubernetes Resource",
						Values:    []*storage.PolicyValue{{Value: "SECRETS"}},
					},
					{
						FieldName: "Kubernetes API Verb",
						Values:    []*storage.PolicyValue{{Value: "GET"}},
					},
					{
						FieldName: "Kubernetes Resource Name",
						Values:    []*storage.PolicyValue{{Value: "kubeadmin"}},
					},
					{
						FieldName: "Kubernetes User Name",
						Negate:    true,
						Values: []*storage.PolicyValue{
							{
								Value: "system:serviceaccount:openshift-authentication-operator:authentication-operator",
							},
							{
								Value: "system:apiserver",
							},
							{
								Value: "system:serviceaccount:openshift-authentication:oauth-openshift",
							},
						},
					},
				}}},
			EventSource: storage.EventSource_AUDIT_LOG_EVENT,
		},
		LifecycleStage: storage.LifecycleStage_RUNTIME,
		Entity: &storage.Alert_Resource_{
			Resource: &storage.Alert_Resource{
				ResourceType: storage.Alert_Resource_SECRETS,
				Name:         "kubeadmin",
				ClusterId:    "ea802db2-f0d0-4746-804e-77cdbbeebeba",
				ClusterName:  "remote",
				Namespace:    "kube-system",
			},
		},
		Violations: []*storage.Alert_Violation{{
			Message: "Access to secret \"kubeadmin\" in \"kube-system\"",
			MessageAttributes: &storage.Alert_Violation_KeyValueAttrs_{
				KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
					Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{{
						Key:   printer.APIVerbKey,
						Value: "GET",
					}, {
						Key:   printer.UsernameKey,
						Value: "system:admin",
					}, {
						Key:   printer.UserGroupsKey,
						Value: "system:masters, system:authenticated",
					}, {
						Key:   printer.UserAgentKey,
						Value: "oc/4.7.0 (darwin/amd64) kubernetes/c66c03f",
					}, {
						Key:   printer.IPAddressKey,
						Value: "67.160.238.22",
					}, {
						Key:   printer.ResourceURIKey,
						Value: "/api/v1/namespaces/kube-system/secrets/kubeadmin",
					}, {
						Key:   printer.ImpersonatedUsernameKey,
						Value: "system:serviceaccount:openshift-authentication-operator:authentication-operator",
					}, {
						Key:   printer.ImpersonatedUserGroupsKey,
						Value: "system:serviceaccounts, system:serviceaccounts:openshift-authentication-operator, system:authenticated",
					}},
				},
			},
			Type: storage.Alert_Violation_K8S_EVENT,
			Time: makeTimestamp("2021-07-15T17:26:35.115310605Z"),
		}, {
			Message: "Access to secret \"kubeadmin\" in \"kube-system\"",
			MessageAttributes: &storage.Alert_Violation_KeyValueAttrs_{
				KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
					Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{{
						Key:   printer.APIVerbKey,
						Value: "PATCH",
					}, {
						Key:   printer.UsernameKey,
						Value: "system:admin",
					}, {
						Key:   printer.UserGroupsKey,
						Value: "system:masters, system:authenticated",
					}, {
						Key:   printer.UserAgentKey,
						Value: "oc/4.7.0 (darwin/amd64) kubernetes/c66c03f",
					}, {
						Key:   printer.IPAddressKey,
						Value: "67.160.238.22",
					}, {
						Key:   printer.ResourceURIKey,
						Value: "/api/v1/namespaces/kube-system/secrets/kubeadmin",
					}},
				},
			},
			Type: storage.Alert_Violation_K8S_EVENT,
			Time: makeTimestamp("2021-07-15T17:36:35.115310605Z"),
		}},
		Time:          makeTimestamp("2021-07-15T17:36:35.115310605Z"),
		FirstOccurred: makeTimestamp("2021-07-15T17:26:35.115310605Z"),
	}
)

func TestViolations(t *testing.T) {
	suite.Run(t, &violationsTestSuite{})
}

func (s *violationsTestSuite) SetupTest() {
	s.deployAlert = deployAlert.Clone()
	s.processAlert = processAlert.Clone()
	s.k8sAlert = k8sAlert.Clone()
	s.networkAlert = networkAlert.Clone()
	s.resourceAlert = resourceAlert.Clone()
	s.allowCtx = sac.WithAllAccess(context.Background())
}

func (s *violationsTestSuite) TestNetworkAlert() {
	vs := s.getViolations(s.prepare().setAlerts(s.networkAlert).runRequestAndGetBody())
	s.Len(vs, 2)

	for _, v := range vs {
		s.Equal("NETWORK_FLOW", s.extr(v, ".violationInfo.violationType"))
		s.checkViolationInfo(v)
		s.checkAlertInfo(v, ".lifecycleStage")

		s.checkDeploymentInfo(v)
		s.Empty(s.extr(v, ".resourceInfo"))

		s.checkPolicy(v)
	}

	firstData := []struct {
		key   string
		value interface{}
	}{
		// source
		{key: ".violationInfo.violationTime", value: "2021-03-21T21:50:46.600080752Z"},
		{key: ".networkFlowInfo.source.name", value: "central"},
		{key: ".networkFlowInfo.source.entityType", value: "DEPLOYMENT"},
		{key: ".networkFlowInfo.source.deploymentNamespace", value: "stackrox"},
		{key: ".networkFlowInfo.source.deploymentType", value: "Deployment"},
		// destination
		{key: ".networkFlowInfo.destination.name", value: "External Entities"},
		{key: ".networkFlowInfo.destination.entityType", value: "INTERNET"},
		{key: ".networkFlowInfo.destination.port", value: 9.0},
		// protocol
		{key: ".networkFlowInfo.protocol", value: "L4_PROTOCOL_UDP"},
	}

	secondData := []struct {
		key   string
		value interface{}
	}{
		// source
		{key: ".violationInfo.violationTime", value: "2021-03-21T21:50:46.741573591Z"},
		{key: ".networkFlowInfo.source.name", value: "central"},
		{key: ".networkFlowInfo.source.entityType", value: "DEPLOYMENT"},
		{key: ".networkFlowInfo.source.deploymentNamespace", value: "stackrox"},
		{key: ".networkFlowInfo.source.deploymentType", value: "Deployment"},
		// destination
		{key: ".networkFlowInfo.destination.name", value: "scanner"},
		{key: ".networkFlowInfo.destination.entityType", value: "DEPLOYMENT"},
		{key: ".networkFlowInfo.destination.deploymentNamespace", value: "stackrox"},
		{key: ".networkFlowInfo.destination.deploymentType", value: "Deployment"},
		{key: ".networkFlowInfo.destination.port", value: 8080.0},
		// protocol
		{key: ".networkFlowInfo.protocol", value: "L4_PROTOCOL_TCP"},
	}

	for _, d := range firstData {
		s.Equal(d.value, s.extr(vs[0], d.key))
	}

	for _, d := range secondData {
		s.Equal(d.value, s.extr(vs[1], d.key))
	}
}

func (s *violationsTestSuite) TestProcessAlert() {
	vs := s.getViolations(s.prepare().setAlerts(s.processAlert).runRequestAndGetBody())
	s.Len(vs, 2)

	for _, v := range vs {
		s.Equal("PROCESS_EVENT", s.extr(v, ".violationInfo.violationType"))
		s.Equal(float64(0), s.extr(v, ".processInfo.processUid"))
		s.Equal(float64(0), s.extr(v, ".processInfo.processGid"))
		s.Equal(s.extr(v, ".processInfo.processCreationTime"), s.extr(v, ".violationInfo.violationTime"))

		s.checkViolationInfo(v, ".podId", ".podUid", ".containerName", ".containerStartTime", ".containerId")
		s.checkProcessInfo(v)
		s.checkAlertInfo(v, ".lifecycleStage")

		s.checkDeploymentInfo(v)
		s.Empty(s.extr(v, ".resourceInfo"))

		s.checkPolicy(v)
	}
}

func (s *violationsTestSuite) TestK8sAlert() {
	vs := s.getViolations(s.prepare().setAlerts(s.k8sAlert).runRequestAndGetBody())
	s.Len(vs, 3)

	for _, v := range vs {
		s.Equal("K8S_EVENT", s.extr(v, ".violationInfo.violationType"))

		s.checkViolationInfo(v, ".violationMessageAttributes")
		s.checkAlertInfo(v, ".lifecycleStage")

		s.checkDeploymentInfo(v)
		s.Empty(s.extr(v, ".resourceInfo"))

		s.checkPolicy(v)
	}

	s.sortViolationsByID(vs)

	s.Equal("2021-02-15T19:04:36.712345678Z", s.extr(vs[0], ".violationInfo.violationTime"))
	s.assertPresent(vs[0], ".violationInfo", ".podId") // port-forward has only pod

	s.Equal("2021-02-15T19:04:36.843302212Z", s.extr(vs[1], ".violationInfo.violationTime"))
	s.assertPresent(vs[1], ".violationInfo", ".containerName", ".podId") // exec has both pod and container

	s.Equal("2021-02-15T19:04:36.659410153Z", s.extr(vs[2], ".violationInfo.violationTime"))
	s.assertPresent(vs[2], ".violationInfo", ".containerName", ".podId")
}

func (s *violationsTestSuite) TestDeployAlert() {
	vs := s.getViolations(s.prepare().setAlerts(s.deployAlert).runRequestAndGetBody())
	s.Len(vs, 1)

	s.Equal("GENERIC", s.extr(vs[0], ".violationInfo.violationType"))
	s.Equal("2021-02-01T16:09:02.128791072Z", s.extr(vs[0], ".alertInfo.alertFirstOccurred"))
	s.Equal("2021-02-01T16:09:02.193352817Z", s.extr(vs[0], ".violationInfo.violationTime"))

	// Splunk Violation message must contain three lines from three generic Violations of the same Alert.
	s.Len(strings.Split(s.extr(vs[0], ".violationInfo.violationMessage").(string), "\n"), 3)

	s.checkViolationInfo(vs[0])
	s.checkAlertInfo(vs[0])
	s.checkDeploymentInfo(vs[0])
	s.checkPolicy(vs[0])
}

func (s *violationsTestSuite) TestResourceAlert() {
	vs := s.getViolations(s.prepare().setAlerts(s.resourceAlert).runRequestAndGetBody())
	s.Len(vs, 2)

	for _, v := range vs {
		s.Equal("K8S_EVENT", s.extr(v, ".violationInfo.violationType"))

		s.checkViolationInfo(v, ".violationMessageAttributes")
		s.checkAlertInfo(v, ".lifecycleStage")

		s.checkResourceInfo(v)
		s.Empty(s.extr(v, ".deploymentInfo"))

		s.checkPolicy(v)
	}

	s.Equal("2021-07-15T17:26:35.115310605Z", s.extr(vs[0], ".violationInfo.violationTime"))
	s.Equal("2021-07-15T17:36:35.115310605Z", s.extr(vs[1], ".violationInfo.violationTime"))

}

func (s *violationsTestSuite) TestViolationIdsAreDistinct() {
	vs := s.getViolations(s.prepare().setAlerts(s.processAlert, s.k8sAlert, s.deployAlert, s.networkAlert).runRequestAndGetBody())

	ids := set.StringSet{}
	for _, v := range vs {
		id, ok := s.extr(v, ".violationInfo.violationId").(string)
		s.Truef(ok, "Detected violationId that is not a string: %v", id)
		s.Truef(ids.Add(id), "violationId=%q is not unique. Already seen ids: %v", id, ids)
	}
}

func (s *violationsTestSuite) TestWithDeploymentImage() {
	alert := s.processAlert.Clone()
	// Change alert's Entity from Alert_Deployment to Alert_Image. Conveniently the former Alert_Deployment has a ContainerImage we can use for testing.
	alert.Entity = &storage.Alert_Image{
		Image: alert.GetDeployment().Containers[0].GetImage(),
	}

	vs := s.getViolations(s.prepare().setAlerts(alert).runRequestAndGetBody())

	s.assertPresent(vs[0], ".deploymentInfo",
		// deploymentImage must obviously be present coming from above
		".deploymentImage",
		// other deployment details are obtained from ProcessViolation
		".deploymentId", ".deploymentNamespace")
}

func (s *violationsTestSuite) TestAlertWithoutPolicy() {
	alert := s.processAlert.Clone()
	alert.Policy = nil
	alert.ProcessViolation.Processes = alert.ProcessViolation.Processes[:1]
	vs := s.getViolations(s.prepare().setAlerts(alert).runRequestAndGetBody())
	s.Nil(s.extr(vs[0], ".policyInfo"))
}

func (s *violationsTestSuite) TestProcessAlertWithoutProcessIndicators() {
	alert := s.processAlert.Clone()
	alert.ProcessViolation.Processes = []*storage.ProcessIndicator{}
	s.Empty(s.getViolations(s.prepare().setAlerts(alert).runRequestAndGetBody()))
}

func (s *violationsTestSuite) TestProcessAlertWithoutProcessSignal() {
	alert := s.processAlert.Clone()
	alert.ProcessViolation.Processes = alert.ProcessViolation.Processes[:1]
	alert.ProcessViolation.Processes[0].Signal = nil
	vs := s.getViolations(s.prepare().setAlerts(alert).runRequestAndGetBody())
	s.checkViolationInfo(vs[0], ".podId", ".podUid", ".containerName", ".containerStartTime") // .containerId isn't available
	s.checkAlertInfo(vs[0])
	// That's all it can gather from ProcessIndicator without ProcessSignal
	s.assertPresent(vs[0], ".processInfo", ".processViolationId")
	s.checkDeploymentInfo(vs[0])
	s.checkPolicy(vs[0])
}

func (s *violationsTestSuite) TestAlertWithoutViolations() {
	alert := s.deployAlert.Clone()
	alert.Violations = []*storage.Alert_Violation{}
	s.Empty(s.getViolations(s.prepare().setAlerts(alert).runRequestAndGetBody()))
}

func (s *violationsTestSuite) TestK8sAlertWithoutDeploymentOrResource() {
	alert := s.k8sAlert.Clone()
	alert.Entity = nil
	alert.Violations = alert.Violations[:1]
	vs := s.getViolations(s.prepare().setAlerts(alert).runRequestAndGetBody())
	s.Empty(s.extr(vs[0], ".deploymentInfo"))
	s.Empty(s.extr(vs[0], ".resourceInfo"))
}

func (s *violationsTestSuite) TestProcessAlertWithoutDeployment() {
	alert := s.processAlert.Clone()
	alert.Entity = nil
	alert.ProcessViolation.Processes = alert.ProcessViolation.Processes[:1]
	vs := s.getViolations(s.prepare().setAlerts(alert).runRequestAndGetBody())
	// deploymentInfo still has some attributes because they came from ProcessIndicator-s
	s.assertPresent(vs[0], ".deploymentInfo", ".deploymentId", ".deploymentNamespace")
}

func (s *violationsTestSuite) TestProcessAlertNotMatchingDeploymentId() {
	alert := s.processAlert.Clone()
	alert.ProcessViolation.Processes = alert.ProcessViolation.Processes[:1]
	alert.ProcessViolation.Processes[0].DeploymentId = "blah"
	vs := s.getViolations(s.prepare().setAlerts(alert).runRequestAndGetBody())
	// DeploymentId value from ProcessIndicator should take priority
	s.Equal("blah", s.extr(vs[0], ".deploymentInfo.deploymentId"))
	s.NotEmpty(s.extr(vs[0], ".deploymentInfo.deploymentNamespace"))
}

func (s *violationsTestSuite) TestProcessAlertNotMatchingDeploymentInfo() {
	alert := s.processAlert.Clone()
	alert.ProcessViolation.Processes = alert.ProcessViolation.Processes[:1]
	alert.ProcessViolation.Processes[0].ClusterId = "blah-cluster"
	alert.ProcessViolation.Processes[0].Namespace = "blah-namespace"
	vs := s.getViolations(s.prepare().setAlerts(alert).runRequestAndGetBody())
	s.Equal("blah-cluster", s.extr(vs[0], ".deploymentInfo.clusterId"))
	s.Equal("blah-namespace", s.extr(vs[0], ".deploymentInfo.deploymentNamespace"))
}

// getViolations extracts "violations" attribute as a slice for later querying them with JSONPath.
func (s *violationsTestSuite) getViolations(body map[string]interface{}) []interface{} {
	violations := s.extr(body, ".violations")
	if violations == nil {
		return nil
	}
	return violations.([]interface{})
}

// extr extracts value from input according to provided jsonPath. Returns nil if given attribute does not exist.
func (s *violationsTestSuite) extr(input interface{}, jsonPath string) interface{} {
	jp := jsonpath.New("")

	err := jp.Parse("{" + jsonPath + "}")
	s.NoError(err)

	val, err := jp.FindResults(input)
	if err != nil && strings.HasSuffix(err.Error(), " is not found") {
		return nil
	}
	s.NoError(err)

	return val[0][0].Interface()
}

func (s *violationsTestSuite) assertPresent(violation interface{}, prefix string, attributes ...string) {
	for _, attr := range attributes {
		s.NotEmpty(s.extr(violation, prefix+attr))
	}
}

func (s *violationsTestSuite) checkViolationInfo(violation interface{}, extraAttrs ...string) {
	s.assertPresent(violation, ".violationInfo",
		".violationId",
		".violationMessage",
		".violationType",
		".violationTime")
	s.assertPresent(violation, ".violationInfo", extraAttrs...)
}

func (s *violationsTestSuite) checkAlertInfo(violation interface{}, extraAttrs ...string) {
	s.assertPresent(violation, ".alertInfo", ".alertId")
	s.assertPresent(violation, ".alertInfo", extraAttrs...)
}

func (s *violationsTestSuite) checkProcessInfo(violation interface{}) {
	s.assertPresent(violation, ".processInfo",
		".processViolationId",
		".processSignalId",
		".processCreationTime",
		".processName",
		".processArgs",
		".execFilePath",
		".pid",
		".processLineageInfo")
}

func (s *violationsTestSuite) checkDeploymentInfo(violation interface{}) {
	s.assertPresent(violation, ".deploymentInfo",
		".deploymentId",
		".deploymentName",
		".deploymentType",
		".deploymentNamespace",
		".deploymentNamespaceId",
		".deploymentLabels",
		".clusterId",
		".clusterName",
		".deploymentContainers",
		".deploymentAnnotations")
}

func (s *violationsTestSuite) checkResourceInfo(violation interface{}) {
	s.assertPresent(violation, ".resourceInfo",
		".resourceType",
		".name",
		".clusterName",
		".namespace")
}

func (s *violationsTestSuite) checkPolicy(violation interface{}) {
	s.assertPresent(violation, ".policyInfo",
		".policyId",
		".policyName",
		".policyDescription",
		".policyRationale",
		".policyCategories",
		".policyLifecycleStages",
		".policySeverity",
		".policyVersion")
}

func (s *violationsTestSuite) TestResponsePagination() {
	s.withAlerts(these(s.processAlert, s.k8sAlert, s.deployAlert, s.networkAlert), func(alertsDS datastore.DataStore) {
		violationsNoPagination := s.getViolations(s.prepare().setAlertsDS(alertsDS).runRequestAndGetBody())
		s.NotEmpty(violationsNoPagination)
		s.sortViolationsByID(violationsNoPagination)

		cases := []paginationSettings{
			{maxAlertsFromQuery: 1, violationsPerResponse: 1},
			{maxAlertsFromQuery: 1, violationsPerResponse: 5},
			{maxAlertsFromQuery: 1, violationsPerResponse: 7},
			{maxAlertsFromQuery: 1, violationsPerResponse: 10},
			{maxAlertsFromQuery: 2, violationsPerResponse: 5},
			{maxAlertsFromQuery: 10, violationsPerResponse: 1},
		}
		for _, c := range cases {
			s.Run(fmt.Sprintf("%v", c), func() {
				violationsPaginated := make([]interface{}, 0, len(violationsNoPagination))
				checkpoint := "2020-01-01T00:00:00Z"
				iterations := 0
				for ; ; iterations++ {
					body := s.prepare().setCheckpoint(checkpoint).setPagination(c).setAlertsDS(alertsDS).runRequestAndGetBody()
					checkpoint = s.extr(body, ".newCheckpoint").(string)
					vs := s.getViolations(body)
					if len(vs) == 0 {
						break
					}
					violationsPaginated = append(violationsPaginated, vs...)
				}
				s.Empty(mustParseCheckpoint(s.T(), checkpoint).fromAlertID) // No more alerts available.

				if c.violationsPerResponse < len(violationsNoPagination) {
					s.Greater(iterations, 1)
				} else {
					s.Equal(1, iterations)
				}
				s.sortViolationsByID(violationsPaginated)
				s.Equal(violationsNoPagination, violationsPaginated)
			})
		}
	})
}

func (s *violationsTestSuite) TestCheckpointIteration() {
	violationsNoPagination := s.getViolations(s.prepare().setAlerts(s.processAlert, s.k8sAlert, s.deployAlert, s.networkAlert).runRequestAndGetBody())
	s.NotEmpty(violationsNoPagination)
	s.sortViolationsByID(violationsNoPagination)

	violationsPaginated := make([]interface{}, 0, len(violationsNoPagination))

	// Get a part of violations from the first three alerts initially.
	checkpoint := "2021-01-01T00:00:00Z__2021-02-15T19:04:36.712345678Z" // ToTimestamp in the middle of k8sAlert violations
	body := s.prepare().setCheckpoint(checkpoint).setAlerts(s.deployAlert, s.processAlert, s.k8sAlert).runRequestAndGetBody()
	checkpoint = s.extr(body, ".newCheckpoint").(string)
	violationsPaginated = append(violationsPaginated, s.getViolations(body)...)
	s.Equal("2021-02-15T19:04:36.712345678Z", checkpoint)
	s.Less(len(violationsPaginated), len(violationsNoPagination))

	// Next, get remaining violations from k8sAlert and a new networkAlert.
	body = s.prepare().setCheckpoint(checkpoint).setAlerts(s.k8sAlert, s.networkAlert).runRequestAndGetBody()
	checkpoint = s.extr(body, ".newCheckpoint").(string)
	violationsPaginated = append(violationsPaginated, s.getViolations(body)...)
	assertCheckpointIsNow(s.T(), checkpoint)

	// We should get the same in the end.
	s.sortViolationsByID(violationsPaginated)
	s.Equal(violationsNoPagination, violationsPaginated)
}

func (s *violationsTestSuite) sortViolationsByID(violations []interface{}) {
	sort.Slice(violations, func(i, j int) bool {
		return s.extr(violations[i], ".violationInfo.violationId").(string) < s.extr(violations[j], ".violationInfo.violationId").(string)
	})
}

func (s *violationsTestSuite) TestCheckpointTimestampFiltering() {
	const now = "now" // just a bit fancier way to denote a special value

	samples := []struct {
		name                                    string
		alerts                                  []*storage.Alert
		fromCheckpoint                          string
		violationsNotBefore, violationsNotAfter string // Time range that should contain all returned violations.
		expectedCount                           int
		expectedNewCheckpoint                   string
	}{
		{
			name:                  "Checkpoint before violations",
			alerts:                []*storage.Alert{s.processAlert, s.k8sAlert, s.deployAlert, s.networkAlert},
			fromCheckpoint:        "2021-01-01T00:00:00Z",
			violationsNotBefore:   "2021-02-01T16:09:02.193352817Z", // The first violation timestamp in the data.
			violationsNotAfter:    "2021-03-21T21:50:46.741573591Z", // The biggest violation timestamp in the data.
			expectedCount:         8,                                // Count of all violations in the data.
			expectedNewCheckpoint: now,
		}, {
			name:                  "Checkpoint without ToTimestamp over all violations",
			alerts:                []*storage.Alert{s.processAlert, s.k8sAlert, s.deployAlert, s.networkAlert},
			fromCheckpoint:        "2021-02-01T17:18:48Z",
			violationsNotBefore:   "2021-02-01T17:18:49.421852357Z", // The first violation timestamp after the checkpoint.
			violationsNotAfter:    "2021-03-21T21:50:46.741573591Z", // The biggest violation timestamp in the data.
			expectedCount:         6,
			expectedNewCheckpoint: now,
		}, {
			name:                  "Checkpoint with ToTimestamp over all violations",
			alerts:                []*storage.Alert{s.processAlert, s.k8sAlert, s.deployAlert, s.networkAlert},
			fromCheckpoint:        "2021-02-01T16:09:02.193352817Z__2021-02-15T19:04:36.712345678Z",
			violationsNotBefore:   "2021-02-01T17:15:56.457252Z",    // The smallest violation timestamp in the data.
			violationsNotAfter:    "2021-02-15T19:04:36.712345678Z", // The last violation is exactly at ToTimestamp.
			expectedCount:         4,
			expectedNewCheckpoint: "2021-02-15T19:04:36.712345678Z", // ToTimestamp value.
		}, {
			name:           "No checkpoint and no violations",
			alerts:         nil,
			fromCheckpoint: "",
			expectedCount:  0,
			// The checkpoint for subsequent querying should start from FromTimestamp=Now() because no Alerts were
			// present before that moment.
			expectedNewCheckpoint: now,
		}, {
			name:                  "No checkpoint and all violations",
			alerts:                []*storage.Alert{s.processAlert, s.k8sAlert, s.deployAlert, s.networkAlert},
			fromCheckpoint:        "",
			violationsNotBefore:   "2021-02-01T16:09:02.193352817Z",
			violationsNotAfter:    "2021-03-21T21:50:46.741573591Z",
			expectedCount:         8,
			expectedNewCheckpoint: now,
		}, {
			// While FromTimestamp==ToTimestamp is possible and won't be an error, it will select no data.
			// Returned newCheckpoint will allow to query data further.
			name:                  "FromTimestamp is equal to ToTimestamp",
			alerts:                []*storage.Alert{s.processAlert, s.k8sAlert, s.deployAlert, s.networkAlert},
			fromCheckpoint:        "2021-02-01T18:00:00Z__2021-02-01T18:00:00Z", // Between violations.
			expectedCount:         0,
			expectedNewCheckpoint: "2021-02-01T18:00:00Z",
		}, {
			// Process violations filtering needs to be checked independently because it is done separately.
			name:                  "ToTimestamp in the middle of Process Indicators",
			alerts:                []*storage.Alert{s.processAlert},
			fromCheckpoint:        "2021-02-01T00:00:00Z__2021-02-01T17:15:56.457252Z",
			violationsNotBefore:   "2021-02-01T17:15:56.457252Z",
			violationsNotAfter:    "2021-02-01T17:15:56.457252Z",
			expectedCount:         1,
			expectedNewCheckpoint: "2021-02-01T17:15:56.457252Z",
		}, {
			name:                  "FromTimestamp in the middle of Process Indicators",
			alerts:                []*storage.Alert{s.processAlert},
			fromCheckpoint:        "2021-02-01T17:15:56.457252Z__2021-02-01T17:18:49.421852357Z",
			violationsNotBefore:   "2021-02-01T17:18:49.421852357Z",
			violationsNotAfter:    "2021-02-01T17:18:49.421852357Z",
			expectedCount:         1,
			expectedNewCheckpoint: "2021-02-01T17:18:49.421852357Z",
		}, {
			// Non-process violations filtering on the example of K8S events.
			name:                  "ToTimestamp in the middle of k8s events",
			alerts:                []*storage.Alert{s.k8sAlert},
			fromCheckpoint:        "2021-02-15T19:04:36Z__2021-02-15T19:04:36.712345678Z",
			violationsNotBefore:   "2021-02-15T19:04:36.659410153Z",
			violationsNotAfter:    "2021-02-15T19:04:36.712345678Z",
			expectedCount:         2,
			expectedNewCheckpoint: "2021-02-15T19:04:36.712345678Z",
		}, {
			name:                  "FromTimestamp in the middle of k8s events",
			alerts:                []*storage.Alert{s.k8sAlert},
			fromCheckpoint:        "2021-02-15T19:04:36.659410153Z__2021-02-15T19:04:37Z",
			violationsNotBefore:   "2021-02-15T19:04:36.712345678Z",
			violationsNotAfter:    "2021-02-15T19:04:36.843302212Z",
			expectedCount:         2,
			expectedNewCheckpoint: "2021-02-15T19:04:37Z",
		}, {
			// Non-runtime violations filtering is again a bit special and so we check it separately.
			name:                  "Deploy violations in the range",
			alerts:                []*storage.Alert{s.deployAlert},
			fromCheckpoint:        "2021-02-01T16:09:02.193352816Z__2021-02-01T16:09:02.193352817Z",
			violationsNotBefore:   "2021-02-01T16:09:02.193352817Z",
			violationsNotAfter:    "2021-02-01T16:09:02.193352817Z",
			expectedCount:         1,
			expectedNewCheckpoint: "2021-02-01T16:09:02.193352817Z",
		}, {
			name:                  "Deploy violations after the range",
			alerts:                []*storage.Alert{s.deployAlert},
			fromCheckpoint:        "2021-01-01T00:00:00Z__2021-02-01T16:09:02.193352816Z",
			expectedCount:         0,
			expectedNewCheckpoint: "2021-02-01T16:09:02.193352816Z",
		}, {
			name:                  "Deploy violations before the range",
			alerts:                []*storage.Alert{s.deployAlert},
			fromCheckpoint:        "2021-02-01T16:09:02.193352817Z__2021-04-01T00:00:00Z",
			expectedCount:         0,
			expectedNewCheckpoint: "2021-04-01T00:00:00Z",
		},
	}

	for _, sample := range samples {
		s.Run(sample.name, func() {
			checkpointParam := []string{sample.fromCheckpoint}
			if sample.fromCheckpoint == "" {
				checkpointParam = nil
			}

			body := s.prepare().setCheckpoint(checkpointParam...).setAlerts(sample.alerts...).runRequestAndGetBody()
			vs := s.getViolations(body)

			s.Len(vs, sample.expectedCount)

			if sample.expectedCount > 0 {
				// Check that all violations fall within the expected range.
				fromTs := makeTimestamp(sample.violationsNotBefore)
				toTs := makeTimestamp(sample.violationsNotAfter)
				for _, v := range vs {
					ts := makeTimestamp(s.extr(v, ".violationInfo.violationTime").(string))
					s.True(ts.Compare(fromTs) >= 0, "Violation timestamp is earlier than expected", v)
					s.True(ts.Compare(toTs) <= 0, "Violation timestamp is later than expected", v)
				}
			}

			newCheckpoint := s.extr(body, ".newCheckpoint").(string)
			if sample.expectedNewCheckpoint == now {
				assertCheckpointIsNow(s.T(), newCheckpoint)
			} else {
				s.Equal(sample.expectedNewCheckpoint, newCheckpoint)
			}
		})
	}
}

func (s *violationsTestSuite) TestCheckpointFromAlertIDFiltering() {
	smallerID, biggerID := s.processAlert.GetId(), s.k8sAlert.GetId()
	if smallerID > biggerID {
		smallerID, biggerID = biggerID, smallerID
	}

	body := s.prepare().setCheckpoint("2000-01-01T00:00:00Z__2021-03-29T14:37:00Z__"+smallerID).
		setAlerts(s.k8sAlert, s.processAlert).runRequestAndGetBody()
	vs := s.getViolations(body)

	s.NotEmpty(vs)

	for _, v := range vs {
		s.Equal(biggerID, s.extr(v, ".alertInfo.alertId"))
	}
}

func (s *violationsTestSuite) TestCheckpointWithFromAlertID() {
	pagination := paginationSettings{violationsPerResponse: 4}
	body := s.prepare().setPagination(pagination).setAlerts(s.deployAlert, s.processAlert, s.k8sAlert, s.networkAlert).runRequestAndGetBody()

	vs := s.getViolations(body)
	s.GreaterOrEqual(len(vs), 4)

	newCheckpoint := mustParseCheckpoint(s.T(), s.extr(body, ".newCheckpoint").(string))
	assertFromTimestampIsLongTimeAgo(s.T(), newCheckpoint)
	assertTimestampIsNow(s.T(), newCheckpoint.toTimestamp)
	// FromAlertID must have a value because there were more alerts available for the time range after the response page
	// was finalized.
	s.NotEmpty(newCheckpoint.fromAlertID)
}

func (s *violationsTestSuite) TestNonParsableCheckpoint() {
	w := httptest.NewRecorder()

	s.prepare().setCheckpoint("This isn't any good timestamp").runRequest(w)

	s.Equal(http.StatusBadRequest, w.Code)
	body := w.Body.String()
	s.Regexp("error parsing.*from_checkpoint.*This isn't any good timestamp", body)
}

func (s *violationsTestSuite) TestCheckpointInTheFuture() {
	w := httptest.NewRecorder()

	s.prepare().setCheckpoint("2130-12-31T23:59:59Z").setAlerts(s.processAlert, s.k8sAlert, s.deployAlert).runRequest(w)

	s.Equal(http.StatusBadRequest, w.Code)
	s.Regexp("error.*validating checkpoint.*FromTimestamp.*in the future", w.Body.String())
}

func (s *violationsTestSuite) TestFirstCheckpointParamWins() {
	checkpointParams := []string{"2021-03-26T09:28:59Z", "2005-01-01T00:00:00Z"}
	vs := s.getViolations(s.prepare().setCheckpoint(checkpointParams...).setAlerts(s.k8sAlert).runRequestAndGetBody())
	s.Empty(vs)
}

// requestBuilder allows to configure parameters for API request in test and to trigger the request itself.
type requestBuilder struct {
	t                *testing.T
	ctx              context.Context
	checkpointParams []string
	pagination       paginationSettings
	alerts           []*storage.Alert
	alertsDS         datastore.DataStore
	// useAlertsDS is true if alertsDS should be used. When false, a new datastore with alerts will be created.
	// In other words, either alerts or alertsDS is used but not both, and useAlertsDS controls which one will be.
	useAlertsDS bool
}

func (s *violationsTestSuite) prepare() *requestBuilder {
	return &requestBuilder{
		t:          s.T(),
		ctx:        s.allowCtx,
		pagination: defaultPaginationSettings,
	}
}
func (rb *requestBuilder) setCheckpoint(checkpointParams ...string) *requestBuilder {
	rb.checkpointParams = checkpointParams
	return rb
}
func (rb *requestBuilder) setPagination(pagination paginationSettings) *requestBuilder {
	rb.pagination = pagination
	return rb
}
func (rb *requestBuilder) setAlerts(alerts ...*storage.Alert) *requestBuilder {
	rb.alerts = alerts
	rb.useAlertsDS = false
	return rb
}
func (rb *requestBuilder) setAlertsDS(alertsDS datastore.DataStore) *requestBuilder {
	rb.alertsDS = alertsDS
	rb.useAlertsDS = true
	return rb
}
func (rb *requestBuilder) runRequest(responseWriter http.ResponseWriter) {
	alertsDS := rb.alertsDS
	if !rb.useAlertsDS {
		ds := makeDS(rb.t, rb.alerts)
		defer ds.teardown(rb.t)
		alertsDS = ds.alertsDS
	}

	handler := newViolationsHandler(alertsDS, rb.pagination)

	u, err := url.Parse("/ignored")
	require.NoError(rb.t, err)
	if len(rb.checkpointParams) > 0 {
		q := u.Query()
		q["from_checkpoint"] = rb.checkpointParams
		u.RawQuery = q.Encode()
	}
	r := httptest.NewRequest("GET", u.String(), nil)
	r = r.WithContext(rb.ctx)

	handler.ServeHTTP(responseWriter, r)
}
func (rb *requestBuilder) runRequestAndGetBody() map[string]interface{} {
	w := httptest.NewRecorder()
	rb.runRequest(w)
	assert.Equal(rb.t, http.StatusOK, w.Code)

	var parsed map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &parsed)
	assert.NoError(rb.t, err)

	return parsed
}

func (s *violationsTestSuite) TestResponseContentType() {
	w := httptest.NewRecorder()

	s.prepare().setAlerts(s.deployAlert).runRequest(w)

	s.Equal(http.StatusOK, w.Code)
	s.Equal("application/json", w.Header().Get("Content-Type"))
}

func (s *violationsTestSuite) TestViolationsHandlerWriteError() {
	w := mock.NewFailingResponseWriter(errors.New("mock http write error"))
	s.PanicsWithError("net/http: abort Handler", func() {
		s.prepare().setAlerts(s.processAlert).runRequest(w)
	})
}

func (s *violationsTestSuite) TestGenerateViolationId() {
	v1Empty := storage.Alert_Violation{}
	v2Empty := storage.Alert_Violation{}

	id1, err := generateViolationID("alert1", &v1Empty)
	s.Require().NoError(err)
	id2, err := generateViolationID("alert1", &v2Empty)
	s.Require().NoError(err)
	s.Equal(id1, id2)

	id2Other, err := generateViolationID("other-alert", &v2Empty)
	s.Require().NoError(err)
	s.NotEqual(id2, id2Other)

	v3 := storage.Alert_Violation{
		Message: "mock message",
		Type:    storage.Alert_Violation_K8S_EVENT,
		Time: &types.Timestamp{
			Seconds: 123,
			Nanos:   456,
		},
	}
	id3, err := generateViolationID("alert1", &v3)
	s.Require().NoError(err)

	v4 := v3.Clone()
	v4.Message = "mock message4"
	id4, err := generateViolationID("alert1", v4)
	s.Require().NoError(err)

	s.NotEqual(id3, id4)
}

func BenchmarkGenerateViolationId(b *testing.B) {
	violations := deployAlert.Clone().Violations
	violations = append(violations, k8sAlert.Clone().Violations...)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := violations[i%len(violations)]
		id, err := generateViolationID(deployAlert.GetId(), v)
		if err != nil || len(id) != 36 {
			b.FailNow()
		}
	}
}
