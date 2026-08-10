package rotation

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCAWithKey struct {
	*CAInfo
	key *ecdsa.PrivateKey
}

func generateTestCAWithKey(t *testing.T, notBefore, notAfter time.Time) *testCAWithKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	return &testCAWithKey{
		CAInfo: NewCAInfo(cert),
		key:    key,
	}
}

func (ca *testCAWithKey) issueLeaf(t *testing.T, notBefore, notAfter time.Time) *x509.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(100),
		NotBefore:    notBefore,
		NotAfter:     notAfter,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, ca.Certificate, &key.PublicKey, ca.key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return cert
}

func makeState(centralNS string, primary, secondary *CAInfo, secrets []SecretCheckData) *ClusterState {
	return &ClusterState{
		CentralNamespace: centralNS,
		PrimaryCA:        primary,
		SecondaryCA:      secondary,
		Secrets:          secrets,
	}
}

func TestVerify_StageNormal(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	primaryCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	leaf := primaryCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", primaryCA.CAInfo, nil, []SecretCheckData{
		{
			SecretName: "scanner-tls",
			Namespace:  "acs-central",
			LeafCert:   leaf,
			CACert:     primaryCA.Certificate,
			SignedBy:   "primary",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StageNormal, report.Stage)
	for _, f := range report.Findings {
		assert.NotEqual(t, SeverityFail, f.Severity, "unexpected FAIL: %s", f.Message)
	}
}

func TestVerify_StageNormal_ExpiredCert(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	primaryCA := generateTestCAWithKey(t, now.Add(-3*365*24*time.Hour), now.Add(2*365*24*time.Hour))
	leaf := primaryCA.issueLeaf(t, now.Add(-2*365*24*time.Hour), now.Add(-24*time.Hour))

	state := makeState("acs-central", primaryCA.CAInfo, nil, []SecretCheckData{
		{
			SecretName: "scanner-tls",
			Namespace:  "acs-central",
			LeafCert:   leaf,
			CACert:     primaryCA.Certificate,
			SignedBy:   "primary",
		},
	})

	report := AnalyzeState(state, now)
	var hasFail bool
	for _, f := range report.Findings {
		if f.Severity == SeverityFail && f.SecretName == "scanner-tls" {
			hasFail = true
		}
	}
	assert.True(t, hasFail, "expected FAIL finding for expired service cert")
}

func TestVerify_StageNormal_UnexpectedSecondaryInServiceSecret(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	primaryCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	spuriousCA := generateTestCAWithKey(t, now, now.Add(5*365*24*time.Hour))
	leaf := primaryCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", primaryCA.CAInfo, nil, []SecretCheckData{
		{
			SecretName:      "scanner-tls",
			Namespace:       "acs-central",
			LeafCert:        leaf,
			CACert:          primaryCA.Certificate,
			SecondaryCACert: spuriousCA.Certificate,
			SignedBy:        "primary",
		},
	})

	report := AnalyzeState(state, now)
	var hasWarn bool
	for _, f := range report.Findings {
		if f.Severity == SeverityWarn && f.SecretName == "scanner-tls" {
			hasWarn = true
		}
	}
	assert.True(t, hasWarn, "expected WARN for unexpected ca-secondary.pem in Normal stage")
}

func TestVerify_StagePreparingRotation_MissingSecondaryInServiceSecret(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	primaryCA := generateTestCAWithKey(t, now.Add(-3*365*24*time.Hour), now.Add(2*365*24*time.Hour))
	secondaryCA := generateTestCAWithKey(t, now, now.Add(5*365*24*time.Hour))
	leaf := primaryCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", primaryCA.CAInfo, secondaryCA.CAInfo, []SecretCheckData{
		{
			SecretName: "scanner-tls",
			Namespace:  "acs-central",
			LeafCert:   leaf,
			CACert:     primaryCA.Certificate,
			SignedBy:   "primary",
		},
	})

	report := AnalyzeState(state, now)
	var hasWarn bool
	for _, f := range report.Findings {
		if f.Severity == SeverityWarn && f.SecretName == "scanner-tls" {
			hasWarn = true
		}
	}
	assert.True(t, hasWarn, "expected WARN for missing ca-secondary.pem during PreparingRotation")
}

func TestVerify_StagePreparingRotation_SecuredClusterSkipsSecondaryCheck(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	primaryCA := generateTestCAWithKey(t, now.Add(-3*365*24*time.Hour), now.Add(2*365*24*time.Hour))
	secondaryCA := generateTestCAWithKey(t, now, now.Add(5*365*24*time.Hour))
	leaf := primaryCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", primaryCA.CAInfo, secondaryCA.CAInfo, []SecretCheckData{
		{
			SecretName: "tls-cert-sensor",
			Namespace:  "acs-sensor",
			LeafCert:   leaf,
			CACert:     primaryCA.Certificate,
			SignedBy:   "primary",
		},
	})

	report := AnalyzeState(state, now)
	for _, f := range report.Findings {
		if f.SecretName == "tls-cert-sensor" {
			assert.NotEqual(t, SeverityFail, f.Severity, "unexpected FAIL for secured cluster secret: %s", f.Message)
			assert.NotContains(t, f.Message, "ca-secondary.pem", "should not check ca-secondary.pem in secured cluster namespace")
		}
	}
}

func TestVerify_StageRotationInProgress_CACertMismatch(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	oldCA := generateTestCAWithKey(t, now.Add(-4*365*24*time.Hour), now.Add(1*365*24*time.Hour))
	newCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	leaf := newCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", newCA.CAInfo, oldCA.CAInfo, []SecretCheckData{
		{
			SecretName:      "scanner-tls",
			Namespace:       "acs-central",
			LeafCert:        leaf,
			CACert:          oldCA.Certificate, // stale: still has old CA as ca.pem
			SecondaryCACert: oldCA.Certificate,
			SignedBy:        "primary",
		},
	})

	report := AnalyzeState(state, now)
	var hasFail bool
	for _, f := range report.Findings {
		if f.Severity == SeverityFail && f.SecretName == "scanner-tls" {
			hasFail = true
		}
	}
	assert.True(t, hasFail, "expected FAIL when ca.pem doesn't match the (new) primary CA")
}

func TestVerify_UnknownIssuer(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	primaryCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	rogueCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	leaf := rogueCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", primaryCA.CAInfo, nil, []SecretCheckData{
		{
			SecretName: "scanner-tls",
			Namespace:  "acs-central",
			LeafCert:   leaf,
			CACert:     primaryCA.Certificate,
			SignedBy:   "unknown",
		},
	})

	report := AnalyzeState(state, now)
	var hasFail bool
	for _, f := range report.Findings {
		if f.Severity == SeverityFail && f.SecretName == "scanner-tls" {
			hasFail = true
		}
	}
	assert.True(t, hasFail, "expected FAIL for cert signed by unknown CA")
}

func TestVerify_StageRotationNeeded(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	primaryCA := generateTestCAWithKey(t, now.Add(-4*365*24*time.Hour), now.Add(1*365*24*time.Hour))
	leaf := primaryCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", primaryCA.CAInfo, nil, []SecretCheckData{
		{
			SecretName: "scanner-tls",
			Namespace:  "acs-central",
			LeafCert:   leaf,
			CACert:     primaryCA.Certificate,
			SignedBy:   "primary",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StageRotationNeeded, report.Stage)
	var hasWarn bool
	for _, f := range report.Findings {
		if f.Severity == SeverityWarn {
			hasWarn = true
		}
	}
	assert.True(t, hasWarn, "expected WARN about overdue rotation")
}

func TestVerify_StageCleanupPending(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	newCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	oldCA := generateTestCAWithKey(t, now.Add(-6*365*24*time.Hour), now.Add(-1*24*time.Hour))
	leaf := newCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", newCA.CAInfo, oldCA.CAInfo, []SecretCheckData{
		{
			SecretName:      "scanner-tls",
			Namespace:       "acs-central",
			LeafCert:        leaf,
			CACert:          newCA.Certificate,
			SecondaryCACert: oldCA.Certificate,
			SignedBy:        "primary",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StageCleanupPending, report.Stage)
	for _, f := range report.Findings {
		if f.SecretName == "scanner-tls" {
			assert.NotEqual(t, SeverityFail, f.Severity, "unexpected FAIL: %s", f.Message)
		}
	}
}

// --- helpers for finding assertions ---

func findFindings(findings []Finding, severity Severity, secretName string) []Finding {
	var matched []Finding
	for _, f := range findings {
		if f.Severity == severity && (secretName == "" || f.SecretName == secretName) {
			matched = append(matched, f)
		}
	}
	return matched
}

func requireFinding(t *testing.T, findings []Finding, severity Severity, secretName, msgSubstring string) {
	t.Helper()
	for _, f := range findings {
		if f.Severity == severity && (secretName == "" || f.SecretName == secretName) {
			if msgSubstring == "" || assert.ObjectsAreEqual(true, true) {
				if msgSubstring == "" {
					return
				}
				if contains(f.Message, msgSubstring) {
					return
				}
			}
		}
	}
	t.Errorf("expected %s finding for secret %q containing %q, got: %v", severity, secretName, msgSubstring, findings)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (substr == "" || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- StageNormal: inconsistency tests ---

func TestVerify_StageNormal_CACertMismatch(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	primaryCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	otherCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	leaf := primaryCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", primaryCA.CAInfo, nil, []SecretCheckData{
		{
			SecretName: "scanner-tls",
			Namespace:  "acs-central",
			LeafCert:   leaf,
			CACert:     otherCA.Certificate,
			SignedBy:   "primary",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StageNormal, report.Stage)
	requireFinding(t, report.Findings, SeverityFail, "scanner-tls", "ca.pem does not match")
}

func TestVerify_StageNormal_LeafPastHalfValidity(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	primaryCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	leaf := primaryCA.issueLeaf(t, now.Add(-200*24*time.Hour), now.Add(100*24*time.Hour))

	state := makeState("acs-central", primaryCA.CAInfo, nil, []SecretCheckData{
		{
			SecretName: "scanner-tls",
			Namespace:  "acs-central",
			LeafCert:   leaf,
			CACert:     primaryCA.Certificate,
			SignedBy:   "primary",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StageNormal, report.Stage)
	requireFinding(t, report.Findings, SeverityWarn, "scanner-tls", "past half of its validity")
}

func TestVerify_StageNormal_CertSignedByWrongCA(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	primaryCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	leaf := primaryCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", primaryCA.CAInfo, nil, []SecretCheckData{
		{
			SecretName: "scanner-tls",
			Namespace:  "acs-central",
			LeafCert:   leaf,
			CACert:     primaryCA.Certificate,
			SignedBy:   "secondary",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StageNormal, report.Stage)
	requireFinding(t, report.Findings, SeverityWarn, "scanner-tls", "signed by secondary CA, expected primary")
}

func TestVerify_StageNormal_MultipleIssues(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	primaryCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	otherCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	expiredLeaf := primaryCA.issueLeaf(t, now.Add(-2*365*24*time.Hour), now.Add(-24*time.Hour))
	rogueLeaf := otherCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", primaryCA.CAInfo, nil, []SecretCheckData{
		{
			SecretName: "scanner-tls",
			Namespace:  "acs-central",
			LeafCert:   expiredLeaf,
			CACert:     primaryCA.Certificate,
			SignedBy:   "primary",
		},
		{
			SecretName: "central-db-tls",
			Namespace:  "acs-central",
			LeafCert:   rogueLeaf,
			CACert:     otherCA.Certificate,
			SignedBy:   "unknown",
		},
	})

	report := AnalyzeState(state, now)
	requireFinding(t, report.Findings, SeverityFail, "scanner-tls", "expired")
	requireFinding(t, report.Findings, SeverityFail, "central-db-tls", "not signed by any known CA")
	requireFinding(t, report.Findings, SeverityFail, "central-db-tls", "ca.pem does not match")
}

// --- StagePreparingRotation: inconsistency tests ---

func TestVerify_StagePreparingRotation_SecondaryCACertWrongCert(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	primaryCA := generateTestCAWithKey(t, now.Add(-3*365*24*time.Hour), now.Add(2*365*24*time.Hour))
	secondaryCA := generateTestCAWithKey(t, now, now.Add(5*365*24*time.Hour))
	wrongCA := generateTestCAWithKey(t, now, now.Add(5*365*24*time.Hour))
	leaf := primaryCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", primaryCA.CAInfo, secondaryCA.CAInfo, []SecretCheckData{
		{
			SecretName:      "scanner-tls",
			Namespace:       "acs-central",
			LeafCert:        leaf,
			CACert:          primaryCA.Certificate,
			SecondaryCACert: wrongCA.Certificate,
			SignedBy:        "primary",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StagePreparingRotation, report.Stage)
	requireFinding(t, report.Findings, SeverityFail, "scanner-tls", "ca-secondary.pem does not match")
}

func TestVerify_StagePreparingRotation_CentralNSCertSignedBySecondary(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	primaryCA := generateTestCAWithKey(t, now.Add(-3*365*24*time.Hour), now.Add(2*365*24*time.Hour))
	secondaryCA := generateTestCAWithKey(t, now, now.Add(5*365*24*time.Hour))
	leaf := secondaryCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", primaryCA.CAInfo, secondaryCA.CAInfo, []SecretCheckData{
		{
			SecretName:      "scanner-tls",
			Namespace:       "acs-central",
			LeafCert:        leaf,
			CACert:          primaryCA.Certificate,
			SecondaryCACert: secondaryCA.Certificate,
			SignedBy:        "secondary",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StagePreparingRotation, report.Stage)
	requireFinding(t, report.Findings, SeverityWarn, "scanner-tls", "signed by secondary CA, expected primary")
}

func TestVerify_StagePreparingRotation_CACertMismatch(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	primaryCA := generateTestCAWithKey(t, now.Add(-3*365*24*time.Hour), now.Add(2*365*24*time.Hour))
	secondaryCA := generateTestCAWithKey(t, now, now.Add(5*365*24*time.Hour))
	wrongCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	leaf := primaryCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", primaryCA.CAInfo, secondaryCA.CAInfo, []SecretCheckData{
		{
			SecretName:      "scanner-tls",
			Namespace:       "acs-central",
			LeafCert:        leaf,
			CACert:          wrongCA.Certificate,
			SecondaryCACert: secondaryCA.Certificate,
			SignedBy:        "primary",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StagePreparingRotation, report.Stage)
	requireFinding(t, report.Findings, SeverityFail, "scanner-tls", "ca.pem does not match")
}

func TestVerify_StagePreparingRotation_SecuredClusterUnknownIssuer(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	primaryCA := generateTestCAWithKey(t, now.Add(-3*365*24*time.Hour), now.Add(2*365*24*time.Hour))
	secondaryCA := generateTestCAWithKey(t, now, now.Add(5*365*24*time.Hour))
	rogueCA := generateTestCAWithKey(t, now, now.Add(5*365*24*time.Hour))
	leaf := rogueCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", primaryCA.CAInfo, secondaryCA.CAInfo, []SecretCheckData{
		{
			SecretName: "tls-cert-sensor",
			Namespace:  "acs-sensor",
			LeafCert:   leaf,
			CACert:     primaryCA.Certificate,
			SignedBy:   "unknown",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StagePreparingRotation, report.Stage)
	requireFinding(t, report.Findings, SeverityFail, "tls-cert-sensor", "not signed by any known CA")
}

func TestVerify_StagePreparingRotation_SecuredClusterSignedByEitherIsOK(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	primaryCA := generateTestCAWithKey(t, now.Add(-3*365*24*time.Hour), now.Add(2*365*24*time.Hour))
	secondaryCA := generateTestCAWithKey(t, now, now.Add(5*365*24*time.Hour))
	leafByPrimary := primaryCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))
	leafBySecondary := secondaryCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", primaryCA.CAInfo, secondaryCA.CAInfo, []SecretCheckData{
		{
			SecretName: "tls-cert-sensor",
			Namespace:  "acs-sensor",
			LeafCert:   leafByPrimary,
			CACert:     primaryCA.Certificate,
			SignedBy:   "primary",
		},
		{
			SecretName: "tls-cert-collector",
			Namespace:  "acs-sensor",
			LeafCert:   leafBySecondary,
			CACert:     primaryCA.Certificate,
			SignedBy:   "secondary",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StagePreparingRotation, report.Stage)
	for _, f := range report.Findings {
		if f.SecretName == "tls-cert-sensor" || f.SecretName == "tls-cert-collector" {
			assert.NotContains(t, f.Message, "not signed by any known CA",
				"secured cluster certs signed by either CA should be accepted")
		}
	}
}

// --- StageRotationInProgress: inconsistency tests ---

func TestVerify_StageRotationInProgress_CentralNSCertSignedByOldCA(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	oldCA := generateTestCAWithKey(t, now.Add(-4*365*24*time.Hour), now.Add(1*365*24*time.Hour))
	newCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	leaf := oldCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", newCA.CAInfo, oldCA.CAInfo, []SecretCheckData{
		{
			SecretName:      "scanner-tls",
			Namespace:       "acs-central",
			LeafCert:        leaf,
			CACert:          newCA.Certificate,
			SecondaryCACert: oldCA.Certificate,
			SignedBy:        "secondary",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StageRotationInProgress, report.Stage)
	requireFinding(t, report.Findings, SeverityWarn, "scanner-tls", "signed by secondary CA, expected primary")
}

func TestVerify_StageRotationInProgress_SecuredClusterUnknownIssuer(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	oldCA := generateTestCAWithKey(t, now.Add(-4*365*24*time.Hour), now.Add(1*365*24*time.Hour))
	newCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	rogueCA := generateTestCAWithKey(t, now, now.Add(5*365*24*time.Hour))
	leaf := rogueCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", newCA.CAInfo, oldCA.CAInfo, []SecretCheckData{
		{
			SecretName: "tls-cert-sensor",
			Namespace:  "acs-sensor",
			LeafCert:   leaf,
			CACert:     newCA.Certificate,
			SignedBy:   "unknown",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StageRotationInProgress, report.Stage)
	requireFinding(t, report.Findings, SeverityFail, "tls-cert-sensor", "not signed by any known CA")
}

func TestVerify_StageRotationInProgress_MissingSecondaryCACert(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	oldCA := generateTestCAWithKey(t, now.Add(-4*365*24*time.Hour), now.Add(1*365*24*time.Hour))
	newCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	leaf := newCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", newCA.CAInfo, oldCA.CAInfo, []SecretCheckData{
		{
			SecretName: "scanner-tls",
			Namespace:  "acs-central",
			LeafCert:   leaf,
			CACert:     newCA.Certificate,
			SignedBy:   "primary",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StageRotationInProgress, report.Stage)
	requireFinding(t, report.Findings, SeverityWarn, "scanner-tls", "missing ca-secondary.pem")
}

func TestVerify_StageRotationInProgress_SecondaryCAExpired(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	// Old CA expired but secondary is still "present" in central-tls.
	// This state means secondary.NotBefore < primary.NotBefore AND secondary is expired.
	// But determineStage checks secondary.NotAfter first — if expired, it's CleanupPending.
	// So for RotationInProgress with expired secondary, we can't get there via determineStage.
	// However, the verifyRotationInProgress function does check for it explicitly.
	// We test the verify function directly.
	oldCA := generateTestCAWithKey(t, now.Add(-6*365*24*time.Hour), now.Add(-1*24*time.Hour))
	newCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	leaf := newCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", newCA.CAInfo, oldCA.CAInfo, []SecretCheckData{
		{
			SecretName:      "scanner-tls",
			Namespace:       "acs-central",
			LeafCert:        leaf,
			CACert:          newCA.Certificate,
			SecondaryCACert: oldCA.Certificate,
			SignedBy:        "primary",
		},
	})

	findings := verify(StageRotationInProgress, state, now)
	var hasWarn bool
	for _, f := range findings {
		if f.Severity == SeverityWarn && contains(f.Message, "cleanup should happen") {
			hasWarn = true
		}
	}
	assert.True(t, hasWarn, "expected WARN that secondary CA expired and cleanup should happen")
}

func TestVerify_StageRotationInProgress_SecuredClusterSignedByOldCAIsOK(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	oldCA := generateTestCAWithKey(t, now.Add(-4*365*24*time.Hour), now.Add(1*365*24*time.Hour))
	newCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	leaf := oldCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", newCA.CAInfo, oldCA.CAInfo, []SecretCheckData{
		{
			SecretName: "tls-cert-sensor",
			Namespace:  "acs-sensor",
			LeafCert:   leaf,
			CACert:     newCA.Certificate,
			SignedBy:   "secondary",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StageRotationInProgress, report.Stage)
	for _, f := range report.Findings {
		if f.SecretName == "tls-cert-sensor" {
			assert.NotContains(t, f.Message, "not signed by any known CA",
				"secured cluster cert signed by old (secondary) CA should be accepted during rotation")
		}
	}
}

// --- StageCleanupPending: inconsistency tests ---

func TestVerify_StageCleanupPending_CertStillSignedByOldCA(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	newCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	oldCA := generateTestCAWithKey(t, now.Add(-6*365*24*time.Hour), now.Add(-1*24*time.Hour))
	leaf := newCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", newCA.CAInfo, oldCA.CAInfo, []SecretCheckData{
		{
			SecretName:      "scanner-tls",
			Namespace:       "acs-central",
			LeafCert:        leaf,
			CACert:          newCA.Certificate,
			SecondaryCACert: oldCA.Certificate,
			SignedBy:        "secondary",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StageCleanupPending, report.Stage)
	requireFinding(t, report.Findings, SeverityWarn, "scanner-tls", "signed by secondary CA, expected primary")
}

func TestVerify_StageCleanupPending_CACertMismatch(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	newCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	oldCA := generateTestCAWithKey(t, now.Add(-6*365*24*time.Hour), now.Add(-1*24*time.Hour))
	wrongCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	leaf := newCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", newCA.CAInfo, oldCA.CAInfo, []SecretCheckData{
		{
			SecretName:      "scanner-tls",
			Namespace:       "acs-central",
			LeafCert:        leaf,
			CACert:          wrongCA.Certificate,
			SecondaryCACert: oldCA.Certificate,
			SignedBy:        "primary",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StageCleanupPending, report.Stage)
	requireFinding(t, report.Findings, SeverityFail, "scanner-tls", "ca.pem does not match")
}

func TestVerify_StageCleanupPending_UnknownIssuer(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	newCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	oldCA := generateTestCAWithKey(t, now.Add(-6*365*24*time.Hour), now.Add(-1*24*time.Hour))
	rogueCA := generateTestCAWithKey(t, now, now.Add(5*365*24*time.Hour))
	leaf := rogueCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", newCA.CAInfo, oldCA.CAInfo, []SecretCheckData{
		{
			SecretName:      "scanner-tls",
			Namespace:       "acs-central",
			LeafCert:        leaf,
			CACert:          newCA.Certificate,
			SecondaryCACert: oldCA.Certificate,
			SignedBy:        "unknown",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StageCleanupPending, report.Stage)
	requireFinding(t, report.Findings, SeverityFail, "scanner-tls", "not signed by any known CA")
}

func TestVerify_StageCleanupPending_SecondaryCACertMismatch(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	newCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	oldCA := generateTestCAWithKey(t, now.Add(-6*365*24*time.Hour), now.Add(-1*24*time.Hour))
	wrongCA := generateTestCAWithKey(t, now.Add(-5*365*24*time.Hour), now.Add(-2*24*time.Hour))
	leaf := newCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", newCA.CAInfo, oldCA.CAInfo, []SecretCheckData{
		{
			SecretName:      "scanner-tls",
			Namespace:       "acs-central",
			LeafCert:        leaf,
			CACert:          newCA.Certificate,
			SecondaryCACert: wrongCA.Certificate,
			SignedBy:        "primary",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StageCleanupPending, report.Stage)
	requireFinding(t, report.Findings, SeverityFail, "scanner-tls", "ca-secondary.pem does not match")
}

// --- StageRotationNeeded: inconsistency tests ---

func TestVerify_StageRotationNeeded_CAExpired(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	primaryCA := generateTestCAWithKey(t, now.Add(-6*365*24*time.Hour), now.Add(-1*24*time.Hour))
	leaf := primaryCA.issueLeaf(t, now.Add(-2*365*24*time.Hour), now.Add(-24*time.Hour))

	state := makeState("acs-central", primaryCA.CAInfo, nil, []SecretCheckData{
		{
			SecretName: "scanner-tls",
			Namespace:  "acs-central",
			LeafCert:   leaf,
			CACert:     primaryCA.Certificate,
			SignedBy:   "primary",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StageRotationNeeded, report.Stage)
	requireFinding(t, report.Findings, SeverityFail, "", "critically overdue")
}

func TestVerify_StageRotationNeeded_UnexpectedSecondaryCACert(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	primaryCA := generateTestCAWithKey(t, now.Add(-4*365*24*time.Hour), now.Add(1*365*24*time.Hour))
	spuriousCA := generateTestCAWithKey(t, now, now.Add(5*365*24*time.Hour))
	leaf := primaryCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", primaryCA.CAInfo, nil, []SecretCheckData{
		{
			SecretName:      "scanner-tls",
			Namespace:       "acs-central",
			LeafCert:        leaf,
			CACert:          primaryCA.Certificate,
			SecondaryCACert: spuriousCA.Certificate,
			SignedBy:        "primary",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StageRotationNeeded, report.Stage)
	requireFinding(t, report.Findings, SeverityWarn, "scanner-tls", "unexpected ca-secondary.pem")
}

func TestVerify_StageRotationNeeded_UnknownIssuer(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	primaryCA := generateTestCAWithKey(t, now.Add(-4*365*24*time.Hour), now.Add(1*365*24*time.Hour))
	rogueCA := generateTestCAWithKey(t, now, now.Add(5*365*24*time.Hour))
	leaf := rogueCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", primaryCA.CAInfo, nil, []SecretCheckData{
		{
			SecretName: "scanner-tls",
			Namespace:  "acs-central",
			LeafCert:   leaf,
			CACert:     primaryCA.Certificate,
			SignedBy:   "unknown",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StageRotationNeeded, report.Stage)
	requireFinding(t, report.Findings, SeverityFail, "scanner-tls", "not signed by any known CA")
}

func TestVerify_StageRotationNeeded_CACertMismatch(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	primaryCA := generateTestCAWithKey(t, now.Add(-4*365*24*time.Hour), now.Add(1*365*24*time.Hour))
	wrongCA := generateTestCAWithKey(t, now.Add(-3*365*24*time.Hour), now.Add(2*365*24*time.Hour))
	leaf := primaryCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", primaryCA.CAInfo, nil, []SecretCheckData{
		{
			SecretName: "scanner-tls",
			Namespace:  "acs-central",
			LeafCert:   leaf,
			CACert:     wrongCA.Certificate,
			SignedBy:   "primary",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StageRotationNeeded, report.Stage)
	requireFinding(t, report.Findings, SeverityFail, "scanner-tls", "ca.pem does not match")
}

// --- Cross-namespace tests ---

func TestVerify_StageNormal_SecuredClusterIgnoresSecondaryCACertCheck(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	primaryCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	spuriousCA := generateTestCAWithKey(t, now, now.Add(5*365*24*time.Hour))
	leaf := primaryCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", primaryCA.CAInfo, nil, []SecretCheckData{
		{
			SecretName:      "tls-cert-sensor",
			Namespace:       "acs-sensor",
			LeafCert:        leaf,
			CACert:          primaryCA.Certificate,
			SecondaryCACert: spuriousCA.Certificate,
			SignedBy:        "primary",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StageNormal, report.Stage)
	for _, f := range report.Findings {
		if f.SecretName == "tls-cert-sensor" {
			assert.NotContains(t, f.Message, "ca-secondary.pem",
				"secured cluster namespace secrets should not be checked for ca-secondary.pem in Normal stage")
		}
	}
}

func TestVerify_CentralTLSSecretIsSkippedByVerification(t *testing.T) {
	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	primaryCA := generateTestCAWithKey(t, now.Add(-1*365*24*time.Hour), now.Add(4*365*24*time.Hour))
	rogueCA := generateTestCAWithKey(t, now, now.Add(5*365*24*time.Hour))
	leaf := rogueCA.issueLeaf(t, now.Add(-24*time.Hour), now.Add(364*24*time.Hour))

	state := makeState("acs-central", primaryCA.CAInfo, nil, []SecretCheckData{
		{
			SecretName:   "central-tls",
			Namespace:    "acs-central",
			IsCentralTLS: true,
			LeafCert:     leaf,
			CACert:       rogueCA.Certificate,
			SignedBy:     "unknown",
		},
	})

	report := AnalyzeState(state, now)
	assert.Equal(t, StageNormal, report.Stage)
	assert.Empty(t, report.Findings, "central-tls secret itself should be skipped by verification checks")
}
