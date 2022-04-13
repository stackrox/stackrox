package common

import (
	"fmt"
	"testing"

	"github.com/stackrox/stackrox/pkg/env"
	"github.com/stackrox/stackrox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

const testTokenVal = "test-token"

func TestToken(t *testing.T) {
	t.Parallel()

	suite.Run(t, new(tokenSuite))
}

type tokenSuite struct {
	suite.Suite

	envIsolator *envisolator.EnvIsolator
}

func (s *tokenSuite) SetupTest() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
}

func (s *tokenSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
}

func (s *tokenSuite) Test_RetrieveAuthToken_WithEnv() {
	s.envIsolator.Setenv(env.TokenEnv.EnvVar(), testTokenVal)

	got, err := retrieveAuthToken()

	s.Require().NoError(err)
	s.Equal(got, testTokenVal, "Did not receive correct rox auth token from environment")
}

func (s *tokenSuite) Test_RetrieveAuthToken_ShouldTrimLeadingAndTrailingWhitespace() {
	s.envIsolator.Setenv(env.TokenEnv.EnvVar(), fmt.Sprintf(" \n %s \n", testTokenVal))

	got, err := retrieveAuthToken()

	s.Require().NoError(err)
	s.Equal(got, testTokenVal, "Expected auth token without whitespaces")
}

func (s *tokenSuite) Test_RetrieveAuthToken_ShouldTrimLeadingAndTrailingWhitespace_Windows() {
	s.envIsolator.Setenv(env.TokenEnv.EnvVar(), fmt.Sprintf(" \r %s \r", testTokenVal))

	got, err := retrieveAuthToken()

	s.Require().NoError(err)
	s.Equal(got, testTokenVal, "Expected auth token without whitespaces")
}
