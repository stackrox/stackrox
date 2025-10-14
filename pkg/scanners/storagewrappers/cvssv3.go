package storagewrappers

import (
	"github.com/stackrox/rox/generated/storage"
)

type CVSSV3Wrapper struct {
	*storage.CVSSV3
}

func (w *CVSSV3Wrapper) AsCVSSV3() *storage.CVSSV3 {
	if w == nil {
		return nil
	}
	return w.CVSSV3
}

func (w *CVSSV3Wrapper) SetVector(vector string) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.Vector = vector
}

func (w *CVSSV3Wrapper) SetExploitabilityScore(score float32) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.ExploitabilityScore = score
}

func (w *CVSSV3Wrapper) SetImpactScore(score float32) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.ImpactScore = score
}

func (w *CVSSV3Wrapper) SetAttackVector(attackVector storage.CVSSV3_AttackVector) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.AttackVector = attackVector
}

func (w *CVSSV3Wrapper) SetAttackComplexity(attackComplexity storage.CVSSV3_Complexity) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.AttackComplexity = attackComplexity
}

func (w *CVSSV3Wrapper) SetPrivilegesRequired(privilegesRequired storage.CVSSV3_Privileges) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.PrivilegesRequired = privilegesRequired
}

func (w *CVSSV3Wrapper) SetUserInteraction(userInteraction storage.CVSSV3_UserInteraction) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.UserInteraction = userInteraction
}

func (w *CVSSV3Wrapper) SetScope(scope storage.CVSSV3_Scope) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.Scope = scope
}

func (w *CVSSV3Wrapper) SetConfidentiality(impact storage.CVSSV3_Impact) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.Confidentiality = impact
}

func (w *CVSSV3Wrapper) SetIntegrity(impact storage.CVSSV3_Impact) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.Integrity = impact
}

func (w *CVSSV3Wrapper) SetAvailability(impact storage.CVSSV3_Impact) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.Availability = impact
}

func (w *CVSSV3Wrapper) SetScore(score float32) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.Score = score
}

func (w *CVSSV3Wrapper) SetSeverity(severity storage.CVSSV3_Severity) {
	if w == nil || w.CVSSV3 == nil {
		return
	}
	w.Severity = severity
}

