package localscanner

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/retry"
)

var (
	errCertRefresherForced = errors.New("cert refresher forced error")
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

func (s *certRefresherSuite) TestRefreshCertificatesImmediateRefresh() {
	now := time.Now()
	testCases := map[string]struct {
		newCertsRenewalTime    time.Time
		newCertsRenewalTimeErr error
	}{
		"success":                  {newCertsRenewalTime: now.Add(24 * time.Hour), newCertsRenewalTimeErr: nil},
		"new certificates invalid": {newCertsRenewalTime: time.UnixMilli(0), newCertsRenewalTimeErr: errCertRefresherForced},
	}
	for tcName, tc := range testCases {
		s.Run(tcName, func() {
			storedCertificates := testIssueCertsResponse(1,
				storage.ServiceType_SCANNER_SERVICE, storage.ServiceType_SCANNER_DB_SERVICE).GetCertificates()
			issueCertsResponse := testIssueCertsResponse(2,
				storage.ServiceType_SCANNER_SERVICE, storage.ServiceType_SCANNER_DB_SERVICE)

			s.dependenciesMock.On("getServiceCertificates", mock.Anything).Once().Return(storedCertificates, nil)
			s.dependenciesMock.On("ensureServiceCertificates", mock.Anything,
				issueCertsResponse.GetCertificates()).Once().Return(nil)

			s.dependenciesMock.On("getCertsRenewalTime", storedCertificates).Once().Return(
				// renew immediately first
				now.Add(-1*time.Hour), nil)
			s.dependenciesMock.On("getCertsRenewalTime", issueCertsResponse.GetCertificates()).Once().Return(
				tc.newCertsRenewalTime, tc.newCertsRenewalTimeErr)

			s.dependenciesMock.On("requestCertificates", mock.Anything).Once().Return(issueCertsResponse, nil)

			timeToNextRefresh, err := s.refreshCertificates()

			if tc.newCertsRenewalTimeErr == nil {
				s.Require().NoError(err)
				s.InDelta(time.Until(tc.newCertsRenewalTime).Seconds(), timeToNextRefresh.Seconds(), 1)
			} else {
				s.Require().ErrorIs(err, tc.newCertsRenewalTimeErr)
				s.NotErrorIs(err, concurrency.ErrNonRecoverable)
			}

			s.dependenciesMock.AssertExpectations(s.T())
		})
	}
}

func (s *certRefresherSuite) TestRefreshCertificatesRefreshLater() {
	now := time.Now()
	var certificates *storage.TypedServiceCertificateSet
	expectedRenewalTime := now.Add(time.Hour)
	s.dependenciesMock.On("getServiceCertificates", mock.Anything).Once().Return(certificates, nil)
	s.dependenciesMock.On("getCertsRenewalTime", certificates).Once().Return(expectedRenewalTime, nil)

	timeToNextRefresh, err := s.refreshCertificates()

	s.Require().NoError(err)
	s.InDelta(time.Until(expectedRenewalTime).Seconds(), timeToNextRefresh.Seconds(), 1)
}

func (s *certRefresherSuite) TestRefreshCertificatesGetCertsInconsistentImmediateRefresh() {
	testCases := map[string]struct {
		recoverableErr error
	}{
		"refresh immediately on ErrDifferentCAForDifferentServiceTypes": {recoverableErr: errors.Wrap(ErrDifferentCAForDifferentServiceTypes, "wrap error")},
		"refresh immediately on ErrMissingSecretData":                   {recoverableErr: errors.Wrap(ErrMissingSecretData, "wrap error")},
		"refresh immediately on missing secrets":                        {recoverableErr: k8sErrors.NewNotFound(schema.GroupResource{Group: "Core", Resource: "Secret"}, "foo")},
	}
	for tcName, tc := range testCases {
		s.Run(tcName, func() {
			s.dependenciesMock.On("getServiceCertificates", mock.Anything).Once().Return(
				(*storage.TypedServiceCertificateSet)(nil), tc.recoverableErr)
			s.dependenciesMock.On("requestCertificates", mock.Anything).Return(
				// stop the test here, as we have already checked this recovers from the first getCertsRenewalTime error.
				(*central.IssueLocalScannerCertsResponse)(nil), errCertRefresherForced).Once().Run(func(args mock.Arguments) {
			})

			_, err := s.refreshCertificates()

			s.ErrorIs(err, errCertRefresherForced)
			s.NotErrorIs(err, concurrency.ErrNonRecoverable)
			s.dependenciesMock.AssertExpectations(s.T())
		})
	}
}

func (s *certRefresherSuite) TestRefreshCertificatesGetCertsUnexpectedOwnerHighestPriorityFailure() {
	getErr := multierror.Append(nil, ErrUnexpectedSecretsOwner, ErrDifferentCAForDifferentServiceTypes, ErrMissingSecretData)
	s.dependenciesMock.On("getServiceCertificates", mock.Anything).Once().Return(
		(*storage.TypedServiceCertificateSet)(nil), getErr)

	_, err := s.refreshCertificates()

	s.ErrorIs(err, concurrency.ErrNonRecoverable)
	s.dependenciesMock.AssertExpectations(s.T())
}

func (s *certRefresherSuite) TestRefreshCertificatesGetCertsOtherErrFailure() {
	s.dependenciesMock.On("getServiceCertificates", mock.Anything).Once().Return(
		(*storage.TypedServiceCertificateSet)(nil), errCertRefresherForced)

	_, err := s.refreshCertificates()

	s.ErrorIs(err, errCertRefresherForced)
	s.NotErrorIs(err, concurrency.ErrNonRecoverable)
	s.dependenciesMock.AssertExpectations(s.T())
}

func (s *certRefresherSuite) TestRefreshCertificatesGetTimeToRefreshFailureRecovery() {
	var certificates *storage.TypedServiceCertificateSet
	s.dependenciesMock.On("getServiceCertificates", mock.Anything).Once().Return(certificates, nil)
	s.dependenciesMock.On("getCertsRenewalTime", certificates).Once().Return(time.UnixMilli(0), errCertRefresherForced)
	s.dependenciesMock.On("requestCertificates", mock.Anything).Return(
		// stop the test here, as we have already checked this recovers from the first getCertsRenewalTime error.
		(*central.IssueLocalScannerCertsResponse)(nil), errCertRefresherForced).Once().Run(func(args mock.Arguments) {
	})

	_, err := s.refreshCertificates()

	s.ErrorIs(err, errCertRefresherForced)
	s.NotErrorIs(err, concurrency.ErrNonRecoverable)
	s.dependenciesMock.AssertExpectations(s.T())
}

func (s *certRefresherSuite) TestRefreshCertificatesRequestCertificatesFailure() {
	var certificates *storage.TypedServiceCertificateSet
	s.dependenciesMock.On("getServiceCertificates", mock.Anything).Once().Return(certificates, nil)
	s.dependenciesMock.On("getCertsRenewalTime", certificates).Once().Return(time.UnixMilli(0), nil)
	s.dependenciesMock.On("requestCertificates", mock.Anything).Once().Return(
		(*central.IssueLocalScannerCertsResponse)(nil), errCertRefresherForced)

	_, err := s.refreshCertificates()

	s.ErrorIs(err, errCertRefresherForced)
	s.NotErrorIs(err, concurrency.ErrNonRecoverable)
	s.dependenciesMock.AssertExpectations(s.T())
}

func (s *certRefresherSuite) TestRefreshCertificatesRequestCertificatesResponseFailure() {
	var certificates *storage.TypedServiceCertificateSet
	s.dependenciesMock.On("getServiceCertificates", mock.Anything).Once().Return(certificates, nil)
	s.dependenciesMock.On("getCertsRenewalTime", certificates).Once().Return(time.UnixMilli(0), nil)
	s.dependenciesMock.On("requestCertificates", mock.Anything).Once().Return(&central.IssueLocalScannerCertsResponse{
		Response: &central.IssueLocalScannerCertsResponse_Error{
			Error: &central.LocalScannerCertsIssueError{
				Message: errCertRefresherForced.Error(),
			},
		},
	}, nil)

	_, err := s.refreshCertificates()

	s.Require().Error(err)
	s.Regexp(errCertRefresherForced.Error(), err.Error())
	s.NotErrorIs(err, concurrency.ErrNonRecoverable)
	s.dependenciesMock.AssertExpectations(s.T())
}

func (s *certRefresherSuite) TestRefreshCertificatesEnsureCertsFailure() {
	var certificates *storage.TypedServiceCertificateSet
	s.dependenciesMock.On("getServiceCertificates", mock.Anything).Once().Return(certificates, nil)
	s.dependenciesMock.On("getCertsRenewalTime", certificates).Once().Return(time.UnixMilli(0), nil)
	issueCertsResponse := testIssueCertsResponse(2,
		storage.ServiceType_SCANNER_SERVICE, storage.ServiceType_SCANNER_DB_SERVICE, storage.ServiceType_SCANNER_V4_INDEXER_SERVICE, storage.ServiceType_SCANNER_V4_DB_SERVICE)
	s.dependenciesMock.On("requestCertificates", mock.Anything).Once().Return(issueCertsResponse, nil)
	s.dependenciesMock.On("ensureServiceCertificates", mock.Anything,
		issueCertsResponse.GetCertificates()).Once().Return(errCertRefresherForced)

	_, err := s.refreshCertificates()

	s.ErrorIs(err, errCertRefresherForced)
	s.NotErrorIs(err, concurrency.ErrNonRecoverable)
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

func (m *dependenciesMock) getServiceCertificates(ctx context.Context) (*storage.TypedServiceCertificateSet, error) {
	args := m.Called(ctx)
	return args.Get(0).(*storage.TypedServiceCertificateSet), args.Error(1)
}

func (m *dependenciesMock) ensureServiceCertificates(ctx context.Context, certificates *storage.TypedServiceCertificateSet) ([]*storage.TypedServiceCertificate, error) {
	args := m.Called(ctx, certificates)
	return certificates.ServiceCerts, args.Error(0)
}

func (m *dependenciesMock) getCertsRenewalTime(certificates *storage.TypedServiceCertificateSet) (time.Time, error) {
	args := m.Called(certificates)
	return args.Get(0).(time.Time), args.Error(1)
}
