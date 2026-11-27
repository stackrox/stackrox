package policyreport

import (
	"crypto/sha256"
	"encoding/hex"
)

// ComputeID derives a stable SecurityEvent identity from the tuple documented
// in security-event-plan.md's "Stable identity" section: cluster ID + report
// UID + normalized source + reported policy name + rule name + subject UID.
//
// Deliberately excludes result-array position and observation timestamp, so
// reordered results and repeated observations of unchanged content produce
// the same ID. If a producer is ever found to emit legitimate duplicates
// under this tuple, add the smallest stable additional discriminator here and
// document why in a test, since any identity change affects alert
// continuity for every existing SecurityEvent.
func ComputeID(clusterID, reportUID string, source Source, reportedPolicy, reportedRule, subjectUID string) string {
	h := sha256.New()
	for _, part := range []string{clusterID, reportUID, string(source), reportedPolicy, reportedRule, subjectUID} {
		h.Write([]byte(part))
		// NUL separator: none of these fields can legitimately contain a NUL
		// byte, so this keeps e.g. ("ab","c") distinguishable from ("a","bc").
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
