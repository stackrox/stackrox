package servicecerttoken

import (
	"context"
	"crypto/x509"
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/service"
	"github.com/stackrox/rox/pkg/grpc/common/authn/servicecerttoken"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
)

const (
	declarativeClientCACertDir = "/var/run/stackrox/declarative-client-ca"
)

var (
	log = logging.LoggerForModule()
)

type extractor struct {
	verifyOpts x509.VerifyOptions
	maxLeeway  time.Duration
	validator  authn.ValidateCertChain
}

func (e extractor) IdentityForRequest(ctx context.Context, ri requestinfo.RequestInfo) (authn.Identity, error) {
	token := authn.ExtractToken(ri.Metadata, servicecerttoken.TokenType)
	if token == "" {
		return nil, nil
	}

	cert, err := servicecerttoken.ParseToken(token, e.maxLeeway)
	if err != nil {
		log.Warnf("Could not parse service cert token: %v", err)
		return nil, errors.New("could not parse service cert token")
	}

	verifiedChains, err := cert.Verify(e.verifyOpts)
	if err != nil {
		return nil, errors.Wrap(err, "could not verify certificate")
	}

	if len(verifiedChains) != 1 {
		return nil, errors.Errorf("UNEXPECTED: %d verified chains found", len(verifiedChains))
	}

	if len(verifiedChains[0]) == 0 {
		return nil, errors.New("UNEXPECTED: verified chain is empty")
	}

	chain := requestinfo.ExtractCertInfoChains(verifiedChains)
	if e.validator != nil {
		if err := e.validator.ValidateClientCertificate(ctx, chain[0]); err != nil {
			log.Errorf("Could not validate client certificate from service cert token: %v", err)
			return nil, errors.New("could not validate client certificate from service cert token")
		}
	}

	log.Debugf("Woot! Someone (%s) is authenticating with a service cert token", verifiedChains[0][0].Subject)

	return service.WrapMTLSIdentity(mtls.IdentityFromCert(chain[0][0])), nil
}

// NewExtractorWithCertValidation returns an extractor which allows to configure a cert chain validation
func NewExtractorWithCertValidation(maxLeeway time.Duration, validator authn.ValidateCertChain) (authn.IdentityExtractor, error) {
	authStore := NewTrustStore() // TODO inject in params
	trustPool := x509.NewCertPool()

	trustedCAs, err := authStore.GetTrustedClientCAs()
	if err != nil {
		return nil, errors.Wrap(err, "could not get trusted client CAs")
	}

	for _, ca := range trustedCAs {
		if !trustPool.AppendCertsFromPEM(ca) {
			log.Warnf("Could not add trusted CA to trust pool")
		}
	}

	verifyOpts := x509.VerifyOptions{
		Roots: trustPool,
	}

	return extractor{
		verifyOpts: verifyOpts,
		maxLeeway:  maxLeeway,
		validator:  validator,
	}, nil
}

func NewTrustStore() TrustStore {
	return &aggregatedTrustStore{
		stores: []TrustStore{
			&fileBasedTrustStore{directory: declarativeClientCACertDir},
			&databaseTrustStore{
				// TODO db: db,
			},
			&builtInTrustStore{},
		},
	}
}

// TrustStore is an interface for getting trusted client CAs
type TrustStore interface {
	GetTrustedClientCAs() ([][]byte, error)
}

// fileBasedTrustStore reads trusted client CAs from a directory
type fileBasedTrustStore struct {
	directory string
}

// databaseTrustStore reads trusted client CAs from a database
type databaseTrustStore struct {
	db *sql.DB
}

// builtInTrustStore reads trusted client CAs from the built-in CA
type builtInTrustStore struct{}

// aggregatedTrustStore aggregates multiple auth stores
type aggregatedTrustStore struct {
	stores []TrustStore
}

var _ TrustStore = &fileBasedTrustStore{}
var _ TrustStore = &databaseTrustStore{}
var _ TrustStore = &builtInTrustStore{}
var _ TrustStore = &aggregatedTrustStore{}

func (s *aggregatedTrustStore) GetTrustedClientCAs() ([][]byte, error) {
	var result [][]byte
	wg := sync.WaitGroup{}
	lock := sync.Mutex{}
	for _, store := range s.stores {
		store := store
		wg.Add(1)
		go func() {
			cas, err := store.GetTrustedClientCAs()
			if err == nil {
				lock.Lock()
				result = append(result, cas...)
				lock.Unlock()
			}
			wg.Done()
		}()
	}
	wg.Wait()
	return result, nil
}

func (s *builtInTrustStore) GetTrustedClientCAs() ([][]byte, error) {
	ca, _, err := mtls.CACert()
	if err != nil {
		return nil, err
	}
	return [][]byte{ca.Raw}, nil
}

func (s *fileBasedTrustStore) GetTrustedClientCAs() ([][]byte, error) {
	// todo: cache & watch directory
	var result [][]byte
	err := filepath.Walk(declarativeClientCACertDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		fileBytes, err := os.ReadFile(path)
		if err != nil {
			log.Warnf("Could not read file %s: %v", path, err)
			return nil
		}
		result = append(result, fileBytes)
		return nil
	})
	if err != nil {
		log.Warnf("Could not read declarative client CA directory %s: %v", declarativeClientCACertDir, err)
		return nil, err
	}
	return result, nil
}

func (s *databaseTrustStore) GetTrustedClientCAs() ([][]byte, error) {
	var result [][]byte
	rows, err := s.db.Query("SELECT ca FROM trusted_client_cas")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var ca []byte
		if err := rows.Scan(&ca); err != nil {
			return nil, err
		}
		result = append(result, ca)
	}
	return result, nil
}
