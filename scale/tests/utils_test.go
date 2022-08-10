package tests

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/grpc/client/authn/basic"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/roxctl/common"
	"github.com/stackrox/rox/pkg/sync"
	"google.golang.org/grpc"
)

var (
	once    sync.Once
	envVars *testEnvVars

	log = logging.LoggerForModule()
)

type testEnvVars struct {
	endpoint string
	password string
}

func getConnection(endpoint, password string) (*grpc.ClientConn, error) {
	serverName, _, _, err := netutil.ParseEndpoint(endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "parsing central endpoint")
	}
	opts := clientconn.Options{
		TLS: clientconn.TLSConfigOptions{
			ServerName:         serverName,
			InsecureSkipVerify: true,
		},
	}
	opts.ConfigureBasicAuth(basic.DefaultUsername, password)
	return clientconn.GRPCConnection(common.Context(), mtls.CentralSubject, endpoint, opts)
}

func asyncWithWaitGroup(function func() error, wg *concurrency.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Add(-1)
		err := function()
		if err != nil {
			log.Fatal(err)
		}
	}()
}

func getHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
}

func getEnvVars() *testEnvVars {
	once.Do(func() {
		envVars = &testEnvVars{}
		envVars.password = os.Getenv("ROX_PASSWORD")
		envVars.endpoint = fmt.Sprintf("%s:%s", os.Getenv("API_HOSTNAME"), os.Getenv("API_PORT"))
	})
	return envVars
}
