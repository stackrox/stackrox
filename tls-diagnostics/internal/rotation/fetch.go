package rotation

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/stackrox/rox/pkg/x509utils"
)

const centralTLSSecretName = "central-tls"

// FetchClusterState reads all TLS-relevant secrets from the cluster and
// returns a snapshot suitable for AnalyzeState.
func FetchClusterState(ctx context.Context, client kubernetes.Interface, centralNamespace string, allNamespaces []string) (*ClusterState, error) {
	secret, err := client.CoreV1().Secrets(centralNamespace).Get(ctx, centralTLSSecretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("reading %s/%s: %w", centralNamespace, centralTLSSecretName, err)
	}

	primaryCA, err := parseFirstCert(secret.Data["ca.pem"])
	if err != nil {
		return nil, fmt.Errorf("parsing primary CA from %s/%s: %w", centralNamespace, centralTLSSecretName, err)
	}

	var secondaryCA *CAInfo
	if raw, ok := secret.Data["ca-secondary.pem"]; ok && len(raw) > 0 {
		parsed, err := parseFirstCert(raw)
		if err != nil {
			return nil, fmt.Errorf("parsing secondary CA from %s/%s: %w", centralNamespace, centralTLSSecretName, err)
		}
		secondaryCA = parsed
	}

	secrets, err := collectSecretCheckData(ctx, client, allNamespaces, centralNamespace, primaryCA, secondaryCA)
	if err != nil {
		return nil, err
	}

	return &ClusterState{
		CentralNamespace: centralNamespace,
		PrimaryCA:        primaryCA,
		SecondaryCA:      secondaryCA,
		Secrets:          secrets,
	}, nil
}

func collectSecretCheckData(ctx context.Context, client kubernetes.Interface, namespaces []string, centralNamespace string, primary, secondary *CAInfo) ([]SecretCheckData, error) {
	var result []SecretCheckData

	for _, ns := range namespaces {
		secrets, err := client.CoreV1().Secrets(ns).List(ctx, metav1.ListOptions{
			LabelSelector: "rhacs.redhat.com/tls=true",
		})
		if err != nil {
			return nil, fmt.Errorf("listing secrets in namespace %s: %w", ns, err)
		}

		for _, secret := range secrets.Items {
			scd := SecretCheckData{
				SecretName:   secret.Name,
				Namespace:    secret.Namespace,
				IsCentralTLS: secret.Name == centralTLSSecretName && secret.Namespace == centralNamespace,
			}

			if raw := secret.Data["ca.pem"]; len(raw) > 0 {
				if certs, err := x509utils.ConvertPEMTox509Certs(raw); err == nil && len(certs) > 0 {
					scd.CACert = certs[0]
				}
			}

			if raw := secret.Data["ca-secondary.pem"]; len(raw) > 0 {
				if certs, err := x509utils.ConvertPEMTox509Certs(raw); err == nil && len(certs) > 0 {
					scd.SecondaryCACert = certs[0]
				}
			}

			for _, key := range []string{"cert.pem", "tls.crt"} {
				raw, ok := secret.Data[key]
				if !ok || len(raw) == 0 {
					continue
				}
				certs, err := x509utils.ConvertPEMTox509Certs(raw)
				if err != nil || len(certs) == 0 {
					continue
				}
				leaf := certs[0]
				if leaf.IsCA {
					continue
				}
				scd.LeafCert = leaf
				scd.SignedBy = identifyIssuer(leaf, primary, secondary)
				break
			}

			result = append(result, scd)
		}
	}

	return result, nil
}

func identifyIssuer(cert *x509.Certificate, primary, secondary *CAInfo) string {
	if err := cert.CheckSignatureFrom(primary.Certificate); err == nil {
		return "primary"
	}
	if secondary != nil {
		if err := cert.CheckSignatureFrom(secondary.Certificate); err == nil {
			return "secondary"
		}
	}
	return "unknown"
}

func parseFirstCert(pemData []byte) (*CAInfo, error) {
	if len(pemData) == 0 {
		return nil, fmt.Errorf("empty PEM data")
	}
	certs, err := x509utils.ConvertPEMTox509Certs(pemData)
	if err != nil {
		return nil, err
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates found in PEM data")
	}

	cert := certs[0]
	fp := sha256.Sum256(cert.Raw)

	return &CAInfo{
		Certificate: cert,
		Fingerprint: hex.EncodeToString(fp[:]),
		NotBefore:   cert.NotBefore,
		NotAfter:    cert.NotAfter,
		Subject:     cert.Subject.String(),
	}, nil
}

// NewCAInfo creates a CAInfo from a parsed certificate.
// Exported for use in tests.
func NewCAInfo(cert *x509.Certificate) *CAInfo {
	fp := sha256.Sum256(cert.Raw)
	return &CAInfo{
		Certificate: cert,
		Fingerprint: hex.EncodeToString(fp[:]),
		NotBefore:   cert.NotBefore,
		NotAfter:    cert.NotAfter,
		Subject:     cert.Subject.String(),
	}
}

// IdentifyIssuer determines which CA signed a certificate. Exported for tests
// that build SecretCheckData manually but want accurate SignedBy values.
func IdentifyIssuer(cert *x509.Certificate, primary, secondary *CAInfo) string {
	return identifyIssuer(cert, primary, secondary)
}

// FormatDuration formats a duration in a human-friendly way.
func FormatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days > 365 {
		years := days / 365
		remainingDays := days % 365
		return fmt.Sprintf("%dy %dd", years, remainingDays)
	}
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	return d.Round(time.Minute).String()
}
