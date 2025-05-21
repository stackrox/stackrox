//go:build sql_integration

package imagecomponentflat

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stretchr/testify/suite"
)

type testCase struct {
	desc           string
	ctx            context.Context
	q              *v1.Query
	matchFilter    *filterImpl
	less           lessFunc
	expectedErr    string
	skipCountTests bool
	testOrder      bool
}

type lessFunc func(records []*imageComponentFlatResponse) func(i, j int) bool

type filterImpl struct {
	matchImage func(image *storage.Image) bool
	matchVuln  func(vuln *storage.EmbeddedVulnerability) bool
}

func matchAllFilter() *filterImpl {
	return &filterImpl{
		matchImage: func(_ *storage.Image) bool {
			return true
		},
		matchVuln: func(_ *storage.EmbeddedVulnerability) bool {
			return true
		},
	}
}

func matchNoneFilter() *filterImpl {
	return &filterImpl{
		matchImage: func(_ *storage.Image) bool {
			return false
		},
		matchVuln: func(_ *storage.EmbeddedVulnerability) bool {
			return false
		},
	}
}

func (f *filterImpl) withImageFilter(fn func(image *storage.Image) bool) *filterImpl {
	f.matchImage = fn
	return f
}

func (f *filterImpl) withVulnFilter(fn func(vuln *storage.EmbeddedVulnerability) bool) *filterImpl {
	f.matchVuln = fn
	return f
}

func TestImageComponentFlatView(t *testing.T) {
	// TODO(ROX-29431): Add test for this
	t.Skip("ROX-29431")
	suite.Run(t, new(ImageComponentFlatViewTestSuite))
}

type ImageComponentFlatViewTestSuite struct {
	suite.Suite

	testDB *pgtest.TestPostgres
}

func (s *ImageComponentFlatViewTestSuite) SetupSuite() {
	s.testDB = pgtest.ForT(s.T())
}
