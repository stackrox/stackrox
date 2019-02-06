package tests

import (
	"os"
	"testing"

	"github.com/stackrox/rox/pkg/clientconn"
	"google.golang.org/grpc"
)

const (
	usernameEnvVar = `ROX_USERNAME`
	passwordEnvVar = `ROX_PASSWORD`

	ipEnvVar = "API_ENDPOINT"
)

var (
	apiEndpoint = os.Getenv(ipEnvVar)

	username, password string
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

	username = os.Getenv(usernameEnvVar)
	password = os.Getenv(passwordEnvVar)
}

func grpcConnection() (*grpc.ClientConn, error) {
	if username != "" && password != "" {
		return clientconn.GRPCConnectionWithBasicAuth(apiEndpoint, username, password)
	}
	return clientconn.UnauthenticatedGRPCConnection(apiEndpoint)
}
