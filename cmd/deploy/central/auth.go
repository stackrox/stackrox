package central

import (
	"bytes"
	"crypto/rand"
	"math/big"

	"github.com/stackrox/rox/pkg/auth/authproviders/userpass/htpasswd"
)

const (
	adminUsername = "admin"

	autogenPasswordLength = 25

	pwCharacters = `abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789`
)

func generateHtpasswd(c *Config) ([]byte, error) {
	if c.Password == "" {
		c.Password = createPassword()
	}

	hf := htpasswd.New()
	hf.Set(adminUsername, c.Password)
	buf := new(bytes.Buffer)
	err := hf.Write(buf)
	return buf.Bytes(), err
}

func createPassword() string {
	var pw string
	max := big.NewInt(int64(len(pwCharacters)))
	for i := 0; i < autogenPasswordLength; i++ {
		randInt, err := rand.Int(rand.Reader, max)
		if err != nil {
			panic(err)
		}
		pw += string(pwCharacters[randInt.Int64()])
	}
	return pw
}
