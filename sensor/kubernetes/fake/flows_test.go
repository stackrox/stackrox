package fake

import (
	"fmt"
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
	for range 1000 {
		generateAndAddIPToPool()
	}
	generateExternalIPPool()

	for range 1000 {
		ip, internal, ok := w.getRandomInternalExternalIP()
		if internal == net.ParseIP(ip).IsPublic() {
			fmt.Println()
			fmt.Println(ip)
			fmt.Println(internal)
			fmt.Println(net.ParseIP(ip).IsPublic())
			fmt.Println()
		}
		s.Equal(true, ok)
		s.Equal(internal, !net.ParseIP(ip).IsPublic())
	}
}
