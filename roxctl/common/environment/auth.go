package environment

import (
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/client/authn/basic"
	"github.com/stackrox/rox/pkg/grpc/client/authn/tokenbased"
	"github.com/stackrox/rox/roxctl/common/auth"
	"github.com/stackrox/rox/roxctl/common/flags"
	"google.golang.org/grpc/credentials"
)

type passwordFlagAuthSource struct{}

func (passwordFlagAuthSource) Name() string {
	return "--password/-p flag"
}

func (passwordFlagAuthSource) GetCreds(_ string) (credentials.PerRPCCredentials, error) {
	passwd := flags.Password()
	if passwd == "" {
		return nil, errox.InvalidArgs.New("no or empty password specified")
	}
	return basic.PerRPCCredentials(basic.DefaultUsername, passwd), nil
}

type tokenFileFlagAuthSource struct{}

func (tokenFileFlagAuthSource) Name() string {
	return "--token-file flag"
}

func (tokenFileFlagAuthSource) GetCreds(_ string) (credentials.PerRPCCredentials, error) {
	token, err := flags.ReadTokenFromFile(flags.APITokenFile())
	if err != nil {
		return nil, errors.Wrapf(err, "could not read token from %q", flags.APITokenFile())
	}
	return tokenbased.PerRPCCredentials(token), nil
}

type tokenEnvVarAuthSource struct{}

func (tokenEnvVarAuthSource) Name() string {
	return "ROX_API_TOKEN environment variable"
}

func (tokenEnvVarAuthSource) GetCreds(_ string) (credentials.PerRPCCredentials, error) {
	token := os.Getenv("ROX_API_TOKEN")
	return tokenbased.PerRPCCredentials(token), nil
}

// determineAuthMethod does just that
func determineAuthMethod(env Environment) (auth.Method, error) {
	if flags.APITokenFile() != "" && flags.Password() != "" {
		return nil, errox.InvalidArgs.New("cannot use password- and token-based authentication at the same time")
	}
	if flags.Password() != "" {
		return passwordFlagAuthSource{}, nil
	}
	if flags.APITokenFile() != "" {
		return tokenFileFlagAuthSource{}, nil
	}
	if os.Getenv("ROX_API_TOKEN") != "" {
		return tokenEnvVarAuthSource{}, nil
	}
	return authFromConfig{env: env}, nil
}
