package flags

import (
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
)

var (
	apiTokenFile        string
	apiTokenFileChanged *bool
)

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
