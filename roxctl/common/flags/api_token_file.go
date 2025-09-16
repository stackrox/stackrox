package flags

import (
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
)

var (
	apiTokenFile        string
	apiTokenFileChanged *bool
)

// AddAPITokenFile adds the token-file flag to the base command.
func AddAPITokenFile(c *cobra.Command) {
	c.PersistentFlags().StringVarP(&apiTokenFile,
		"token-file",
		"",
		"",
		"Use the API token in the provided file to authenticate. "+
			"Alternatively, set the path via the ROX_API_TOKEN_FILE environment variable or "+
			"set the token via the ROX_API_TOKEN environment variable.")
	apiTokenFileChanged = &c.PersistentFlags().Lookup("token-file").Changed
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
