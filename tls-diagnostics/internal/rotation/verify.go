package rotation

import (
	"bytes"
	"fmt"
	"time"
)

// verify dispatches to the stage-specific verification flow.
func verify(stage Stage, state *ClusterState, now time.Time) []Finding {
	switch stage {
	case StageNormal:
		return verifyNormal(state, now)
	case StagePreparingRotation:
		return verifyPreparingRotation(state, now)
	case StageRotationInProgress:
		return verifyRotationInProgress(state, now)
	case StageCleanupPending:
		return verifyCleanupPending(state, now)
	case StageRotationNeeded:
		return verifyRotationNeeded(state, now)
	default:
		return nil
	}
}

// --- Stage 1: Normal ---
// Single CA, before 3/5 of validity. No rotation activity.
func verifyNormal(state *ClusterState, now time.Time) []Finding {
	var f []Finding
	f = append(f, checkCANotExpired(state.PrimaryCA, "primary", now)...)

	for _, s := range state.Secrets {
		if s.IsCentralTLS {
			continue
		}
		f = append(f, checkLeafCertValidity(s, now)...)
		f = append(f, checkCACertMatchesPrimary(s, state.PrimaryCA)...)
		f = append(f, checkSignedByExpected(s, "primary")...)
		f = append(f, checkNoSecondaryCACert(s, state.CentralNamespace)...)
	}

	return f
}

// --- Stage 2: PreparingRotation ---
// Secondary CA has been added (newer than primary). Both trusted.
// Central-NS certs signed by primary; secured-cluster certs may be signed by either.
// Central-NS service secrets must contain ca-secondary.pem matching secondary CA.
func verifyPreparingRotation(state *ClusterState, now time.Time) []Finding {
	var f []Finding
	f = append(f, checkCANotExpired(state.PrimaryCA, "primary", now)...)
	f = append(f, checkCANotExpired(state.SecondaryCA, "secondary", now)...)

	for _, s := range state.Secrets {
		if s.IsCentralTLS {
			continue
		}
		f = append(f, checkLeafCertValidity(s, now)...)
		f = append(f, checkCACertMatchesPrimary(s, state.PrimaryCA)...)

		isCentralNS := s.Namespace == state.CentralNamespace
		if isCentralNS {
			f = append(f, checkSignedByExpected(s, "primary")...)
			f = append(f, checkSecondaryCACertPresent(s, state.SecondaryCA)...)
		} else {
			f = append(f, checkSignedByKnown(s)...)
		}
	}

	return f
}

// --- Stage 3: RotationInProgress ---
// Secondary promoted to primary. Primary is the newer CA.
// Old CA (now secondary) still valid and trusted.
// Central-NS certs must be signed by the new primary.
// Secured-cluster certs may lag behind (still signed by old/secondary).
func verifyRotationInProgress(state *ClusterState, now time.Time) []Finding {
	var f []Finding
	f = append(f, checkCANotExpired(state.PrimaryCA, "primary", now)...)
	if now.After(state.SecondaryCA.NotAfter) {
		f = append(f, Finding{
			Severity: SeverityWarn,
			Message:  "secondary (old) CA has expired — cleanup should happen soon",
		})
	}

	for _, s := range state.Secrets {
		if s.IsCentralTLS {
			continue
		}
		f = append(f, checkLeafCertValidity(s, now)...)
		f = append(f, checkCACertMatchesPrimary(s, state.PrimaryCA)...)

		isCentralNS := s.Namespace == state.CentralNamespace
		if isCentralNS {
			f = append(f, checkSignedByExpected(s, "primary")...)
			f = append(f, checkSecondaryCACertPresent(s, state.SecondaryCA)...)
		} else {
			f = append(f, checkSignedByKnown(s)...)
		}
	}

	return f
}

// --- Stage 4: CleanupPending ---
// Secondary CA has expired. Should be removed on next reconciliation.
// All service certs must be signed by primary.
func verifyCleanupPending(state *ClusterState, now time.Time) []Finding {
	var f []Finding
	f = append(f, checkCANotExpired(state.PrimaryCA, "primary", now)...)

	for _, s := range state.Secrets {
		if s.IsCentralTLS {
			continue
		}
		f = append(f, checkLeafCertValidity(s, now)...)
		f = append(f, checkCACertMatchesPrimary(s, state.PrimaryCA)...)
		f = append(f, checkSignedByExpected(s, "primary")...)

		if s.Namespace == state.CentralNamespace {
			f = append(f, checkSecondaryCACertPresent(s, state.SecondaryCA)...)
		}
	}

	return f
}

// --- Stage 5: RotationNeeded ---
// Single CA past the 3/5 threshold. Rotation overdue.
func verifyRotationNeeded(state *ClusterState, now time.Time) []Finding {
	var f []Finding

	if now.After(state.PrimaryCA.NotAfter) {
		f = append(f, Finding{
			Severity: SeverityFail,
			Message:  "primary CA has expired — rotation is critically overdue",
		})
	} else {
		remaining := state.PrimaryCA.NotAfter.Sub(now)
		f = append(f, Finding{
			Severity: SeverityWarn,
			Message:  fmt.Sprintf("CA rotation overdue — primary CA expires in %s", FormatDuration(remaining)),
		})
	}

	for _, s := range state.Secrets {
		if s.IsCentralTLS {
			continue
		}
		f = append(f, checkLeafCertValidity(s, now)...)
		f = append(f, checkCACertMatchesPrimary(s, state.PrimaryCA)...)
		f = append(f, checkSignedByExpected(s, "primary")...)
		f = append(f, checkNoSecondaryCACert(s, state.CentralNamespace)...)
	}

	return f
}

// --- Shared check helpers ---

func checkCANotExpired(ca *CAInfo, label string, now time.Time) []Finding {
	if ca == nil {
		return nil
	}
	if now.After(ca.NotAfter) {
		return []Finding{{
			Severity: SeverityFail,
			Message:  fmt.Sprintf("%s CA has expired (at %s)", label, ca.NotAfter.Format(time.DateTime+" MST")),
		}}
	}
	return nil
}

func checkLeafCertValidity(s SecretCheckData, now time.Time) []Finding {
	if s.LeafCert == nil {
		return nil
	}
	var f []Finding

	if now.After(s.LeafCert.NotAfter) {
		f = append(f, Finding{
			Severity:   SeverityFail,
			SecretName: s.SecretName,
			Namespace:  s.Namespace,
			Message:    fmt.Sprintf("service certificate has expired (at %s)", s.LeafCert.NotAfter.Format(time.DateTime+" MST")),
		})
	} else {
		halfValidity := s.LeafCert.NotBefore.Add(s.LeafCert.NotAfter.Sub(s.LeafCert.NotBefore) / 2)
		if now.After(halfValidity) {
			f = append(f, Finding{
				Severity:   SeverityWarn,
				SecretName: s.SecretName,
				Namespace:  s.Namespace,
				Message:    fmt.Sprintf("service certificate past half of its validity (renewal expected at %s)", halfValidity.Format(time.DateTime+" MST")),
			})
		}
	}

	return f
}

func checkCACertMatchesPrimary(s SecretCheckData, primary *CAInfo) []Finding {
	if s.CACert == nil {
		return nil
	}
	if !bytes.Equal(s.CACert.Raw, primary.Certificate.Raw) {
		return []Finding{{
			Severity:   SeverityFail,
			SecretName: s.SecretName,
			Namespace:  s.Namespace,
			Message:    "ca.pem does not match the primary CA from central-tls",
		}}
	}
	return nil
}

func checkSignedByExpected(s SecretCheckData, expected string) []Finding {
	if s.LeafCert == nil {
		return nil
	}
	if s.SignedBy == "unknown" {
		return []Finding{{
			Severity:   SeverityFail,
			SecretName: s.SecretName,
			Namespace:  s.Namespace,
			Message:    "service certificate not signed by any known CA",
		}}
	}
	if s.SignedBy != expected {
		return []Finding{{
			Severity:   SeverityWarn,
			SecretName: s.SecretName,
			Namespace:  s.Namespace,
			Message:    fmt.Sprintf("service certificate signed by %s CA, expected %s CA", s.SignedBy, expected),
		}}
	}
	return nil
}

func checkSignedByKnown(s SecretCheckData) []Finding {
	if s.LeafCert == nil {
		return nil
	}
	if s.SignedBy == "unknown" {
		return []Finding{{
			Severity:   SeverityFail,
			SecretName: s.SecretName,
			Namespace:  s.Namespace,
			Message:    "service certificate not signed by any known CA",
		}}
	}
	return nil
}

func checkSecondaryCACertPresent(s SecretCheckData, secondary *CAInfo) []Finding {
	if secondary == nil {
		return nil
	}
	if s.SecondaryCACert == nil {
		return []Finding{{
			Severity:   SeverityWarn,
			SecretName: s.SecretName,
			Namespace:  s.Namespace,
			Message:    "missing ca-secondary.pem (expected when secondary CA exists)",
		}}
	}
	if !bytes.Equal(s.SecondaryCACert.Raw, secondary.Certificate.Raw) {
		return []Finding{{
			Severity:   SeverityFail,
			SecretName: s.SecretName,
			Namespace:  s.Namespace,
			Message:    "ca-secondary.pem does not match the secondary CA from central-tls",
		}}
	}
	return nil
}

func checkNoSecondaryCACert(s SecretCheckData, centralNamespace string) []Finding {
	if s.Namespace != centralNamespace {
		return nil
	}
	if s.SecondaryCACert != nil {
		return []Finding{{
			Severity:   SeverityWarn,
			SecretName: s.SecretName,
			Namespace:  s.Namespace,
			Message:    "unexpected ca-secondary.pem present (no secondary CA expected)",
		}}
	}
	return nil
}
