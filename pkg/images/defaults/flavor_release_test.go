//go:build release
// +build release

package defaults

import (
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo/testbuildinfo"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/suite"
)

type devPanicTestSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
}

func TestShouldPanic(t *testing.T) {
	suite.Run(t, new(devPanicTestSuite))
}

func (s *devPanicTestSuite) TestShouldPanic() {
	testbuildinfo.SetForTest(s.T())
	testutils.SetExampleVersion(s.T())
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
	s.envIsolator.Setenv(imageFlavorEnvName, "development_build")
	s.Panics(func() { GetImageFlavorFromEnv() })
}
