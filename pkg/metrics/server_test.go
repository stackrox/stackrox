package metrics

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/mtls/verifier/mocks"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var (
	fakeClientCAFile   = "./testdata/ca.pem"
	fakeClientCertFile = "./testdata/client.crt"
	fakeClientKeyFile  = "./testdata/client.key"
	fakeCertFile       = "./testdata/tls.crt"
	fakeKeyFile        = "./testdata/tls.key"
)

func TestMetricsServerAddressEnvs(t *testing.T) {
	cases := map[string]struct {
		metricsPort         string
		enableSecureMetrics string
		secureMetricsPort   string
	}{
		"default": {
			metricsPort:         "",
			enableSecureMetrics: "false",
			secureMetricsPort:   "",
		},
		"only metricsPort set": {
			metricsPort:         ":8008",
			enableSecureMetrics: "false",
			secureMetricsPort:   "",
		},
		"only secureMetricsPort set": {
			metricsPort:         "",
			enableSecureMetrics: "true",
			secureMetricsPort:   ":8009",
		},
		"metrisPort and secureMetricsPort set": {
			metricsPort:         ":8008",
			enableSecureMetrics: "true",
			secureMetricsPort:   ":8009",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv(env.MetricsPort.EnvVar(), c.metricsPort)
			t.Setenv(env.EnableSecureMetrics.EnvVar(), c.enableSecureMetrics)
			t.Setenv(env.SecureMetricsPort.EnvVar(), c.secureMetricsPort)

			server := NewServer(CentralSubsystem, &nilTLSConfigurer{})

			require.NotNil(t, server)
			assert.Equal(t, env.MetricsPort.Setting(), server.metricsServer.Addr)
			if c.enableSecureMetrics == "true" {
				require.NotNil(t, server.secureMetricsServer)
				assert.Equal(t, env.SecureMetricsPort.Setting(), server.secureMetricsServer.Addr)
			} else {
				assert.Nil(t, server.secureMetricsServer)
			}
		})
	}
}

func TestMetricsServerPanic(t *testing.T) {
	cases := map[string]struct {
		metricsPort         string
		enableSecureMetrics string
		secureMetricsPort   string
		releaseBuild        bool
	}{
		"metrics error - debug build panics": {
			metricsPort:         "error",
			enableSecureMetrics: "false",
			secureMetricsPort:   "",
			releaseBuild:        false,
		},
		"metrics error - release build does not panic": {
			metricsPort:         "error",
			enableSecureMetrics: "false",
			secureMetricsPort:   "",
			releaseBuild:        true,
		},
		"secureMetrics error - debug build panics": {
			metricsPort:         "disabled",
			enableSecureMetrics: "true",
			secureMetricsPort:   "error",
			releaseBuild:        false,
		},
		"secureMetrics error - release build does not panic": {
			metricsPort:         "disabled",
			enableSecureMetrics: "true",
			secureMetricsPort:   "error",
			releaseBuild:        true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if buildinfo.ReleaseBuild != c.releaseBuild {
				t.SkipNow()
			}
			t.Setenv(env.MetricsPort.EnvVar(), c.metricsPort)
			t.Setenv(env.EnableSecureMetrics.EnvVar(), c.enableSecureMetrics)
			t.Setenv(env.SecureMetricsPort.EnvVar(), c.secureMetricsPort)
			server := NewServer(CentralSubsystem, &nilTLSConfigurer{})
			defer server.Stop(context.TODO())

			if c.releaseBuild {
				assert.NotPanics(t, func() { server.RunForever() })
			} else {
				assert.Panics(t, func() { server.RunForever() })
			}
		})
	}
}

// getFreePort asks the kernel for a free open port that is ready to use.
func getFreePort() (port int, err error) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	defer utils.IgnoreError(listener.Close)
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func TestMetricsServerHTTPRequest(t *testing.T) {
	freePort, err := getFreePort()
	require.NoError(t, err)
	t.Setenv(env.MetricsPort.EnvVar(), fmt.Sprintf(":%d", freePort))
	t.Setenv(env.EnableSecureMetrics.EnvVar(), "false")
	server := NewServer(CentralSubsystem, &nilTLSConfigurer{})
	defer server.Stop(context.TODO())
	server.RunForever()

	url := fmt.Sprintf("http://localhost:%d/metrics", freePort)
	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		resp, err := http.Get(url)
		require.NoError(t, err)
		defer utils.IgnoreError(resp.Body.Close)
		msg, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(msg), "go_gc_duration_seconds")
	}, 1*time.Second, 50*time.Millisecond)
}

func fakeTLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(fakeCertFile, fakeKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "loading test certificate failed")
	}

	certPool := x509.NewCertPool()
	pem, err := os.ReadFile(fakeClientCAFile)
	if err != nil {
		return nil, errors.Wrap(err, "loading test client CA certificate")
	}
	if !certPool.AppendCertsFromPEM(pem) {
		return nil, errors.Wrap(err, "failed to add client certificate")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	}
	return tlsConfig, nil
}

func testClient() (*http.Client, error) {
	cert, err := tls.LoadX509KeyPair(fakeClientCertFile, fakeClientKeyFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load client certificate")
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			// We are using a self-signed certificate for testing.
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Transport: tr}
	return client, nil
}

func TestSecureMetricsServerHTTPRequest(t *testing.T) {
	t.Setenv(env.MetricsPort.EnvVar(), "disabled")
	t.Setenv(env.EnableSecureMetrics.EnvVar(), "true")
	freePort, err := getFreePort()
	require.NoError(t, err)
	t.Setenv(env.SecureMetricsPort.EnvVar(), fmt.Sprintf(":%d", freePort))
	t.Setenv(env.SecureMetricsCertDir.EnvVar(), "./testdata")
	ctrl := gomock.NewController(t)
	fakeTLSConfigurer := mocks.NewMockTLSConfigurer(ctrl)
	fakeTLSConfigurer.EXPECT().TLSConfig().Return(fakeTLSConfig())

	server := NewServer(CentralSubsystem, fakeTLSConfigurer)
	defer server.Stop(context.TODO())
	server.RunForever()

	client, err := testClient()
	require.NoError(t, err)
	url := fmt.Sprintf("https://localhost:%d/metrics", freePort)
	assert.EventuallyWithT(t, func(collect *assert.CollectT) {
		resp, err := client.Get(url)
		require.NoError(t, err)
		defer utils.IgnoreError(resp.Body.Close)
		msg, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(msg), "go_gc_duration_seconds")
	}, 1*time.Second, 50*time.Millisecond)
}
