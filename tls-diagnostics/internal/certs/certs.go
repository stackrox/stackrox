package certs

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/stackrox/rox/pkg/x509utils"
)

var skipKeys = map[string]bool{
	"key.pem":          true,
	"ca-key.pem":       true,
	"jwt-key.pem":      true,
	"tls.key":          true,
	"ca-secondary.pem": false,
}

type CertInfo struct {
	SecretName      string    `json:"secretName"`
	SecretType      string    `json:"secretType"`
	Namespace       string    `json:"namespace"`
	DataKey         string    `json:"dataKey"`
	Subject         string    `json:"subject"`
	Issuer          string    `json:"issuer"`
	CommonName      string    `json:"commonName"`
	SANs            []string  `json:"sans,omitempty"`
	SerialNumber    string    `json:"serialNumber"`
	NotBefore       time.Time `json:"notBefore"`
	NotAfter        time.Time `json:"notAfter"`
	IsExpired       bool      `json:"isExpired"`
	IsCA            bool      `json:"isCA"`
	Algorithm       string    `json:"algorithm"`
	PostQuantumSafe bool      `json:"postQuantumSafe"`
	Fingerprint     string    `json:"fingerprint"`
}

type SecretReport struct {
	SecretName string     `json:"secretName"`
	Namespace  string     `json:"namespace"`
	SecretType string     `json:"secretType"`
	Certs      []CertInfo `json:"certs"`
}

func Collect(ctx context.Context, client kubernetes.Interface, namespaces []string) ([]SecretReport, error) {
	var reports []SecretReport

	for _, ns := range namespaces {
		secrets, err := client.CoreV1().Secrets(ns).List(ctx, metav1.ListOptions{
			LabelSelector: "rhacs.redhat.com/tls=true",
		})
		if err != nil {
			return nil, fmt.Errorf("listing secrets in namespace %s: %w", ns, err)
		}

		for _, secret := range secrets.Items {
			report := SecretReport{
				SecretName: secret.Name,
				Namespace:  secret.Namespace,
				SecretType: string(secret.Type),
			}

			keys := sortedKeys(secret.Data)
			for _, key := range keys {
				if skipKeys[key] {
					continue
				}

				certs, err := x509utils.ConvertPEMTox509Certs(secret.Data[key])
				if err != nil {
					continue
				}

				for _, cert := range certs {
					info := buildCertInfo(secret.Name, string(secret.Type), secret.Namespace, key, cert)
					report.Certs = append(report.Certs, info)
				}
			}

			if len(report.Certs) > 0 {
				reports = append(reports, report)
			}
		}
	}

	sort.Slice(reports, func(i, j int) bool {
		if reports[i].Namespace != reports[j].Namespace {
			return reports[i].Namespace < reports[j].Namespace
		}
		return reports[i].SecretName < reports[j].SecretName
	})

	return reports, nil
}

func buildCertInfo(secretName, secretType, namespace, dataKey string, cert *x509.Certificate) CertInfo {
	now := time.Now()
	fingerprint := sha256.Sum256(cert.Raw)

	var sans []string
	for _, dns := range cert.DNSNames {
		sans = append(sans, dns)
	}
	for _, ip := range cert.IPAddresses {
		sans = append(sans, ip.String())
	}

	return CertInfo{
		SecretName:      secretName,
		SecretType:      secretType,
		Namespace:       namespace,
		DataKey:         dataKey,
		Subject:         cert.Subject.String(),
		Issuer:          cert.Issuer.String(),
		CommonName:      cert.Subject.CommonName,
		SANs:            sans,
		SerialNumber:    cert.SerialNumber.Text(16),
		NotBefore:       cert.NotBefore,
		NotAfter:        cert.NotAfter,
		IsExpired:       now.After(cert.NotAfter),
		IsCA:            cert.IsCA,
		Algorithm:       describeAlgorithm(cert),
		PostQuantumSafe: isPostQuantumSafe(cert),
		Fingerprint:     hex.EncodeToString(fingerprint[:]),
	}
}

func describeAlgorithm(cert *x509.Certificate) string {
	switch pub := cert.PublicKey.(type) {
	case *ecdsa.PublicKey:
		return fmt.Sprintf("ECDSA %s", pub.Curve.Params().Name)
	case *rsa.PublicKey:
		return fmt.Sprintf("RSA %d", pub.N.BitLen())
	case ed25519.PublicKey:
		return "Ed25519"
	default:
		return cert.PublicKeyAlgorithm.String()
	}
}

func isPostQuantumSafe(cert *x509.Certificate) bool {
	switch cert.SignatureAlgorithm {
	case x509.PureEd25519:
		return false
	}
	switch cert.PublicKeyAlgorithm {
	case x509.Ed25519:
		return false
	case x509.ECDSA, x509.RSA, x509.DSA:
		return false
	}
	// Unknown/future algorithms (e.g. ML-DSA, SLH-DSA) are conservatively
	// treated as not PQ-safe until Go's x509 package supports them with
	// dedicated constants that can be checked here.
	return false
}

func sortedKeys(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		if !strings.HasSuffix(k, ".pem") && k != "tls.crt" && k != "tls.key" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
