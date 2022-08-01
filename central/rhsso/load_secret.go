package rhsso

import (
	"io"
	"os"
)

func LoadRhSsoSecret() string {
	f, err := os.Open(`/run/secrets/stackrox.io/rhsso/clientSecret`)
	if err != nil {
		panic(err)
	}
	content, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}
	return string(content)
}
