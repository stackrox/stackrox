package storagewrappers

import (
	"github.com/stackrox/rox/generated/storage"
)

type CVSSV2Writer interface {
	AsCVSSV2() *storage.CVSSV2
	GetVector() string
	GetScore() float32

	SetVector(vector string)
	SetAttackVector(attackVector storage.CVSSV2_AttackVector)
	SetAccessComplexity(accessComplexity storage.CVSSV2_AccessComplexity)
	SetAuthentication(authentication storage.CVSSV2_Authentication)
	SetConfidentiality(impact storage.CVSSV2_Impact)
	SetIntegrity(impact storage.CVSSV2_Impact)
	SetAvailability(impact storage.CVSSV2_Impact)
	SetExploitabilityScore(score float32)
	SetImpactScore(score float32)
	SetScore(score float32)
	SetSeverity(severity storage.CVSSV2_Severity)
}

type CVSSV2Wrapper struct {
	*storage.CVSSV2
}

var _ CVSSV2Writer = (*CVSSV2Wrapper)(nil)

func (w *CVSSV2Wrapper) AsCVSSV2() *storage.CVSSV2 {
	if w == nil {
		return nil
	}
	return w.CVSSV2
}

func (w *CVSSV2Wrapper) SetVector(vector string) {
	if w == nil || w.CVSSV2 == nil {
		return
	}
	w.Vector = vector
}

func (w *CVSSV2Wrapper) SetAttackVector(attackVector storage.CVSSV2_AttackVector) {
	if w == nil || w.CVSSV2 == nil {
		return
	}
	w.AttackVector = attackVector
}

func (w *CVSSV2Wrapper) SetAccessComplexity(accessComplexity storage.CVSSV2_AccessComplexity) {
	if w == nil || w.CVSSV2 == nil {
		return
	}
	w.AccessComplexity = accessComplexity
}

func (w *CVSSV2Wrapper) SetAuthentication(authentification storage.CVSSV2_Authentication) {
	if w == nil || w.CVSSV2 == nil {
		return
	}
	w.Authentication = authentification
}

func (w *CVSSV2Wrapper) SetConfidentiality(impact storage.CVSSV2_Impact) {
	if w == nil || w.CVSSV2 == nil {
		return
	}
	w.Confidentiality = impact
}

func (w *CVSSV2Wrapper) SetIntegrity(impact storage.CVSSV2_Impact) {
	if w == nil || w.CVSSV2 == nil {
		return
	}
	w.Integrity = impact
}

func (w *CVSSV2Wrapper) SetAvailability(impact storage.CVSSV2_Impact) {
	if w == nil || w.CVSSV2 == nil {
		return
	}
	w.Availability = impact
}

func (w *CVSSV2Wrapper) SetExploitabilityScore(score float32) {
	if w == nil || w.CVSSV2 == nil {
		return
	}
	w.ExploitabilityScore = score
}

func (w *CVSSV2Wrapper) SetImpactScore(score float32) {
	if w == nil || w.CVSSV2 == nil {
		return
	}
	w.ImpactScore = score
}

func (w *CVSSV2Wrapper) SetScore(score float32) {
	if w == nil || w.CVSSV2 == nil {
		return
	}
	w.Score = score
}

func (w *CVSSV2Wrapper) SetSeverity(severity storage.CVSSV2_Severity) {
	if w == nil || w.CVSSV2 == nil {
		return
	}
	w.Severity = severity
}
