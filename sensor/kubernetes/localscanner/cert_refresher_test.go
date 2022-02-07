package localscanner

import (
	"context"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	forcedErr = errors.New("forced")
	timeout   = time.Second
	backoff   = wait.Backoff{
		Duration: 100 * time.Millisecond,
		Factor:   1,
		Jitter:   0,
		Steps:    1,
		Cap:      100 * time.Millisecond,
	}
)

func TestCertRefresher(t *testing.T) {
	suite.Run(t, new(certRefresherSuite))
}

type certRefresherSuite struct {
	suite.Suite
	dependenciesMock *dependenciesMock
	refresher        *certRefresherImpl
}

func (s *certRefresherSuite) SetupTest() {
	s.dependenciesMock = &dependenciesMock{}
	s.refresher = newCertRefresher(s.dependenciesMock.requestCertificates, timeout, backoff, s.dependenciesMock)
	s.refresher.createTicker = func() concurrency.RetryTicker {
		return s.dependenciesMock
	}
	s.refresher.getCertsRenewalTime = s.dependenciesMock.getCertsRenewalTime
}

func (s *certRefresherSuite) TestTickerStartedAndStopped() {
	s.dependenciesMock.On("Start").Once()
	s.dependenciesMock.On("Stop").Once()

	s.refresher.Start()
	s.refresher.Stop()

	s.dependenciesMock.AssertExpectations(s.T())
}

func (s *certRefresherSuite) TestRefreshCertificateMissingCertificatesImmediateRefreshSuccess() {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	doneSignal := concurrency.NewErrorSignal()
	now := time.Now()
	certRenewalTime := now.Add(24 * time.Hour)
	storedCertificates := testIssueCertsResponse(1,
		storage.ServiceType_SCANNER_SERVICE, storage.ServiceType_SCANNER_DB_SERVICE).GetCertificates()
	issueCertsResponse := testIssueCertsResponse(2,
		storage.ServiceType_SCANNER_SERVICE, storage.ServiceType_SCANNER_DB_SERVICE)

	s.dependenciesMock.On("Start").Once().Run(func(args mock.Arguments) {
		timeToNextRefresh, err := s.refresher.RefreshCertificates(ctx)
		s.Require().NoError(err)
		s.InDelta(time.Until(certRenewalTime).Seconds(), timeToNextRefresh.Seconds(), 1)
		doneSignal.Signal()
	})

	s.dependenciesMock.On("GetServiceCertificates", mock.Anything).Return(storedCertificates, nil).Once()
	s.dependenciesMock.On("PutServiceCertificates", mock.Anything,
		issueCertsResponse.GetCertificates()).Return(nil).Once()

	s.dependenciesMock.On("getCertsRenewalTime", storedCertificates).Once().Return(
		// renew immediately first
		now.Add(-1*time.Hour), nil)
	s.dependenciesMock.On("getCertsRenewalTime", issueCertsResponse.GetCertificates()).Once().Return(
		certRenewalTime, nil)

	s.dependenciesMock.On("requestCertificates", mock.Anything).Return(issueCertsResponse, nil)

	s.refresher.Start()

	_, ok := doneSignal.WaitWithTimeout(timeout)
	s.Require().True(ok)
	s.dependenciesMock.AssertExpectations(s.T())
}

func (s *certRefresherSuite) TestRefreshCertificateMissingCertificatesImmediateRefreshGetCertsFailure() {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	doneSignal := concurrency.NewErrorSignal()

	s.dependenciesMock.On("Start").Once().Run(func(args mock.Arguments) {
		_, err := s.refresher.RefreshCertificates(ctx)
		s.Error(err)
		doneSignal.Signal()
	})
	s.dependenciesMock.On("GetServiceCertificates", mock.Anything).Return((*storage.TypedServiceCertificateSet)(nil), forcedErr).Once()

	s.refresher.Start()

	_, ok := doneSignal.WaitWithTimeout(timeout)
	s.Require().True(ok)
	s.dependenciesMock.AssertExpectations(s.T())
}

func (s *certRefresherSuite) TestRefreshCertificateMissingCertificatesImmediateRefreshGetTimeToRefreshFailure() {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	doneSignal := concurrency.NewErrorSignal()

	s.dependenciesMock.On("Start").Once().Run(func(args mock.Arguments) {
		_, err := s.refresher.RefreshCertificates(ctx)
		s.Error(err)
		doneSignal.Signal()
	})
	certificates := (*storage.TypedServiceCertificateSet)(nil)
	s.dependenciesMock.On("GetServiceCertificates", mock.Anything).Return(certificates, nil).Once()
	s.dependenciesMock.On("getCertsRenewalTime", certificates).Once().Return(time.UnixMilli(0), forcedErr)

	s.refresher.Start()

	_, ok := doneSignal.WaitWithTimeout(timeout)
	s.Require().True(ok)
	s.dependenciesMock.AssertExpectations(s.T())
}

func (s *certRefresherSuite) TestRefreshCertificateMissingCertificatesImmediateRefreshRequestCertificatesFailure() {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	doneSignal := concurrency.NewErrorSignal()

	s.dependenciesMock.On("Start").Once().Run(func(args mock.Arguments) {
		_, err := s.refresher.RefreshCertificates(ctx)
		s.Error(err)
		doneSignal.Signal()
	})
	certificates := (*storage.TypedServiceCertificateSet)(nil)
	s.dependenciesMock.On("GetServiceCertificates", mock.Anything).Return(certificates, nil).Once()
	s.dependenciesMock.On("getCertsRenewalTime", certificates).Once().Return(time.UnixMilli(0), nil)
	s.dependenciesMock.On("requestCertificates", mock.Anything).Return((*central.IssueLocalScannerCertsResponse)(nil), forcedErr)

	s.refresher.Start()

	_, ok := doneSignal.WaitWithTimeout(timeout)
	s.Require().True(ok)
	s.dependenciesMock.AssertExpectations(s.T())
}

func (s *certRefresherSuite) TestRefreshCertificateMissingCertificatesImmediateRefreshRequestCertificatesErrorResponseFailure() {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	doneSignal := concurrency.NewErrorSignal()

	s.dependenciesMock.On("Start").Once().Run(func(args mock.Arguments) {
		_, err := s.refresher.RefreshCertificates(ctx)
		s.Error(err)
		doneSignal.Signal()
	})

	certificates := (*storage.TypedServiceCertificateSet)(nil)
	s.dependenciesMock.On("GetServiceCertificates", mock.Anything).Return(certificates, nil).Once()
	s.dependenciesMock.On("getCertsRenewalTime", certificates).Once().Return(time.UnixMilli(0), nil)
	s.dependenciesMock.On("requestCertificates", mock.Anything).Return(&central.IssueLocalScannerCertsResponse{
		Response: &central.IssueLocalScannerCertsResponse_Error{
			Error: &central.LocalScannerCertsIssueError{
				Message: forcedErr.Error(),
			},
		},
	}, nil)

	s.refresher.Start()

	_, ok := doneSignal.WaitWithTimeout(timeout)
	s.Require().True(ok)
	s.dependenciesMock.AssertExpectations(s.T())
}

func (s *certRefresherSuite) TestRefreshCertificateMissingCertificatesImmediateRefreshPutCertsFailure() {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	doneSignal := concurrency.NewErrorSignal()

	s.dependenciesMock.On("Start").Once().Run(func(args mock.Arguments) {
		_, err := s.refresher.RefreshCertificates(ctx)
		s.Error(err)
		doneSignal.Signal()
	})

	certificates := (*storage.TypedServiceCertificateSet)(nil)
	s.dependenciesMock.On("GetServiceCertificates", mock.Anything).Return(certificates, nil).Once()
	s.dependenciesMock.On("getCertsRenewalTime", certificates).Once().Return(time.UnixMilli(0), nil)
	issueCertsResponse := testIssueCertsResponse(2,
		storage.ServiceType_SCANNER_SERVICE, storage.ServiceType_SCANNER_DB_SERVICE)
	s.dependenciesMock.On("requestCertificates", mock.Anything).Return(issueCertsResponse, nil)
	s.dependenciesMock.On("PutServiceCertificates", mock.Anything,
		issueCertsResponse.GetCertificates()).Return(forcedErr).Once()

	s.refresher.Start()

	_, ok := doneSignal.WaitWithTimeout(timeout)
	s.Require().True(ok)
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

func (m *dependenciesMock) Start() error {
	m.Called()
	return nil
}

func (m *dependenciesMock) Stop() {
	m.Called()
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
