package localscanner

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"k8s.io/client-go/util/retry"
)

var (
	errForced = errors.New("forced")
)

func TestCertRefresher(t *testing.T) {
	suite.Run(t, new(certRefresherSuite))
}

type certRefresherSuite struct {
	suite.Suite
	cancel              context.CancelFunc
	dependenciesMock    *dependenciesMock
	refreshCertificates func() (timeToNextRefresh time.Duration, err error)
}

func (s *certRefresherSuite) SetupTest() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	s.cancel = cancel
	s.dependenciesMock = &dependenciesMock{}
	s.refreshCertificates = func() (timeToNextRefresh time.Duration, err error) {
		return refreshCertificates(ctx, s.dependenciesMock.requestCertificates, s.dependenciesMock.getCertsRenewalTime,
			s.dependenciesMock)
	}
}

func (s *certRefresherSuite) TearDownTest() {
	s.cancel()
}

func (s *certRefresherSuite) TestNewCertificatesRefresherSmokeTest() {
	s.NotNil(newCertificatesRefresher(s.dependenciesMock.requestCertificates, s.dependenciesMock,
		time.Second, retry.DefaultBackoff))
}

func (s *certRefresherSuite) TestRefreshCertificatesImmediateRefreshSuccess() {
	now := time.Now()
	certRenewalTime := now.Add(24 * time.Hour)
	storedCertificates := testIssueCertsResponse(1,
		storage.ServiceType_SCANNER_SERVICE, storage.ServiceType_SCANNER_DB_SERVICE).GetCertificates()
	issueCertsResponse := testIssueCertsResponse(2,
		storage.ServiceType_SCANNER_SERVICE, storage.ServiceType_SCANNER_DB_SERVICE)

	s.dependenciesMock.On("GetServiceCertificates", mock.Anything).Once().Return(storedCertificates, nil)
	s.dependenciesMock.On("PutServiceCertificates", mock.Anything,
		issueCertsResponse.GetCertificates()).Once().Return(nil)

	s.dependenciesMock.On("getCertsRenewalTime", storedCertificates).Once().Return(
		// renew immediately first
		now.Add(-1*time.Hour), nil)
	s.dependenciesMock.On("getCertsRenewalTime", issueCertsResponse.GetCertificates()).Once().Return(
		certRenewalTime, nil)

	s.dependenciesMock.On("requestCertificates", mock.Anything).Once().Return(issueCertsResponse, nil)

	timeToNextRefresh, err := s.refreshCertificates()

	s.Require().NoError(err)
	s.InDelta(time.Until(certRenewalTime).Seconds(), timeToNextRefresh.Seconds(), 1)
	s.dependenciesMock.AssertExpectations(s.T())
}

func (s *certRefresherSuite) TestRefreshCertificatesGetCertsFailure() {
	s.dependenciesMock.On("GetServiceCertificates", mock.Anything).Once().Return(
		(*storage.TypedServiceCertificateSet)(nil), errForced)

	_, err := s.refreshCertificates()

	s.Error(err)
	s.dependenciesMock.AssertExpectations(s.T())
}

func (s *certRefresherSuite) TestRefreshCertificatesGetTimeToRefreshFailureRecovery() {
	certificates := (*storage.TypedServiceCertificateSet)(nil)
	s.dependenciesMock.On("GetServiceCertificates", mock.Anything).Once().Return(certificates, nil)
	s.dependenciesMock.On("getCertsRenewalTime", certificates).Once().Return(time.UnixMilli(0), errForced)
	s.dependenciesMock.On("requestCertificates", mock.Anything).Return(
		// stop the test here, as we have already checked this recovers from the first getCertsRenewalTime error.
		(*central.IssueLocalScannerCertsResponse)(nil), errForced).Once().Run(func(args mock.Arguments) {
	})

	_, err := s.refreshCertificates()

	s.Error(err)
	s.dependenciesMock.AssertExpectations(s.T())
}

func (s *certRefresherSuite) TestRefreshCertificatesRequestCertificatesFailure() {
	certificates := (*storage.TypedServiceCertificateSet)(nil)
	s.dependenciesMock.On("GetServiceCertificates", mock.Anything).Once().Return(certificates, nil)
	s.dependenciesMock.On("getCertsRenewalTime", certificates).Once().Return(time.UnixMilli(0), nil)
	s.dependenciesMock.On("requestCertificates", mock.Anything).Once().Return(
		(*central.IssueLocalScannerCertsResponse)(nil), errForced)

	_, err := s.refreshCertificates()

	s.Error(err)

	s.dependenciesMock.AssertExpectations(s.T())
}

func (s *certRefresherSuite) TestRefreshCertificatesRequestCertificatesResponseFailure() {
	certificates := (*storage.TypedServiceCertificateSet)(nil)
	s.dependenciesMock.On("GetServiceCertificates", mock.Anything).Once().Return(certificates, nil)
	s.dependenciesMock.On("getCertsRenewalTime", certificates).Once().Return(time.UnixMilli(0), nil)
	s.dependenciesMock.On("requestCertificates", mock.Anything).Once().Return(&central.IssueLocalScannerCertsResponse{
		Response: &central.IssueLocalScannerCertsResponse_Error{
			Error: &central.LocalScannerCertsIssueError{
				Message: errForced.Error(),
			},
		},
	}, nil)

	_, err := s.refreshCertificates()

	s.Error(err)
	s.dependenciesMock.AssertExpectations(s.T())
}

func (s *certRefresherSuite) TestRefreshCertificatesPutCertsFailure() {
	certificates := (*storage.TypedServiceCertificateSet)(nil)
	s.dependenciesMock.On("GetServiceCertificates", mock.Anything).Once().Return(certificates, nil)
	s.dependenciesMock.On("getCertsRenewalTime", certificates).Once().Return(time.UnixMilli(0), nil)
	issueCertsResponse := testIssueCertsResponse(2,
		storage.ServiceType_SCANNER_SERVICE, storage.ServiceType_SCANNER_DB_SERVICE)
	s.dependenciesMock.On("requestCertificates", mock.Anything).Once().Return(issueCertsResponse, nil)
	s.dependenciesMock.On("PutServiceCertificates", mock.Anything,
		issueCertsResponse.GetCertificates()).Once().Return(errForced)

	_, err := s.refreshCertificates()

	s.Error(err)
	s.dependenciesMock.AssertExpectations(s.T())
}

// testIssueCertsResponse return a test response with certificates for serviceTypes. Different values of seed
// produce different certificates.
func testIssueCertsResponse(seed uint, serviceTypes ...storage.ServiceType) *central.IssueLocalScannerCertsResponse {
	serviceCerts := make([]*storage.TypedServiceCertificate, len(serviceTypes))
	for i, serviceType := range serviceTypes {
		serviceCerts[i] = testServiceCertificate(seed, serviceType)
	}
	return &central.IssueLocalScannerCertsResponse{
		Response: &central.IssueLocalScannerCertsResponse_Certificates{
			Certificates: &storage.TypedServiceCertificateSet{
				CaPem:        make([]byte, 1*seed),
				ServiceCerts: serviceCerts,
			},
		},
	}
}

// testServiceCertificate return a test certificate for the specified serviceType. Different values of seed
// produce different certificates.
func testServiceCertificate(seed uint, serviceType storage.ServiceType) *storage.TypedServiceCertificate {
	return &storage.TypedServiceCertificate{
		ServiceType: serviceType,
		Cert: &storage.ServiceCertificate{
			CertPem: make([]byte, 2*seed),
			KeyPem:  make([]byte, 3*seed),
		},
	}
}

type dependenciesMock struct {
	mock.Mock
}

func (m *dependenciesMock) requestCertificates(ctx context.Context) (*central.IssueLocalScannerCertsResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).(*central.IssueLocalScannerCertsResponse), args.Error(1)
}

func (m *dependenciesMock) GetServiceCertificates(ctx context.Context) (*storage.TypedServiceCertificateSet, error) {
	args := m.Called(ctx)
	return args.Get(0).(*storage.TypedServiceCertificateSet), args.Error(1)
}

func (m *dependenciesMock) PutServiceCertificates(ctx context.Context, certificates *storage.TypedServiceCertificateSet) error {
	return m.Called(ctx, certificates).Error(0)
}

func (m *dependenciesMock) getCertsRenewalTime(certificates *storage.TypedServiceCertificateSet) (time.Time, error) {
	args := m.Called(certificates)
	return args.Get(0).(time.Time), args.Error(1)
}
