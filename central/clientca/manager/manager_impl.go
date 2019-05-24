package manager

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/stackrox/rox/central/clientca/store"
	"github.com/stackrox/rox/central/tlsconfig"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dberrors"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls/verifier"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	errCertLacksSKI = errors.New("Certificate lacks ID")
	log             = logging.LoggerForModule()
)

type certData struct {
	stored *storage.Certificate
	parsed *x509.Certificate
}

type managerImpl struct {
	store store.Store

	mutex    sync.RWMutex
	allCerts map[string]certData
}

func (m *managerImpl) TLSConfigurer() verifier.TLSConfigurer {
	return verifier.TLSConfigurerFunc(func() (*tls.Config, error) {
		cfg, err := tlsconfig.NewCentralTLSConfigurer().TLSConfig()
		if err != nil {
			return nil, err
		}
		m.mutex.RLock()
		defer m.mutex.RUnlock()
		for _, c := range m.allCerts {
			log.Infof("Adding client CA cert to the TLS trust pool: %q", (c.stored.GetId()))
			cfg.ClientCAs.AddCert(c.parsed)
		}
		return cfg, nil
	})
}

func (m *managerImpl) Initialize() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	all, err := m.store.ListCertificates(context.TODO())
	if err != nil {
		return err
	}
	allCerts := make(map[string]certData)
	for _, cert := range all {
		c, err := helpers.ParseCertificatePEM([]byte(cert.GetPem()))
		if err != nil {
			return err
		}
		allCerts[cert.GetId()] = certData{
			stored: cert,
			parsed: c,
		}
	}
	m.allCerts = allCerts
	return nil
}

func (m *managerImpl) GetAllClientCAs(ctx context.Context) []*storage.Certificate {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	output := make([]*storage.Certificate, len(m.allCerts))
	i := 0
	for _, v := range m.allCerts {
		output[i] = v.stored
		i++
	}
	return output
}

func (m *managerImpl) GetClientCA(ctx context.Context, id string) (*storage.Certificate, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	val, ok := m.allCerts[id]
	return val.stored, ok
}

func (m *managerImpl) RemoveClientCA(ctx context.Context, id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	_, ok := m.allCerts[id]
	if !ok {
		return dberrors.ErrNotFound{Type: "ClientCA", ID: id}
	}
	err := m.store.DeleteCertificate(ctx, id)
	delete(m.allCerts, id)
	return err
}

func (m *managerImpl) AddClientCA(ctx context.Context, certificatePEM string) (*storage.Certificate, error) {
	c, err := helpers.ParseCertificatePEM([]byte(certificatePEM))
	if err != nil {
		return nil, err
	}
	err = validateCACert(c)
	if err != nil {
		return nil, err
	}
	stored := &storage.Certificate{
		Id:  formatID(c.SubjectKeyId),
		Pem: string(certificatePEM),
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()
	err = m.store.UpsertCertificates(ctx, []*storage.Certificate{stored})
	m.allCerts[stored.Id] = certData{
		stored: stored,
		parsed: c,
	}
	certificate := protoutils.CloneStorageCertificate(stored)
	return certificate, err
}

func validateCACert(cert *x509.Certificate) error {
	errorList := errorhelpers.NewErrorList("Validating CA certificate")
	if len(cert.SubjectKeyId) == 0 {
		errorList.AddError(errCertLacksSKI)
	}
	return errorList.ToError()
}
