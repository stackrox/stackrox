package tests

import (
	"log"
	"os"
	"os/exec"
	"testing"

	"bitbucket.org/stack-rox/apollo/tests/platform"
)

var (
	localEndpoint = `localhost:8000`
)

func TestMain(m *testing.M) {
	setup()

	os.Exit(m.Run())
}

// only kubernetes is currently supported.
func setup() {
	p := platform.NewFromEnv()

	cmd := exec.Command(`bash`, `-c`, p.SetupScript())
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}

func init() {
	if le, ok := os.LookupEnv(`LOCAL_API_ENDPOINT`); ok {
		localEndpoint = le
	}
}
