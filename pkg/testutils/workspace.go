package testutils

import (
	"os"
	"path"
	"reflect"
	"strings"
)

type empty struct{}

// GetTestWorkspaceDir returns the root directory of our workspace we're building, which may or may not be under GOPATH
func GetTestWorkspaceDir() string {
	utilsPath := reflect.TypeOf(empty{}).PkgPath()
	workspaceName := strings.TrimSuffix(utilsPath, "/pkg/utils")

	// GOPATH env sometimes return /home/user/go:/home/user/go when running in IDE
	return path.Join(strings.Split(os.Getenv("GOPATH"), ":")[0], "src", workspaceName)
}
