package tests

import (
	"os"
	"testing"
)

var (
	apiEndpoint = `localhost:8000`
)

func TestMain(m *testing.M) {
	setup()

	os.Exit(m.Run())
}

func setup() {
	// TODO: good practice would be to wait for the server to be responsive / warmed up (e.g. with timeout 10 sec)
}

func init() {
	if le, ok := os.LookupEnv(`API_ENDPOINT`); ok {
		apiEndpoint = le
	}
}
