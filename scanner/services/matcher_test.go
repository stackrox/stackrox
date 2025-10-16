package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/quay/claircore"
	"github.com/quay/claircore/toolkit/types/cpe"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
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
	testutils.AssertAuthzWorks(s.T(), &matcherService{})
}

func (s *matcherServiceTestSuite) Test_matcherService_NewMatcherService() {
	// when Indexer is nil, empty content is disabled
	srv := NewMatcherService(s.matcherMock, nil)
	s.True(srv.disableEmptyContents)
	// when Indexer is nil, empty content is disabled
	srv = NewMatcherService(s.matcherMock, s.indexerMock)
	s.False(srv.disableEmptyContents)
}

func (s *matcherServiceTestSuite) Test_matcherService_GetVulnerabilities_empty_contents_disabled() {
	// when empty content is disabled and empty contents then error
	srv := NewMatcherService(s.matcherMock, nil)
	s.matcherMock.
		EXPECT().
		Initialized(gomock.Any()).
		Return(nil)
	gvr := &v4.GetVulnerabilitiesRequest{}
	gvr.SetHashId("/v4/containerimage/sample-hash-id")
	gvr.ClearContents()
	res, err := srv.GetVulnerabilities(s.ctx, gvr)
	s.ErrorContains(err, "empty contents is disabled")
	s.Nil(res)
}

func (s *matcherServiceTestSuite) Test_matcherService_GetVulnerabilities_not_initialized() {
	// when matcher is not initialized then error
	srv := NewMatcherService(s.matcherMock, nil)
	s.matcherMock.
		EXPECT().
		Initialized(gomock.Any()).
		Return(errors.New("not initialized"))
	gvr := &v4.GetVulnerabilitiesRequest{}
	gvr.SetHashId("/v4/containerimage/sample-hash-id")
	gvr.ClearContents()
	res, err := srv.GetVulnerabilities(s.ctx, gvr)
	s.ErrorContains(err, "not initialized")
	s.Nil(res)
}

func (s *matcherServiceTestSuite) Test_matcherService_GetVulnerabilities_empty_contents_enabled() {
	emptyCPE := "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"
	emptyNormalizedVersion := &v4.NormalizedVersion{}
	emptyNormalizedVersion.SetKind("")
	emptyNormalizedVersion.SetV([]int32{0, 0, 0, 0, 0, 0, 0, 0, 0, 0})

	s.Run("when empty content is enable and empty contents then retrieve index report", func() {
		ir := &claircore.IndexReport{Success: true}
		s.indexerMock.
			EXPECT().
			GetIndexReport(gomock.Any(), gomock.Eq(hashID), false).
			Return(ir, true, nil)
		s.matcherMock.
			EXPECT().
			GetVulnerabilities(gomock.Any(), gomock.Eq(ir)).
			Return(&claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {ID: "1", Name: "Foobar"},
				},
			}, nil)
		s.matcherMock.
			EXPECT().
			Initialized(gomock.Any()).
			Return(nil)
		srv := NewMatcherService(s.matcherMock, s.indexerMock)
		gvr := &v4.GetVulnerabilitiesRequest{}
		gvr.SetHashId(hashID)
		gvr.ClearContents()
		res, err := srv.GetVulnerabilities(s.ctx, gvr)
		s.NoError(err)
		protoassert.Equal(s.T(), v4.VulnerabilityReport_builder{
			HashId: hashID,
			Contents: v4.Contents_builder{
				Packages: map[string]*v4.Package{
					"1": v4.Package_builder{Id: "1", Name: "Foobar", Cpe: emptyCPE, NormalizedVersion: &emptyNormalizedVersion}.Build(),
				},
				PackagesDEPRECATED: []*v4.Package{
					v4.Package_builder{Id: "1", Name: "Foobar", Cpe: emptyCPE, NormalizedVersion: &emptyNormalizedVersion}.Build(),
				},
			}.Build(),
			Notes: []v4.VulnerabilityReport_Note{v4.VulnerabilityReport_NOTE_OS_UNKNOWN},
		}.Build(), res)

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
		s.matcherMock.
			EXPECT().
			Initialized(gomock.Any()).
			Return(nil)
		srv := NewMatcherService(s.matcherMock, nil)
		res, err := srv.GetVulnerabilities(s.ctx, v4.GetVulnerabilitiesRequest_builder{
			HashId: hashID,
			Contents: v4.Contents_builder{
				Packages: map[string]*v4.Package{
					"1": v4.Package_builder{Id: "1", Name: "Foobar", Cpe: emptyCPE}.Build(),
				},
			}.Build(),
		}.Build())
		s.NoError(err)
		protoassert.Equal(s.T(), v4.VulnerabilityReport_builder{
			HashId: hashID,
			Contents: v4.Contents_builder{
				Packages: map[string]*v4.Package{
					"1": v4.Package_builder{Id: "1", Name: "Foobar", Cpe: emptyCPE, NormalizedVersion: &emptyNormalizedVersion}.Build(),
				},
				PackagesDEPRECATED: []*v4.Package{
					v4.Package_builder{Id: "1", Name: "Foobar", Cpe: emptyCPE, NormalizedVersion: &emptyNormalizedVersion}.Build(),
				},
			}.Build(),
			Notes: []v4.VulnerabilityReport_Note{v4.VulnerabilityReport_NOTE_OS_UNKNOWN},
		}.Build(), res)

	})
}

func (s *matcherServiceTestSuite) Test_matcherService_GetMetadata() {
	now := time.Now()
	protoNow, err := protocompat.ConvertTimeToTimestampOrError(now)
	s.Require().NoError(err)

	s.matcherMock.
		EXPECT().
		GetLastVulnerabilityUpdate(gomock.Any()).
		Return(now, nil)

	srv := NewMatcherService(s.matcherMock, nil)
	res, err := srv.GetMetadata(s.ctx, protocompat.ProtoEmpty())
	s.NoError(err)
	metadata := &v4.Metadata{}
	metadata.SetLastVulnerabilityUpdate(protoNow)
	protoassert.Equal(s.T(), metadata, res)

}

func (s *matcherServiceTestSuite) Test_matcherService_GetMetadata_error() {
	s.matcherMock.
		EXPECT().
		GetLastVulnerabilityUpdate(gomock.Any()).
		Return(time.Time{}, errors.New("some error"))

	srv := NewMatcherService(s.matcherMock, nil)
	res, err := srv.GetMetadata(s.ctx, protocompat.ProtoEmpty())
	s.Error(err)
	s.Nil(res)
}

func (s *matcherServiceTestSuite) Test_matcherService_notes() {
	dists := []claircore.Distribution{
		{
			DID:       "rhel",
			VersionID: "8",
			Version:   "8",
		},
		{
			DID:       "rhel",
			VersionID: "9",
			Version:   "9",
		},
		{
			DID:       "ubuntu",
			VersionID: "22.04",
			Version:   "22.04 (Jammy)",
		},
		{
			DID:       "debian",
			VersionID: "10",
			Version:   "10 (buster)",
		},
		{
			DID:       "alpine",
			VersionID: "",
			Version:   "3.17",
		},
		{
			DID:       "alpine",
			VersionID: "",
			Version:   "3.18",
		},
		{
			DID:       "alpine",
			VersionID: "3.19",
			Version:   "",
		},
	}

	srv := NewMatcherService(s.matcherMock, nil)

	// Empty notes.
	s.matcherMock.
		EXPECT().
		GetKnownDistributions(gomock.Any()).
		Return(dists)
	notes := srv.notes(s.ctx, v4.VulnerabilityReport_builder{
		Contents: v4.Contents_builder{
			Distributions: map[string]*v4.Distribution{
				"0": v4.Distribution_builder{
					Did:       "alpine",
					VersionId: "3.18",
				}.Build(),
			},
		}.Build(),
	}.Build())
	s.Empty(notes)

	// Unsupported OS.
	s.matcherMock.
		EXPECT().
		GetKnownDistributions(gomock.Any()).
		Return(dists)
	notes = srv.notes(s.ctx, v4.VulnerabilityReport_builder{
		Contents: v4.Contents_builder{
			Distributions: map[string]*v4.Distribution{
				"1": v4.Distribution_builder{
					Did:       "debian",
					VersionId: "8",
				}.Build(),
			},
		}.Build(),
	}.Build())
	s.ElementsMatch([]v4.VulnerabilityReport_Note{v4.VulnerabilityReport_NOTE_OS_UNSUPPORTED}, notes)

	// No known OSes is the same as unsupported.
	s.matcherMock.
		EXPECT().
		GetKnownDistributions(gomock.Any()).
		Return([]claircore.Distribution{})
	notes = srv.notes(s.ctx, v4.VulnerabilityReport_builder{
		Contents: v4.Contents_builder{
			Distributions: map[string]*v4.Distribution{
				"2": v4.Distribution_builder{
					Did:       "alpine",
					VersionId: "3.18",
				}.Build(),
			},
		}.Build(),
	}.Build())
	s.ElementsMatch([]v4.VulnerabilityReport_Note{v4.VulnerabilityReport_NOTE_OS_UNSUPPORTED}, notes)

	// Unknown OS.
	notes = srv.notes(s.ctx, v4.VulnerabilityReport_builder{
		Contents: v4.Contents_builder{
			Distributions: map[string]*v4.Distribution{
				"2": v4.Distribution_builder{
					Did:       "alpine",
					VersionId: "3.18",
				}.Build(),
				"3": v4.Distribution_builder{
					Did:       "alpine",
					VersionId: "3.19",
				}.Build(),
			},
		}.Build(),
	}.Build())
	s.ElementsMatch([]v4.VulnerabilityReport_Note{v4.VulnerabilityReport_NOTE_OS_UNKNOWN}, notes)
}

func (s *matcherServiceTestSuite) Test_matcherService_GetSBOM() {
	s.Run("error on empty request", func() {
		srv := NewMatcherService(nil, nil)
		_, err := srv.GetSBOM(s.ctx, nil)
		s.ErrorContains(err, "empty request")
	})

	s.Run("error on no id", func() {
		srv := NewMatcherService(nil, nil)
		_, err := srv.GetSBOM(s.ctx, &v4.GetSBOMRequest{})
		s.ErrorContains(err, "id is required")
	})

	s.Run("error on no name", func() {
		srv := NewMatcherService(nil, nil)
		gsbomr := &v4.GetSBOMRequest{}
		gsbomr.SetId("id")
		_, err := srv.GetSBOM(s.ctx, gsbomr)
		s.ErrorContains(err, "name is required")
	})

	s.Run("error on no uri", func() {
		srv := NewMatcherService(nil, nil)
		gsbomr := &v4.GetSBOMRequest{}
		gsbomr.SetId("id")
		gsbomr.SetName("name")
		_, err := srv.GetSBOM(s.ctx, gsbomr)
		s.ErrorContains(err, "uri is required")
	})

	s.Run("error on empty contents", func() {
		srv := NewMatcherService(nil, nil)
		gsbomr := &v4.GetSBOMRequest{}
		gsbomr.SetId("id")
		gsbomr.SetName("name")
		gsbomr.SetUri("uri")
		_, err := srv.GetSBOM(s.ctx, gsbomr)
		s.ErrorContains(err, "contents are required")

	})

	s.Run("error when sbom generation fails", func() {
		s.matcherMock.EXPECT().GetSBOM(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("broken"))
		srv := NewMatcherService(s.matcherMock, nil)
		gsbomr := &v4.GetSBOMRequest{}
		gsbomr.SetId("id")
		gsbomr.SetName("name")
		gsbomr.SetUri("uri")
		gsbomr.SetContents(&v4.Contents{})
		_, err := srv.GetSBOM(s.ctx, gsbomr)
		s.ErrorContains(err, "broken")
	})

	s.Run("success", func() {
		fakeSbomB := []byte("fake sbom")
		s.matcherMock.EXPECT().GetSBOM(gomock.Any(), gomock.Any(), gomock.Any()).Return(fakeSbomB, nil)
		srv := NewMatcherService(s.matcherMock, nil)
		gsbomr := &v4.GetSBOMRequest{}
		gsbomr.SetId("id")
		gsbomr.SetName("name")
		gsbomr.SetUri("uri")
		gsbomr.SetContents(&v4.Contents{})
		res, err := srv.GetSBOM(s.ctx, gsbomr)
		s.Require().NoError(err)
		s.Equal(res.GetSbom(), fakeSbomB)
	})
}
