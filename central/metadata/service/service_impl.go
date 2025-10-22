package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"

	cTLS "github.com/google/certificate-transparency-go/tls"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/metadata/centralcapabilities"
	systemInfoStorage "github.com/stackrox/rox/central/systeminfo/store/postgres"
	"github.com/stackrox/rox/central/tlsconfig"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/version"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.Authenticated(): {
			v1.MetadataService_GetDatabaseStatus_FullMethodName,
			v1.MetadataService_GetDatabaseBackupStatus_FullMethodName,
			v1.MetadataService_GetCentralCapabilities_FullMethodName,
		},
		// When this endpoint was public, Sensor relied on it to check Central's
		// availability. While Sensor might not do so today, we need to ensure
		// backward compatibility with older Sensors.
		or.SensorOr(user.Authenticated()): {
			v1.MetadataService_GetMetadata_FullMethodName,
		},
		allow.Anonymous(): {
			v1.MetadataService_TLSChallenge_FullMethodName,
		},
	})

	// Primary leaf certificate caching
	primaryLeafCertOnce sync.Once
	primaryLeafCert     tls.Certificate
	primaryLeafCertErr  error

	// Secondary CA leaf certificate caching
	secondaryCALeafCertOnce sync.Once
	secondaryCALeafCert     tls.Certificate
	secondaryCALeafCertErr  error
)

// CertificateProvider provides certificates for TLS challenge operations
type CertificateProvider interface {
	// GetPrimaryCACert returns the primary CA certificate and its DER bytes
	GetPrimaryCACert() (*x509.Certificate, []byte, error)
	// GetPrimaryLeafCert returns the primary leaf certificate
	GetPrimaryLeafCert() (tls.Certificate, error)
	// GetSecondaryCAForSigning returns the secondary CA for signing operations
	GetSecondaryCAForSigning() (mtls.CA, error)
	// GetSecondaryCACert returns the secondary CA certificate and its DER bytes
	GetSecondaryCACert() (*x509.Certificate, []byte, error)
	// GetSecondaryLeafCert returns an ephemeral leaf certificate from the secondary CA
	// for use only in TLS challenge cryptographic proofs. This certificate is never persisted
	// and exists only in memory, to prevent accidental misuse as a service certificate.
	GetSecondaryLeafCert() (tls.Certificate, error)
}

// defaultCertificateProvider implements CertificateProvider using global mtls functions
type defaultCertificateProvider struct{}

func (p *defaultCertificateProvider) GetPrimaryCACert() (*x509.Certificate, []byte, error) {
	return mtls.CACert()
}

func (p *defaultCertificateProvider) GetPrimaryLeafCert() (tls.Certificate, error) {
	primaryLeafCertOnce.Do(func() {
		cert, err := mtls.LeafCertificateFromFile()
		if err != nil {
			primaryLeafCertErr = err
			return
		}
		primaryLeafCert = cert
	})

	return primaryLeafCert, primaryLeafCertErr
}

func (p *defaultCertificateProvider) GetSecondaryCAForSigning() (mtls.CA, error) {
	return mtls.SecondaryCAForSigning()
}

func (p *defaultCertificateProvider) GetSecondaryCACert() (*x509.Certificate, []byte, error) {
	return mtls.SecondaryCACert()
}

func (p *defaultCertificateProvider) GetSecondaryLeafCert() (tls.Certificate, error) {
	secondaryCALeafCertOnce.Do(func() {
		leafCert, err := issueSecondaryCALeafCert(p)
		if err != nil {
			secondaryCALeafCertErr = err
			return
		}
		secondaryCALeafCert = leafCert
	})

	if secondaryCALeafCertErr != nil {
		return tls.Certificate{}, secondaryCALeafCertErr
	}

	return secondaryCALeafCert, nil
}

func issueSecondaryCALeafCert(certProvider CertificateProvider) (tls.Certificate, error) {
	secondaryCA, err := certProvider.GetSecondaryCAForSigning()
	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "failed to load secondary CA for signing")
	}

	issuedCert, issueErr := secondaryCA.IssueCertForSubject(mtls.CentralSubject)
	if issueErr != nil {
		return tls.Certificate{}, errors.Wrap(issueErr, "failed to issue leaf certificate from secondary CA")
	}

	leafCert, err := tls.X509KeyPair(issuedCert.CertPEM, issuedCert.KeyPEM)
	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "failed to load X509 key pair for temporary leaf cert from secondary CA")
	}

	return leafCert, nil
}

// Service is the struct that manages the Metadata API
type serviceImpl struct {
	v1.UnimplementedMetadataServiceServer

	db              postgres.DB
	systemInfoStore systemInfoStorage.Store
	certProvider    CertificateProvider
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterMetadataServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterMetadataServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetMetadata returns the metadata for Rox.
func (s *serviceImpl) GetMetadata(ctx context.Context, _ *v1.Empty) (*v1.Metadata, error) {
	metadata := &v1.Metadata{
		BuildFlavor:   buildinfo.BuildFlavor,
		ReleaseBuild:  buildinfo.ReleaseBuild,
		LicenseStatus: v1.Metadata_VALID,
	}
	// Only return the version to logged in users, not anonymous users.
	if authn.IdentityFromContextOrNil(ctx) != nil {
		metadata.Version = version.GetMainVersion()
	}
	return metadata, nil
}

// TLSChallenge returns all trusted CAs (i.e. secret/additional-ca) and centrals cert chain. This is necessary if
// central is running behind load balancer with self-signed certificates.
//
// To validate that the list of trust roots comes directly from central and have not been tampered with,
// Central will cryptographically sign it with the private key of its service certificate.
//
// 1. External challenge token, generated by the external service
// 2. Central challenge token, generated by central itself
// 3. Payload (i.e. certificates)
func (s *serviceImpl) TLSChallenge(_ context.Context, req *v1.TLSChallengeRequest) (*v1.TLSChallengeResponse, error) {
	sensorChallenge := req.GetChallengeToken()
	sensorChallengeBytes, err := base64.URLEncoding.DecodeString(sensorChallenge)
	if err != nil {
		return nil, errox.InvalidArgs.CausedByf("challenge token must be a valid base64 string: %v", err)
	}
	if len(sensorChallengeBytes) != centralsensor.ChallengeTokenLength {
		return nil, errox.InvalidArgs.CausedByf("base64 decoded challenge token must be %d bytes long, received challenge %q was %d bytes", centralsensor.ChallengeTokenLength, sensorChallenge, len(sensorChallengeBytes))
	}

	// Create central challenge token
	nonceGenerator := cryptoutils.NewNonceGenerator(centralsensor.ChallengeTokenLength, nil)
	centralChallenge, err := nonceGenerator.Nonce()
	if err != nil {
		return nil, errors.Errorf("Could not create central challenge: %s", err)
	}

	_, caCertDERBytes, err := s.certProvider.GetPrimaryCACert()
	if err != nil {
		return nil, errors.Errorf("Could not read CA cert and private key: %s", err)
	}

	leafCert, err := s.certProvider.GetPrimaryLeafCert()
	if err != nil {
		return nil, errors.Errorf("Could not load leaf certificate: %s", err)
	}

	additionalCAs, err := tlsconfig.GetAdditionalCAs()
	if err != nil {
		return nil, errors.Errorf("reading additional CAs: %s", err)
	}

	// add default leaf cert to additional CAs
	defaultCertChain, err := tlsconfig.MaybeGetDefaultCertChain()
	if err != nil {
		return nil, errors.Errorf("could not read default cert chain: %s", err)
	}
	if len(defaultCertChain) > 0 {
		additionalCAs = append(additionalCAs, defaultCertChain[0])
	}

	// Write trust info to proto struct
	trustInfo := &v1.TrustInfo{
		CentralChallenge: centralChallenge,
		SensorChallenge:  sensorChallenge,
		CertChain: [][]byte{
			leafCert.Certificate[0],
			caCertDERBytes,
		},
		AdditionalCas: additionalCAs,
	}

	// if a secondary CA exists, add its chain to TrustInfo
	secondaryLeafCert, secondaryLeafCertErr := s.certProvider.GetSecondaryLeafCert()
	if secondaryLeafCertErr == nil {
		_, secondaryCACertDERBytes, secondaryCACertErr := s.certProvider.GetSecondaryCACert()

		if secondaryCACertErr == nil {
			trustInfo.SecondaryCertChain = [][]byte{
				secondaryLeafCert.Certificate[0],
				secondaryCACertDERBytes,
			}
		}
	}

	trustInfoBytes, err := trustInfo.MarshalVT()
	if err != nil {
		return nil, errors.Errorf("Could not marshal trust info: %s", err)
	}

	// Create signature from CA key
	sign, err := cTLS.CreateSignature(cryptoutils.DerefPrivateKey(leafCert.PrivateKey), cTLS.SHA256, trustInfoBytes)
	if err != nil {
		return nil, errors.Errorf("Could not sign trust info: %s", err)
	}

	resp := &v1.TLSChallengeResponse{
		Signature:           sign.Signature,
		TrustInfoSerialized: trustInfoBytes,
	}

	// Optionally also sign with the secondary CA
	if secondaryLeafCertErr == nil {
		secondarySign, err := cTLS.CreateSignature(cryptoutils.DerefPrivateKey(secondaryLeafCert.PrivateKey), cTLS.SHA256, trustInfoBytes)
		if err != nil {
			log.Warnf("Failed to create secondary signature (primary signature will still be used): %v", err)
		} else {
			resp.SignatureSecondaryCa = secondarySign.Signature
		}
	}

	return resp, nil
}

// GetDatabaseStatus returns the database status for Rox.
func (s *serviceImpl) GetDatabaseStatus(ctx context.Context, _ *v1.Empty) (*v1.DatabaseStatus, error) {
	dbStatus := &v1.DatabaseStatus{
		DatabaseAvailable: true,
	}

	dbType := v1.DatabaseStatus_PostgresDB
	if err := s.db.Ping(ctx); err != nil {
		dbStatus.DatabaseAvailable = false
		log.Warn("central is unable to communicate with the database.")
		return dbStatus, nil
	}

	dbVersion := globaldb.GetPostgresVersion(ctx, s.db)

	// Only return the database type and version to logged in users, not anonymous users.
	if authn.IdentityFromContextOrNil(ctx) != nil {
		dbStatus.DatabaseVersion = dbVersion
		dbStatus.DatabaseType = dbType
		dbStatus.DatabaseIsExternal = pgconfig.IsExternalDatabase()
	}

	return dbStatus, nil
}

// GetDatabaseBackupStatus return the database backup status.
func (s *serviceImpl) GetDatabaseBackupStatus(ctx context.Context, _ *v1.Empty) (*v1.DatabaseBackupStatus, error) {
	sysInfo, found, err := s.systemInfoStore.Get(ctx)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errox.NotFound
	}
	return &v1.DatabaseBackupStatus{
		BackupInfo: sysInfo.GetBackupInfo(),
	}, nil
}

// GetCentralCapabilities returns central services capabilities.
func (s *serviceImpl) GetCentralCapabilities(_ context.Context, _ *v1.Empty) (*v1.CentralServicesCapabilities, error) {
	return centralcapabilities.GetCentralCapabilities(), nil
}
