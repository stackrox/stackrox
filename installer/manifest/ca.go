package manifest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/mtls"

	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type CAGenerator struct{}

func (g CAGenerator) Name() string {
	return "Certificate Authority (CA)"
}

func (g CAGenerator) Exportable() bool {
	return true
}

func (g CAGenerator) Generate(ctx context.Context, m *manifestGenerator) ([]Resource, error) {
	certManager := GetCertificateManager()
	
	// Try to load existing CA first
	ca, err := certManager.LoadCA()
	if err != nil {
		// Check if certs don't exist vs actual read error
		if os.IsNotExist(err) {
			// Certs don't exist, generate new ones
			ca, err = certManager.GenerateCA()
			if err != nil {
				return []Resource{}, fmt.Errorf("Error generating CA: %v", err)
			}
		} else {
			// Actual error reading certs
			return []Resource{}, fmt.Errorf("Error loading CA: %v", err)
		}
	}

	fileMap := make(map[string][]byte)
	certgen.AddCAToFileMap(fileMap, ca)

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "additional-ca",
		},
		Data: fileMap,
	}
	secret.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Secret"))

	m.CA = ca

	return []Resource{{
		Object:       secret,
		Name:         secret.Name,
		IsUpdateable: false,
	}}, nil
}

func (g *CAGenerator) GetCA(ctx context.Context, m *manifestGenerator) error {
	secret, err := m.Client.CoreV1().Secrets(m.Config.Namespace).Get(ctx, "additional-ca", metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("Error fetching additional-ca secret: %w", err)
	}

	ca, err := certgen.LoadCAFromFileMap(secret.Data)
	if err != nil {
		return fmt.Errorf("Error loading CA from additional-ca secret: %v", err)
	}

	m.CA = ca
	return nil
}

// CertificateManager handles generating, loading, and saving certificates
type CertificateManager struct {
	ca       mtls.CA
	certPath string
}

var certificateManager *CertificateManager

// InitializeCertificateManager initializes the singleton certificate manager and loads existing CA
func InitializeCertificateManager(config *Config) {
	certificateManager = &CertificateManager{
		certPath: config.CertPath,
	}
	
	// Try to load existing CA from disk (but don't generate a new one)
	if ca, err := certificateManager.loadCAFromDisk(); err == nil && ca != nil {
		certificateManager.ca = ca
		log.Info("Loaded existing CA from disk during initialization")
	}
}

// GetCertificateManager returns the singleton certificate manager
func GetCertificateManager() *CertificateManager {
	return certificateManager
}

// GetCACertificate returns the CA certificate PEM bytes
func (cm *CertificateManager) GetCACertificate() []byte {
	if cm.ca == nil {
		return nil
	}
	return cm.ca.CertPEM()
}

// GenerateCA generates a new CA and saves it to disk
func (cm *CertificateManager) GenerateCA() (mtls.CA, error) {
	// Generate new CA
	ca, err := certgen.GenerateCA()
	if err != nil {
		return nil, fmt.Errorf("failed to generate CA: %w", err)
	}

	cm.ca = ca

	// Save newly generated CA to disk
	if err := cm.saveCAToDisk(ca); err != nil {
		log.Warnf("Failed to save CA to disk: %v", err)
	} else {
		log.Info("Saved new CA to disk")
	}

	return cm.ca, nil
}

// LoadCA loads an existing CA from disk
func (cm *CertificateManager) LoadCA() (mtls.CA, error) {
	if cm.ca != nil {
		return cm.ca, nil
	}

	ca, err := cm.loadCAFromDisk()
	if err != nil {
		return nil, err // Don't wrap the error so os.IsNotExist works
	}

	cm.ca = ca
	log.Info("Loaded existing CA from disk")
	return cm.ca, nil
}

func (cm *CertificateManager) loadCAFromDisk() (mtls.CA, error) {
	certPath := filepath.Join(cm.certPath, "ca-cert.pem")
	keyPath := filepath.Join(cm.certPath, "ca-key.pem")

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}

	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	return mtls.LoadCAForSigning(certPEM, keyPEM)
}

func (cm *CertificateManager) saveCAToDisk(ca mtls.CA) error {
	if err := os.MkdirAll(cm.certPath, 0755); err != nil {
		return fmt.Errorf("failed to create cert directory: %w", err)
	}

	certPath := filepath.Join(cm.certPath, "ca-cert.pem")
	keyPath := filepath.Join(cm.certPath, "ca-key.pem")

	if err := os.WriteFile(certPath, ca.CertPEM(), 0644); err != nil {
		return fmt.Errorf("failed to write CA cert: %w", err)
	}

	if err := os.WriteFile(keyPath, ca.KeyPEM(), 0600); err != nil {
		return fmt.Errorf("failed to write CA key: %w", err)
	}

	return nil
}

func init() {
	central = append(central, CAGenerator{})
	crs = append(crs, CAGenerator{})
}
