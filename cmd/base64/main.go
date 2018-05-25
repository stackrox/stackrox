package main

import (
	"encoding/base64"
	"fmt"
	"os"

	"bitbucket.org/stack-rox/apollo/pkg/errorhelpers"
)

const (
	registryUsernameVar = "REGISTRY_USERNAME"
	registryPasswordVar = "REGISTRY_PASSWORD"

	authTemplate = `{"username": "%s", "password": "%s"}`
)

func main() {
	var errors []string
	username, ok := os.LookupEnv(registryUsernameVar)
	if !ok {
		errors = append(errors, fmt.Sprintf("'%s' is required", registryUsernameVar))
	}
	password, ok := os.LookupEnv(registryPasswordVar)
	if !ok {
		errors = append(errors, fmt.Sprintf("'%s' is required", registryPasswordVar))
	}
	if len(errors) != 0 {
		fmt.Println(errorhelpers.FormatErrorStrings("Environment", errors))
		os.Exit(1)
	}
	authString := fmt.Sprintf(authTemplate, username, password)
	env := base64.URLEncoding.EncodeToString([]byte(authString))
	fmt.Print(env)
}
