package rotation

import (
	"crypto/x509"
	"time"
)

// Stage represents the observed CA rotation lifecycle stage.
type Stage string

const (
	// StageNormal indicates a single active CA, well within its validity period (before 3/5).
	StageNormal Stage = "Normal"

	// StagePreparingRotation indicates a secondary CA has been generated and added.
	// Both CAs are trusted. The secondary is newer and will eventually be promoted.
	StagePreparingRotation Stage = "PreparingRotation"

	// StageRotationInProgress indicates the secondary CA has been promoted to primary.
	// The old CA (now secondary) is still valid and trusted.
	StageRotationInProgress Stage = "RotationInProgress"

	// StageCleanupPending indicates the old CA (now secondary) has expired.
	// It should be removed on the next operator reconciliation.
	StageCleanupPending Stage = "CleanupPending"

	// StageRotationNeeded indicates a single CA exists but has passed the 3/5 validity
	// threshold. The operator should have added a secondary CA but hasn't yet.
	StageRotationNeeded Stage = "RotationNeeded"
)

// Severity indicates how serious a verification finding is.
type Severity string

const (
	SeverityWarn Severity = "WARN"
	SeverityFail Severity = "FAIL"
)

// CAInfo holds parsed information about a CA certificate.
type CAInfo struct {
	Certificate *x509.Certificate `json:"-"`
	Fingerprint string            `json:"fingerprint"`
	NotBefore   time.Time         `json:"notBefore"`
	NotAfter    time.Time         `json:"notAfter"`
	Subject     string            `json:"subject"`
}

// SecretCheckData holds the data extracted from a single TLS secret.
type SecretCheckData struct {
	SecretName      string
	Namespace       string
	IsCentralTLS    bool
	LeafCert        *x509.Certificate
	CACert          *x509.Certificate
	SecondaryCACert *x509.Certificate
	SignedBy        string // "primary", "secondary", or "unknown"
}

// Finding is a single verification result.
type Finding struct {
	Severity   Severity `json:"severity"`
	SecretName string   `json:"secretName,omitempty"`
	Namespace  string   `json:"namespace,omitempty"`
	Message    string   `json:"message"`
}

// ServiceCertIssuer records which CA signed a given service certificate.
type ServiceCertIssuer struct {
	SecretName string `json:"secretName"`
	Namespace  string `json:"namespace"`
	Subject    string `json:"subject"`
	SignedBy   string `json:"signedBy"`
}

// ClusterState holds all pre-fetched data needed for rotation analysis.
// All analysis operates on this snapshot — no I/O or time lookups are
// performed during analysis.
type ClusterState struct {
	CentralNamespace string
	PrimaryCA        *CAInfo
	SecondaryCA      *CAInfo // nil if absent from central-tls
	Secrets          []SecretCheckData
}

// Report contains the full CA rotation diagnostic result.
type Report struct {
	Stage            Stage               `json:"stage"`
	StageDescription string              `json:"stageDescription"`
	PrimaryCA        *CAInfo             `json:"primaryCA,omitempty"`
	SecondaryCA      *CAInfo             `json:"secondaryCA,omitempty"`
	AddSecondaryAt   time.Time           `json:"addSecondaryAt"`
	PromoteAt        time.Time           `json:"promoteAt"`
	NextAction       string              `json:"nextAction"`
	Issuers          []ServiceCertIssuer `json:"issuers,omitempty"`
	Findings         []Finding           `json:"findings,omitempty"`
}

// AnalyzeState performs all analysis on pre-fetched cluster state.
// It is a pure function of its inputs — no I/O, no time lookups.
func AnalyzeState(state *ClusterState, now time.Time) *Report {
	validity := state.PrimaryCA.NotAfter.Sub(state.PrimaryCA.NotBefore)
	addAt := state.PrimaryCA.NotBefore.Add(3 * validity / 5)
	promoteAt := state.PrimaryCA.NotBefore.Add(4 * validity / 5)

	stage, desc, action := determineStage(state.PrimaryCA, state.SecondaryCA, now, addAt, promoteAt)

	report := &Report{
		Stage:            stage,
		StageDescription: desc,
		NextAction:       action,
		PrimaryCA:        state.PrimaryCA,
		SecondaryCA:      state.SecondaryCA,
		AddSecondaryAt:   addAt,
		PromoteAt:        promoteAt,
	}

	for _, s := range state.Secrets {
		if s.LeafCert != nil {
			report.Issuers = append(report.Issuers, ServiceCertIssuer{
				SecretName: s.SecretName,
				Namespace:  s.Namespace,
				Subject:    s.LeafCert.Subject.CommonName,
				SignedBy:   s.SignedBy,
			})
		}
	}

	report.Findings = verify(stage, state, now)

	return report
}

func determineStage(primary, secondary *CAInfo, now, addAt, promoteAt time.Time) (Stage, string, string) {
	if secondary == nil {
		if now.After(promoteAt) {
			return StageRotationNeeded,
				"Single CA, past 4/5 of validity. Both AddSecondary and PromoteSecondary are overdue.",
				"AddSecondaryAndPromote"
		}
		if now.After(addAt) {
			return StageRotationNeeded,
				"Single CA, past 3/5 of validity. A secondary CA should have been added.",
				"AddSecondary"
		}
		return StageNormal,
			"Single CA, within normal validity period. No rotation activity needed.",
			"NoAction"
	}

	if now.After(secondary.NotAfter) {
		return StageCleanupPending,
			"The secondary CA has expired and should be removed.",
			"DeleteSecondary"
	}

	if secondary.NotBefore.After(primary.NotBefore) {
		if now.After(promoteAt) {
			return StagePreparingRotation,
				"Secondary CA added, promotion threshold reached. Promotion is pending.",
				"PromoteSecondary"
		}
		return StagePreparingRotation,
			"Secondary CA has been added. Both CAs are trusted. Promotion will occur at 4/5 of primary validity.",
			"NoAction"
	}

	return StageRotationInProgress,
		"Secondary CA was promoted to primary. The old CA (now secondary) is still valid and trusted.",
		"NoAction"
}
