package grpc

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

type APIServerSuite struct {
	suite.Suite
}

func Test_APIServerSuite(t *testing.T) {
	suite.Run(t, new(APIServerSuite))
}

func (a *APIServerSuite) TestEnvValues() {
	cases := map[string]int{
		"":         defaultMaxResponseMsgSize,
		"notAnInt": defaultMaxResponseMsgSize,
		"1337":     1337,
	}

	for envValue, expected := range cases {
		a.Run(fmt.Sprintf("%s=%d", envValue, expected), func() {
			a.T().Setenv(maxResponseMsgSizeSetting.EnvVar(), envValue)
			a.Assert().Equal(expected, maxResponseMsgSize())
		})
	}
}
