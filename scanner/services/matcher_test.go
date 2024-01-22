package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/quay/claircore"
	"github.com/quay/claircore/pkg/cpe"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	indexermocks "github.com/stackrox/rox/scanner/indexer/mocks"
	matchermocks "github.com/stackrox/rox/scanner/matcher/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type matcherServiceTestSuite struct {
	suite.Suite
	ctx         context.Context
	matcherMock *matchermocks.MockMatcher
	indexerMock *indexermocks.MockIndexer
	mockCtrl    *gomock.Controller
}

func TestMatcherServiceSuite(t *testing.T) {
	suite.Run(t, new(matcherServiceTestSuite))
}
func (s *matcherServiceTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.ctx = context.Background()
	s.matcherMock = matchermocks.NewMockMatcher(s.mockCtrl)
	s.indexerMock = indexermocks.NewMockIndexer(s.mockCtrl)
}

func (s *matcherServiceTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *matcherServiceTestSuite) TestAuthz() {
	testutils.AssertAuthzWorks(s.T(), &MatcherService{})
}

func (s *matcherServiceTestSuite) Test_matcherService_NewMatcherService() {
	// when Indexer is nil, empty content is disabled
	srv := NewMatcherService(s.matcherMock, nil)
	s.True(srv.disableEmptyContents)
	// when Indexer is nil, empty content is disabled
	srv = NewMatcherService(s.matcherMock, s.indexerMock)
	s.False(srv.disableEmptyContents)
}

func (s *matcherServiceTestSuite) Test_matcherService_GetVulnerabilities_empty_contents_disbled() {
	// when empty content is disabled and empty contents then error
	srv := NewMatcherService(s.matcherMock, nil)
	res, err := srv.GetVulnerabilities(s.ctx, &v4.GetVulnerabilitiesRequest{
		HashId:   "/v4/containerimage/sample-hash-id",
		Contents: nil,
	})
	s.ErrorContains(err, "empty contents is disabled")
	s.Nil(res)
}

func (s *matcherServiceTestSuite) Test_matcherService_GetVulnerabilities_empty_contents_enabled() {
	emptyCPE := "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"
	emptyNormalizedVersion := v4.NormalizedVersion{
		Kind: "",
		V:    []int32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
	}

	s.Run("when empty content is enable and empty contents then retrieve index report", func() {
		ir := &claircore.IndexReport{}
		s.indexerMock.
			EXPECT().
			GetIndexReport(gomock.Any(), gomock.Eq(hashID)).
			Return(ir, true, nil)
		s.matcherMock.
			EXPECT().
			GetVulnerabilities(gomock.Any(), gomock.Eq(ir)).
			Return(&claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {ID: "1", Name: "Foobar"},
				},
			}, nil)
		srv := NewMatcherService(s.matcherMock, s.indexerMock)
		res, err := srv.GetVulnerabilities(s.ctx, &v4.GetVulnerabilitiesRequest{
			HashId:   hashID,
			Contents: nil,
		})
		s.NoError(err)
		s.Equal(res, &v4.VulnerabilityReport{
			HashId: hashID,
			Contents: &v4.Contents{
				Packages: []*v4.Package{
					{Id: "1", Name: "Foobar", Cpe: emptyCPE, NormalizedVersion: &emptyNormalizedVersion},
				},
			},
		})
	})

	s.Run("when contents is provided then parse index report and return", func() {
		s.matcherMock.
			EXPECT().
			GetVulnerabilities(gomock.Any(), gomock.Eq(&claircore.IndexReport{
				Packages: map[string]*claircore.Package{
					"1": {ID: "1", Name: "Foobar", CPE: cpe.MustUnbind(emptyCPE)},
				},
			})).
			Return(&claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {ID: "1", Name: "Foobar", CPE: cpe.MustUnbind(emptyCPE)},
				},
			}, nil)
		srv := NewMatcherService(s.matcherMock, nil)
		res, err := srv.GetVulnerabilities(s.ctx, &v4.GetVulnerabilitiesRequest{
			HashId: hashID,
			Contents: &v4.Contents{Packages: []*v4.Package{
				{Id: "1", Name: "Foobar", Cpe: emptyCPE},
			}},
		})
		s.NoError(err)
		s.Equal(res, &v4.VulnerabilityReport{
			HashId: hashID,
			Contents: &v4.Contents{
				Packages: []*v4.Package{
					{Id: "1", Name: "Foobar", Cpe: emptyCPE, NormalizedVersion: &emptyNormalizedVersion},
				},
			},
		})
	})
}

func (s *matcherServiceTestSuite) Test_matcherService_GetMetadata() {
	now := time.Now()
	protoNow, err := types.TimestampProto(now)
	s.Require().NoError(err)

	s.matcherMock.
		EXPECT().
		GetLastVulnerabilityUpdate(gomock.Any()).
		Return(now, nil)

	srv := NewMatcherService(s.matcherMock, nil)
	res, err := srv.GetMetadata(s.ctx, &types.Empty{})
	s.NoError(err)
	s.Equal(&v4.Metadata{
		LastVulnerabilityUpdate: protoNow,
	}, res)
}

func (s *matcherServiceTestSuite) Test_matcherService_GetMetadata_error() {
	s.matcherMock.
		EXPECT().
		GetLastVulnerabilityUpdate(gomock.Any()).
		Return(time.Time{}, errors.New("some error"))

	srv := NewMatcherService(s.matcherMock, nil)
	res, err := srv.GetMetadata(s.ctx, &types.Empty{})
	s.Error(err)
	s.Nil(res)
}
