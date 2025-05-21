//go:build sql_integration

package imagecomponentflat

import (
	"testing"

	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stretchr/testify/suite"
)

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
