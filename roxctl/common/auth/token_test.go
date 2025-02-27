package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stretchr/testify/suite"
)

const testTokenVal = "test-token"

func TestToken(t *testing.T) {
	suite.Run(t, new(tokenSuite))
}

type tokenSuite struct {
	suite.Suite

	c *cobra.Command
}

var _ suite.SetupTestSuite = (*tokenSuite)(nil)
var _ suite.SetupSubTest = (*tokenSuite)(nil)

func (s *tokenSuite) SetupTest() {
	// Reset the APITokenFile value between tests.
	s.c = &cobra.Command{}
	flags.AddCentralConnectionFlags(s.c)
	s.NoError(s.c.ParseFlags([]string{}))
	tokenFileFlag := s.c.PersistentFlags().Lookup("token-file")
	s.NoError(tokenFileFlag.Value.Set(""))
	tokenFileFlag.Changed = false
	s.T().Setenv(env.TokenFileEnv.EnvVar(), "")
	s.T().Setenv(env.TokenEnv.EnvVar(), "")
}

func (s *tokenSuite) SetupSubTest() {
	s.SetupTest()
}

func (s *tokenSuite) Test_RetrieveAuthToken_WithEnv() {
	s.T().Setenv(env.TokenEnv.EnvVar(), testTokenVal)

	method := tokenMethod{}
	got, err := method.retrieveAuthToken()

	s.Require().NoError(err)
	s.Equal(got, testTokenVal, "Did not receive correct rox auth token from environment")
}

func (s *tokenSuite) Test_RetrieveAuthToken_WithFileEnv() {
	tstDir := s.T().TempDir()
	filePath := filepath.Join(tstDir, "token")
	err := os.WriteFile(filePath, []byte(testTokenVal), 0600)
	s.NoError(err)
	s.T().Setenv(env.TokenFileEnv.EnvVar(), filePath)

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

func (s *tokenSuite) Test_RetrieveAuthToken_Precedence() {
	const tokenForFlag = "for-flag"
	const tokenForFileEnv = "token-for-file-env"
	const tokenForEnv = "for-env"

	tstDir := s.T().TempDir()
	forFlagFilePath := filepath.Join(tstDir, tokenForFlag)
	initErr := os.WriteFile(forFlagFilePath, []byte(tokenForFlag), 0600)
	s.NoError(initErr)
	forEnvFilePath := filepath.Join(tstDir, tokenForFileEnv)
	initErr = os.WriteFile(forEnvFilePath, []byte(tokenForFileEnv), 0600)
	s.NoError(initErr)

	s.Run("error when neither flag nor env vars are not set", func() {
		method := tokenMethod{}
		token, err := method.retrieveAuthToken()
		s.ErrorIs(err, errox.InvalidArgs)
		s.Equal("", token)
	})
	testCases := map[string]struct {
		flagValue     string
		fileEnvValue  string
		tokenEnvValue string
		expectedToken string
	}{

		"flag has precedence over env vars": {
			flagValue:     forFlagFilePath,
			fileEnvValue:  forEnvFilePath,
			tokenEnvValue: tokenForEnv,
			expectedToken: tokenForFlag,
		},
		"flag has precedence over file env var": {
			flagValue:     forFlagFilePath,
			fileEnvValue:  forEnvFilePath,
			tokenEnvValue: "",
			expectedToken: tokenForFlag,
		},
		"flag has precedence over token env var": {
			flagValue:     forFlagFilePath,
			fileEnvValue:  "",
			tokenEnvValue: tokenForEnv,
			expectedToken: tokenForFlag,
		},
		"file env var has precedence over token env var when no flag": {
			flagValue:     "",
			fileEnvValue:  forEnvFilePath,
			tokenEnvValue: tokenForEnv,
			expectedToken: tokenForFileEnv,
		},
		"token env var is used when neither flag nor file env var are present": {
			flagValue:     "",
			fileEnvValue:  "",
			tokenEnvValue: tokenForEnv,
			expectedToken: tokenForEnv,
		},
	}

	for name, testCase := range testCases {
		s.Run(name, func() {
			s.T().Setenv(env.TokenFileEnv.EnvVar(), testCase.fileEnvValue)
			s.T().Setenv(env.TokenEnv.EnvVar(), testCase.tokenEnvValue)

			if testCase.flagValue != "" {
				s.NoError(s.c.ParseFlags([]string{"--token-file", testCase.flagValue}))
			}

			method := tokenMethod{}
			token, err := method.retrieveAuthToken()
			s.NoError(err)
			s.Equal(testCase.expectedToken, token)
		})
	}
	s.Run("error when neither flag nor env vars are non-empty", func() {
		s.NoError(s.c.ParseFlags([]string{"--token-file", ""}))
		s.T().Setenv(env.TokenFileEnv.EnvVar(), "")
		s.T().Setenv(env.TokenEnv.EnvVar(), "")

		method := tokenMethod{}
		token, err := method.retrieveAuthToken()
		s.ErrorIs(err, errox.InvalidArgs)
		s.Equal("", token)
	})
}
