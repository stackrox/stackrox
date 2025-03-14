package fake

import (
	"testing"

	"github.com/stackrox/rox/pkg/net"
	"github.com/stretchr/testify/suite"
)

type flowsSuite struct {
	suite.Suite
}

func TestFlowsSuite(t *testing.T) {
	suite.Run(t, new(flowsSuite))
}

func (s *flowsSuite) TestGetRandomInternalExternalIP() {
	var w WorkloadManager

	_, _, ok := w.getRandomInternalExternalIP()
	s.False(ok)

	_, _, ok = w.getRandomSrcDst()
	s.False(ok)

	for range 1000 {
		generateAndAddIPToPool()
	}

	generateExternalIPPool()

	for range 1000 {
		ip, internal, ok := w.getRandomInternalExternalIP()
		s.True(ok)
		s.Equal(internal, !net.ParseIP(ip).IsPublic())
	}

	for range 1000 {
		src, dst, ok := w.getRandomSrcDst()
		// At least one has to be internal
		s.True(!net.ParseIP(src).IsPublic() || !net.ParseIP(dst).IsPublic())
		s.True(ok)
	}
}
