package tests

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/clientca/manager"
	clientCAStoreMocks "github.com/stackrox/rox/central/clientca/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/suite"
)

var (
	validCA = &storage.Certificate{
		Id: "5E:66:DE:6C:04:C2:4B:49:BF:DD:92:AB:9D:F0:25:31:3C:B0:56:6F",
		Pem: `-----BEGIN CERTIFICATE-----
MIIB6TCCAZCgAwIBAgIUbGOpHZbFSR5YWYw7YctWg7OU6P8wCgYIKoZIzj0EAwIw
UzELMAkGA1UEBhMCVVMxFjAUBgNVBAcTDU1vdW50YWluIFZpZXcxETAPBgNVBAsT
CHRlc3Qgb3BzMRkwFwYDVQQDExBTdGFja1JveCB0ZXN0IENBMB4XDTE5MDUxNTEx
MjAwMFoXDTI0MDUxMzExMjAwMFowUzELMAkGA1UEBhMCVVMxFjAUBgNVBAcTDU1v
dW50YWluIFZpZXcxETAPBgNVBAsTCHRlc3Qgb3BzMRkwFwYDVQQDExBTdGFja1Jv
eCB0ZXN0IENBMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEphDoW19aVRa+1J3C
B6QDZ28dJLasxnx5Afx9w7slRF2Ps8zgv7cLec7SOBIHJ259iZNO5MIlTssDflLt
kEr3yaNCMEAwDgYDVR0PAQH/BAQDAgEGMA8GA1UdEwEB/wQFMAMBAf8wHQYDVR0O
BBYEFF5m3mwEwktJv92Sq53wJTE8sFZvMAoGCCqGSM49BAMCA0cAMEQCIEYGHXBF
zYu1yiu7yST6LmwaZ3teO6TxVnKWjhUyKd1FAiBmrZondCGOlOM7pKNh0nfHx1VC
lOe7Pfuxy92bG6emRw==
-----END CERTIFICATE-----`,
	}
)

type managerTestSuite struct {
	suite.Suite

	mockCtrl  *gomock.Controller
	mockStore *clientCAStoreMocks.MockStore

	mgr manager.ClientCAManager
	ctx context.Context
}

func TestManager(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(managerTestSuite))
}

func (s *managerTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.mockStore = clientCAStoreMocks.NewMockStore(s.mockCtrl)

	s.mgr = manager.New(s.mockStore)
	s.ctx = context.TODO()
}

func (s *managerTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *managerTestSuite) TestInitializeEmpty() {
	s.mockStore.EXPECT().ListCertificates(s.ctx).Return(nil, nil)
	s.Assert().NoError(s.mgr.Initialize())
	all := s.mgr.GetAllClientCAs(s.ctx)
	s.Assert().Empty(all)
}

func (s *managerTestSuite) TestInitializeWithCA() {
	s.mockStore.EXPECT().ListCertificates(s.ctx).Return([]*storage.Certificate{validCA}, nil)
	s.Assert().NoError(s.mgr.Initialize())
	all := s.mgr.GetAllClientCAs(s.ctx)
	s.Assert().Contains(all, validCA)
}

func (s *managerTestSuite) TestAddClientCA() {
	s.mockStore.EXPECT().ListCertificates(s.ctx).Return(nil, nil)
	s.mockStore.EXPECT().UpsertCertificates(s.ctx, []*storage.Certificate{validCA}).Return(nil)
	s.Assert().NoError(s.mgr.Initialize())
	ret, err := s.mgr.AddClientCA(s.ctx, validCA.GetPem())
	s.Assert().NoError(err)
	s.Assert().Equal(validCA.GetId(), ret.GetId())
	all := s.mgr.GetAllClientCAs(s.ctx)
	s.Assert().Equal([]*storage.Certificate{validCA}, all)
}

func (s *managerTestSuite) TestAddEmptyInput() {
	ret, err := s.mgr.AddClientCA(s.ctx, "")
	s.Assert().Nil(ret)
	s.Assert().NotNil(err)
}

func (s *managerTestSuite) TestRemoveClientCA() {
	s.mockStore.EXPECT().ListCertificates(s.ctx).Return([]*storage.Certificate{validCA}, nil)
	s.mockStore.EXPECT().DeleteCertificate(s.ctx, validCA.GetId()).Return(nil)
	s.Assert().NoError(s.mgr.Initialize())
	all := s.mgr.GetAllClientCAs(s.ctx)
	s.Assert().Contains(all, validCA)
	err := s.mgr.RemoveClientCA(s.ctx, validCA.GetId())
	s.Assert().NoError(err)
	all = s.mgr.GetAllClientCAs(s.ctx)
	s.Assert().Empty(all)
}
