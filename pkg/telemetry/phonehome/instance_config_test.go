package phonehome

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stretchr/testify/suite"
)

type configTestSuite struct {
	suite.Suite
	mockCtrl *gomock.Controller
}

var _ interface {
	suite.SetupTestSuite
	suite.TearDownTestSuite
} = (*configTestSuite)(nil)

func (s *configTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
}

func (s *configTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func TestConfig(t *testing.T) {
	suite.Run(t, new(configTestSuite))
}

func (s *configTestSuite) TestConfig_GetUserMetadata() {
	config := &Config{
		CentralID: "id",
		Identity:  map[string]any{},
	}

	m := config.GetUserMetadata(nil)
	s.Equal("id", m["CentralId"])
	s.Equal("orgid", m["OrganizationId"])
	s.Equal("tenantid", m["TenantId"])
	s.Equal("unauthenticated", m["UserId"])

	id := mocks.NewMockIdentity(s.mockCtrl)
	id.EXPECT().UID().Times(1).Return("test")

	m = config.GetUserMetadata(id)
	s.Equal("n4bQgYhMfWWaL+qgxVrQFaO/TxsrC4Is0V1sFbDwCgg=", m["UserId"])
}
