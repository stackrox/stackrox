package auth

import (
	"fmt"
	"os"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/client/authn/basic"
	"github.com/stackrox/rox/roxctl/common/flags"
	"google.golang.org/grpc/credentials"
)

type basicMethod struct{}

var (
	_ Method = (*basicMethod)(nil)
)

// BasicAuth provides an auth.Method for basic authentication.
// It will use the inputs from the --password flag or the ROX_ADMIN_PASSWORD environment variable.
func BasicAuth() Method {
	return &basicMethod{}
}

func (b basicMethod) Type() string {
	return "basic auth"
}

func (b basicMethod) GetCredentials(_ string) (credentials.PerRPCCredentials, error) {
	password := flags.Password()
	if password == "" {
		return nil, errox.InvalidArgs.New("no password specified either via flag or environment variable")
	}
	username := os.Getenv("ROX_USERNAME")
	if username == "" {
		username = basic.DefaultUsername
	}
	fmt.Printf("Connecting as user %s\n", username)

	return basic.PerRPCCredentials(username, password), nil
}
