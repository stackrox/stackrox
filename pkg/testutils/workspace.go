package testutils

import (
	"os"
	"path"
	"reflect"
	"strings"
)

type empty struct{}

// GetBazelWorkspaceDir returns the root of our workspace if we're running under bazel or "" if not.
func GetBazelWorkspaceDir() string {
	bzlSrcdir := os.Getenv("TEST_SRCDIR")
	bzlTestWs := os.Getenv("TEST_WORKSPACE")
	if bzlSrcdir != "" && bzlTestWs != "" {
		// we're in a bazel test
		return bzlSrcdir + "/" + bzlTestWs
	}
	return ""
}

// GetTestWorkspaceDir returns the root directory of our workspace we're building, which may or may not be under GOPATH
func GetTestWorkspaceDir() string {
	bzl := GetBazelWorkspaceDir()
	if bzl != "" {
		return bzl
	}

	utilsPath := reflect.TypeOf(empty{}).PkgPath()
	workspaceName := strings.TrimSuffix(utilsPath, "/pkg/utils")

	// GOPATH env sometimes return /home/user/go:/home/user/go when running in IDE
	return path.Join(strings.Split(os.Getenv("GOPATH"), ":")[0], "src", workspaceName)
}
