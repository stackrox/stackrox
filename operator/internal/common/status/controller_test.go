package status

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestDetermineAvailableState(t *testing.T) {
	tests := map[string]struct {
		statuses        []workloadStatus
		expectedStatus  corev1.ConditionStatus
		expectedReason  string
		messageContains string
	}{
		"no workloads returns not available": {
			statuses:       nil,
			expectedStatus: corev1.ConditionFalse,
			expectedReason: "NoWorkloads",
		},
		"single ready deployment": {
			statuses: []workloadStatus{
				deploymentStatus{
					namedWorkload: namedWorkload{name: "central"},
					status: appsv1.DeploymentStatus{
						Conditions: []appsv1.DeploymentCondition{
							{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
						},
					},
				},
			},
			expectedStatus: corev1.ConditionTrue,
			expectedReason: "WorkloadsReady",
		},
		"single not-ready deployment": {
			statuses: []workloadStatus{
				deploymentStatus{
					namedWorkload: namedWorkload{name: "central"},
					status: appsv1.DeploymentStatus{
						Conditions: []appsv1.DeploymentCondition{
							{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionFalse},
						},
					},
				},
			},
			expectedStatus:  corev1.ConditionFalse,
			expectedReason:  "WorkloadsNotReady",
			messageContains: "central",
		},
		"single ready daemonset": {
			statuses: []workloadStatus{
				daemonSetStatus{
					namedWorkload: namedWorkload{name: "collector"},
					status: appsv1.DaemonSetStatus{
						DesiredNumberScheduled: 3,
						NumberAvailable:        3,
					},
				},
			},
			expectedStatus: corev1.ConditionTrue,
			expectedReason: "WorkloadsReady",
		},
		"single not-ready daemonset": {
			statuses: []workloadStatus{
				daemonSetStatus{
					namedWorkload: namedWorkload{name: "collector"},
					status: appsv1.DaemonSetStatus{
						DesiredNumberScheduled: 3,
						NumberAvailable:        1,
					},
				},
			},
			expectedStatus:  corev1.ConditionFalse,
			expectedReason:  "WorkloadsNotReady",
			messageContains: "collector",
		},
		"mixed workloads all ready": {
			statuses: []workloadStatus{
				deploymentStatus{
					namedWorkload: namedWorkload{name: "central"},
					status: appsv1.DeploymentStatus{
						Conditions: []appsv1.DeploymentCondition{
							{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
						},
					},
				},
				deploymentStatus{
					namedWorkload: namedWorkload{name: "scanner"},
					status: appsv1.DeploymentStatus{
						Conditions: []appsv1.DeploymentCondition{
							{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
						},
					},
				},
				daemonSetStatus{
					namedWorkload: namedWorkload{name: "collector"},
					status: appsv1.DaemonSetStatus{
						DesiredNumberScheduled: 5,
						NumberAvailable:        5,
					},
				},
			},
			expectedStatus: corev1.ConditionTrue,
			expectedReason: "WorkloadsReady",
		},
		"mixed workloads with not-ready deployment": {
			statuses: []workloadStatus{
				deploymentStatus{
					namedWorkload: namedWorkload{name: "central"},
					status: appsv1.DeploymentStatus{
						Conditions: []appsv1.DeploymentCondition{
							{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionFalse},
						},
					},
				},
				daemonSetStatus{
					namedWorkload: namedWorkload{name: "collector"},
					status: appsv1.DaemonSetStatus{
						DesiredNumberScheduled: 3,
						NumberAvailable:        3,
					},
				},
			},
			expectedStatus:  corev1.ConditionFalse,
			expectedReason:  "WorkloadsNotReady",
			messageContains: "central",
		},
		"mixed workloads with not-ready daemonset": {
			statuses: []workloadStatus{
				deploymentStatus{
					namedWorkload: namedWorkload{name: "sensor"},
					status: appsv1.DeploymentStatus{
						Conditions: []appsv1.DeploymentCondition{
							{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
						},
					},
				},
				daemonSetStatus{
					namedWorkload: namedWorkload{name: "collector"},
					status: appsv1.DaemonSetStatus{
						DesiredNumberScheduled: 3,
						NumberAvailable:        1,
					},
				},
			},
			expectedStatus:  corev1.ConditionFalse,
			expectedReason:  "WorkloadsNotReady",
			messageContains: "collector",
		},
		"deployment without Available condition is not ready": {
			statuses: []workloadStatus{
				deploymentStatus{
					namedWorkload: namedWorkload{name: "central"},
					status:        appsv1.DeploymentStatus{},
				},
			},
			expectedStatus:  corev1.ConditionFalse,
			expectedReason:  "WorkloadsNotReady",
			messageContains: "central",
		},
		"daemonset with zero desired is not available": {
			statuses: []workloadStatus{
				daemonSetStatus{
					namedWorkload: namedWorkload{name: "collector"},
					status: appsv1.DaemonSetStatus{
						DesiredNumberScheduled: 0,
						NumberAvailable:        0,
					},
				},
			},
			expectedStatus:  corev1.ConditionFalse,
			expectedReason:  "WorkloadsNotReady",
			messageContains: "collector",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			status, reason, message := determineAvailableState(tt.statuses)

			assert.Equal(t, string(tt.expectedStatus), string(status), "status mismatch")
			assert.Equal(t, tt.expectedReason, string(reason), "reason mismatch")
			if tt.messageContains != "" {
				assert.Contains(t, message, tt.messageContains, "message does not contain expected text")
			}
		})
	}
}
