package flags

import (
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
)

var (
	password        string
	passwordChanged *bool

	apiTokenFile        string
	apiTokenFileChanged *bool

	authFlagSet = func() *pflag.FlagSet {
		fs := pflag.NewFlagSet("auth", pflag.ExitOnError)
		fs.StringVarP(&password, "password", "p", "",
			"Password for basic auth. Alternatively, set the password via the ROX_ADMIN_PASSWORD environment variable")
		passwordChanged = &fs.Lookup("password").Changed

		fs.StringVarP(&apiTokenFile,
			"token-file",
			"",
			"",
			"Use the API token in the provided file to authenticate. "+
				"Alternatively, set the path via the ROX_API_TOKEN_FILE environment variable or "+
				"set the token via the ROX_API_TOKEN environment variable")
		apiTokenFileChanged = &fs.Lookup("token-file").Changed

		return fs
	}()
)

// Password returns the set password.
func Password() string {
	return flagOrSettingValue(password, *passwordChanged, env.PasswordEnv)
}

// PasswordChanged returns whether the password is provided as an argument.
func PasswordChanged() bool {
	return *passwordChanged
}

// APITokenFile returns the currently specified API token file name.
func APITokenFile() string {
	return flagOrSettingValue(apiTokenFile, APITokenFileChanged(), env.TokenFileEnv)
}

// APITokenFileChanged returns whether the token-file is provided as an argument.
func APITokenFileChanged() bool {
	return apiTokenFileChanged != nil && *apiTokenFileChanged
}

// ReadTokenFromFile attempts to retrieve a token from the currently specified file.
func ReadTokenFromFile(fileName string) (string, error) {
	content, err := os.ReadFile(fileName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to retrieve token from file %q", fileName)
	}
	token := strings.TrimSpace(string(content))
	if token != "" {
		return token, nil
	}
	return "", errox.NotFound.Newf("failed to retrieve token from file %q: file is empty", fileName)
}

func addCentralAuthFlags(c *cobra.Command) {
	c.PersistentFlags().AddFlagSet(authFlagSet)
}
