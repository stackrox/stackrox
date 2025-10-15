package storagewrappers

import (
	"github.com/stackrox/rox/generated/storage"
)

type CVSSV3Writer interface {
	AsCVSSV3() *storage.CVSSV3
	GetVector() string
	GetScore() float32

	SetVector(vector string)
	SetExploitabilityScore(score float32)
	SetImpactScore(score float32)
	SetAttackVector(attackVector storage.CVSSV3_AttackVector)
	SetAttackComplexity(attackComplexity storage.CVSSV3_Complexity)
	SetPrivilegesRequired(privilegesRequired storage.CVSSV3_Privileges)
	SetUserInteraction(userInteraction storage.CVSSV3_UserInteraction)
	SetScope(scope storage.CVSSV3_Scope)
	SetConfidentiality(impact storage.CVSSV3_Impact)
	SetIntegrity(impact storage.CVSSV3_Impact)
	SetAvailability(impact storage.CVSSV3_Impact)
	SetScore(score float32)
	SetSeverity(severity storage.CVSSV3_Severity)
}

type CVSSV3Wrapper struct {
	*storage.CVSSV3
}

var _ CVSSV3Writer = (*CVSSV3Wrapper)(nil)

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
