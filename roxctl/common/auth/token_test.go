package auth

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/suite"
)

const testTokenVal = "test-token"

func TestToken(t *testing.T) {
	suite.Run(t, new(tokenSuite))
}

type tokenSuite struct {
	suite.Suite
}

func (s *tokenSuite) Test_RetrieveAuthToken_WithEnv() {
	s.T().Setenv(env.TokenEnv.EnvVar(), testTokenVal)

	method := tokenMethod{}
	got, err := method.retrieveAuthToken()

	s.Require().NoError(err)
	s.Equal(got, testTokenVal, "Did not receive correct rox auth token from environment")
}

func (s *tokenSuite) Test_RetrieveAuthToken_ShouldTrimLeadingAndTrailingWhitespace() {
	s.T().Setenv(env.TokenEnv.EnvVar(), fmt.Sprintf(" \n %s \n", testTokenVal))

	method := tokenMethod{}
	got, err := method.retrieveAuthToken()

	s.Require().NoError(err)
	s.Equal(got, testTokenVal, "Expected auth token without whitespaces")
}

func (s *tokenSuite) Test_RetrieveAuthToken_ShouldTrimLeadingAndTrailingWhitespace_Windows() {
	s.T().Setenv(env.TokenEnv.EnvVar(), fmt.Sprintf(" \r %s \r", testTokenVal))

	method := tokenMethod{}
	got, err := method.retrieveAuthToken()

	s.Require().NoError(err)
	s.Equal(got, testTokenVal, "Expected auth token without whitespaces")
}
