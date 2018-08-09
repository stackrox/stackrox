package main

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/stackrox/rox/pkg/errorhelpers"
)

const (
	registryUsernameVar = "REGISTRY_USERNAME"
	registryPasswordVar = "REGISTRY_PASSWORD"

	authTemplate = `{"username": "%s", "password": "%s"}`
)

func main() {
	errorList := errorhelpers.NewErrorList("Environment")
	username, ok := os.LookupEnv(registryUsernameVar)
	if !ok {
		errorList.AddString(fmt.Sprintf("'%s' is required", registryUsernameVar))
	}
	password, ok := os.LookupEnv(registryPasswordVar)
	if !ok {
		errorList.AddString(fmt.Sprintf("'%s' is required", registryPasswordVar))
	}
	if err := errorList.ToError(); err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	authString := fmt.Sprintf(authTemplate, username, password)
	env := base64.URLEncoding.EncodeToString([]byte(authString))
	fmt.Print(env)
}
