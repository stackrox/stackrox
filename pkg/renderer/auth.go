package renderer

import (
	"bytes"

	"github.com/stackrox/rox/pkg/auth/htpasswd"
	"github.com/stackrox/rox/pkg/grpc/client/authn/basic"
	"github.com/stackrox/rox/pkg/random"
)

const (
	adminUsername = basic.DefaultUsername

	autogenPasswordLength = 25
)

// GenerateHtpasswd creates a password for admin user if it was not created during the install
func GenerateHtpasswd(c *Config) ([]byte, error) {
	if c.Password == "" {
		c.Password = CreatePassword()
		c.PasswordAuto = true
	}
	return CreateHtpasswd(c.Password)
}

// CreateHtpasswd creates the contents for the htpasswd secret.
func CreateHtpasswd(password string) ([]byte, error) {
	hf := htpasswd.New()
	if err := hf.Set(adminUsername, password); err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	err := hf.Write(buf)
	return buf.Bytes(), err
}

// CreatePassword generates an alphanumeric password
func CreatePassword() string {
	password, err := random.GenerateString(autogenPasswordLength, random.AlphanumericCharacters)
	if err != nil {
		panic(err)
	}
	return password
}
