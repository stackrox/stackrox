package manager

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/grpc/util"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/scanner/pkg/clairify/client"
)

var (
	log = logging.LoggerForModule()
)

type Manager struct {
	httpClient   *client.Clairify
	gRPCEndpoint string
	tlsConfig    *tls.Config

	conn *util.LazyClientConn
}

func NewManager(httpEndpoint, gRPCEndpoint string) (*Manager, error) {
	if httpEndpoint == "" && gRPCEndpoint == "" {
		return nil, errors.New("no Scanner endpoints configured. Require both HTTP and gRPC")
	}

	if httpEndpoint == "" {
		return nil, errors.New("no Scanner HTTP endpoint configured")
	}
	parts := strings.SplitN(httpEndpoint, "://", 2)
	switch parts[0] {
	case "https":
		break
	default:
		if len(parts) == 1 {
			httpEndpoint = fmt.Sprintf("https://%s", httpEndpoint)
			break
		}
		return nil, errors.Errorf("Scanner HTTP endpoint has unsupported scheme: %s", parts[0])
	}

	if gRPCEndpoint == "" {
		return nil, errors.New("no Scanner gRPC endpoint configured")
	}
	gRPCEndpoint = strings.TrimPrefix(gRPCEndpoint, "https://")
	if strings.Contains(gRPCEndpoint, "://") {
		return nil, errors.Errorf("Scanner endpoint has unsupported scheme: %s", gRPCEndpoint)
	}

	tlsConfig, err := clientconn.TLSConfig(mtls.ScannerSubject, clientconn.TLSConfigOptions{
		UseClientCert: clientconn.MustUseClientCert,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize Scanner TLS config")
	}



	return &Manager{
		httpClient: client.NewWithClient(httpEndpoint, &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
				Proxy:           proxy.FromConfig(),
			},
		}),
		gRPCEndpoint: gRPCEndpoint,
		tlsConfig:    tlsConfig,

		conn: util.NewLazyClientConn(),
	}, nil
}

func (m *Manager) Start() {
	go m.start()
}

func (m *Manager) start() {
	m.waitUntilScannerIsReady()
}

func (m *Manager) waitUntilScannerIsReady() {
	exponential := backoff.NewExponentialBackOff()
	exponential.MaxElapsedTime = 5 * time.Minute
	exponential.MaxInterval = 32 * time.Second
	err := backoff.RetryNotify(func() error {
		// By default, the Ping timeout is 5 seconds.
		// It can be reconfigured via:
		// client.PingTimeout = <some duration>
		return m.httpClient.Ping()
	}, exponential, func(err error, d time.Duration) {
		log.Infof("Check Central status failed: %s. Retrying after %s...", err, d.Round(time.Millisecond))
	})
	if err != nil {
		s.stoppedSig.SignalWithErrorWrapf(err, "checking central status failed after %s", exponential.GetElapsedTime())
	}
}
