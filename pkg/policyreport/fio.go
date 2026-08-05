package policyreport

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// CanonicalizeFIO converts a FileIntegrityNodeStatus resource into SecurityEvents.
// Only "Failed" results produce events; "Succeeded" and "Errored" are ignored
// for the MVP (Errored is an operational issue, not a security finding).
func CanonicalizeFIO(clusterID string, u *unstructured.Unstructured) ([]SecurityEvent, error) {
	nodeName, _, _ := unstructured.NestedString(u.Object, "nodeName")
	if nodeName == "" {
		return nil, fmt.Errorf("FileIntegrityNodeStatus missing nodeName")
	}

	ownerName := u.GetLabels()["file-integrity.openshift.io/owner"]
	if ownerName == "" {
		ownerName = "unknown"
	}

	results, found, err := unstructured.NestedSlice(u.Object, "results")
	if err != nil {
		return nil, fmt.Errorf("FileIntegrityNodeStatus results field error: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("FileIntegrityNodeStatus missing results field")
	}

	var events []SecurityEvent
	for _, r := range results {
		result, ok := r.(map[string]interface{})
		if !ok {
			continue
		}

		condition, _, _ := unstructured.NestedString(result, "condition")
		if condition != "Failed" {
			continue
		}

		filesAdded, _, _ := unstructured.NestedInt64(result, "filesAdded")
		filesChanged, _, _ := unstructured.NestedInt64(result, "filesChanged")
		filesRemoved, _, _ := unstructured.NestedInt64(result, "filesRemoved")

		probeTime := parseFIOTimestamp(result)

		message := fmt.Sprintf("File integrity check failed on node %s: %d files changed, %d added, %d removed",
			nodeName, filesChanged, filesAdded, filesRemoved)

		id := ComputeID(clusterID, string(u.GetUID()), SourceFIO, ownerName, "aide-check", nodeName)

		events = append(events, SecurityEvent{
			ID:         id,
			Source:     SourceFIO,
			Type:       EventTypePolicyResult,
			ObservedAt: probeTime,
			Report: ReportRef{
				APIVersion:      "fileintegrity.openshift.io/v1alpha1",
				Kind:            "FileIntegrityNodeStatus",
				UID:             string(u.GetUID()),
				Name:            u.GetName(),
				Namespace:       u.GetNamespace(),
				ResourceVersion: u.GetResourceVersion(),
			},
			Subject: Subject{
				APIVersion: "v1",
				Kind:       "Node",
				Name:       nodeName,
			},
			ResolvedEntity: ResolvedEntity{
				Type: EntityTypeNode,
				ID:   nodeName,
			},
			Details: PolicyResult{
				ReportedPolicy:   ownerName,
				ReportedRule:     "aide-check",
				Category:         "File Integrity",
				Result:           PolicyResultFail,
				ReportedSeverity: SeverityHigh,
				Message:          message,
				OriginalSource:   "file-integrity-operator",
			},
		})
	}

	return events, nil
}

func parseFIOTimestamp(result map[string]interface{}) time.Time {
	ts, ok := result["lastProbeTime"].(string)
	if !ok || ts == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return time.Time{}
	}
	return t
}
