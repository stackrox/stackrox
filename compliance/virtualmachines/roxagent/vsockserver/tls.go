package vsockserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/mdlayher/vsock"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protowire"
)

var log = logging.LoggerForModule()

const (
	// KubeVirt distributes its CA via CID 2 (host), port 1.
	kubevirtCACID  = 2
	kubevirtCAPort = 1

	defaultRefreshInterval = 1 * time.Hour
	caFetchTimeout         = 10 * time.Second

	// gRPC method for the KubeVirt System.CABundle RPC.
	// Proto: kubevirt.io/kubevirt/pkg/vsock/system/v1/system.proto
	caBundleMethod = "/kubevirt.vsock.system.v1.System/CABundle"
)

// FetchKubeVirtCA calls the KubeVirt System.CABundle gRPC service on
// VSOCK CID 2 (host), port 1 and returns the CA bundle PEM bytes.
//
// virt-handler serves a gRPC System service on this port (not raw PEM).
// We use a raw-bytes codec to avoid importing kubevirt.io/client-go
// (its init() panics due to glog -v flag conflicts).
func FetchKubeVirtCA() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), caFetchTimeout)
	defer cancel()

	conn, err := grpc.NewClient(
		"passthrough:///vsock",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) {
			return vsock.Dial(kubevirtCACID, kubevirtCAPort, nil)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("creating gRPC client for KubeVirt CA (CID %d, port %d): %w",
			kubevirtCACID, kubevirtCAPort, err)
	}
	defer func() { _ = conn.Close() }()

	// EmptyRequest marshals to zero bytes; response is Bundle { bytes Raw = 1; }.
	var resp []byte
	if err := conn.Invoke(ctx, caBundleMethod, []byte(nil), &resp,
		grpc.ForceCodec(rawBytesCodec{})); err != nil {
		return nil, fmt.Errorf("calling KubeVirt CABundle RPC: %w", err)
	}

	ca, err := extractBundleRaw(resp)
	if err != nil {
		return nil, fmt.Errorf("parsing CA bundle response: %w", err)
	}
	if len(ca) == 0 {
		return nil, errors.New("empty CA bundle from KubeVirt CA service")
	}
	return ca, nil
}

// rawBytesCodec is a gRPC codec that passes raw protobuf-encoded bytes
// without requiring generated message types.
type rawBytesCodec struct{}

func (rawBytesCodec) Marshal(v any) ([]byte, error) {
	b, ok := v.([]byte)
	if !ok {
		return nil, fmt.Errorf("rawBytesCodec: expected []byte, got %T", v)
	}
	return b, nil
}

func (rawBytesCodec) Unmarshal(data []byte, v any) error {
	b, ok := v.(*[]byte)
	if !ok {
		return fmt.Errorf("rawBytesCodec: expected *[]byte, got %T", v)
	}
	*b = append((*b)[:0], data...)
	return nil
}

func (rawBytesCodec) Name() string { return "proto" }

// extractBundleRaw decodes the KubeVirt Bundle protobuf response and
// returns the Raw field (field number 1, bytes wire type).
// Proto definition: message Bundle { bytes Raw = 1; }
func extractBundleRaw(data []byte) ([]byte, error) {
	for len(data) > 0 {
		num, wtyp, n := protowire.ConsumeTag(data)
		if n < 0 {
			return nil, errors.New("malformed KubeVirt Bundle response")
		}
		data = data[n:]

		switch wtyp {
		case protowire.BytesType:
			val, vn := protowire.ConsumeBytes(data)
			if vn < 0 {
				return nil, errors.New("malformed protobuf bytes field")
			}
			if num == 1 {
				return val, nil
			}
			data = data[vn:]
		case protowire.VarintType:
			_, vn := protowire.ConsumeVarint(data)
			if vn < 0 {
				return nil, errors.New("malformed protobuf varint field")
			}
			data = data[vn:]
		case protowire.Fixed32Type:
			_, vn := protowire.ConsumeFixed32(data)
			if vn < 0 {
				return nil, errors.New("malformed protobuf fixed32 field")
			}
			data = data[vn:]
		case protowire.Fixed64Type:
			_, vn := protowire.ConsumeFixed64(data)
			if vn < 0 {
				return nil, errors.New("malformed protobuf fixed64 field")
			}
			data = data[vn:]
		default:
			return nil, fmt.Errorf("unsupported protobuf wire type %d", wtyp)
		}
	}
	return nil, errors.New("Bundle.Raw field not found in KubeVirt CA response")
}

// CARefresher periodically fetches the KubeVirt CA and builds a TLS config.
type CARefresher struct {
	mu       sync.RWMutex
	pool     *x509.CertPool
	interval time.Duration
	fetchCA  func() ([]byte, error)
}

// NewCARefresher creates a refresher. Call Start() to begin periodic fetching.
func NewCARefresher(opts ...CARefresherOption) *CARefresher {
	r := &CARefresher{
		interval: defaultRefreshInterval,
		fetchCA:  FetchKubeVirtCA,
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// CARefresherOption configures the CARefresher.
type CARefresherOption func(*CARefresher)

// WithRefreshInterval sets the CA refresh interval.
func WithRefreshInterval(d time.Duration) CARefresherOption {
	return func(r *CARefresher) { r.interval = d }
}

// WithFetchFunc overrides the CA fetch function (for testing).
func WithFetchFunc(f func() ([]byte, error)) CARefresherOption {
	return func(r *CARefresher) { r.fetchCA = f }
}

// TryInitialFetch attempts one CA fetch. Returns nil on success.
func (r *CARefresher) TryInitialFetch() error {
	if err := r.refresh(); err != nil {
		return fmt.Errorf("initial CA fetch: %w", err)
	}
	return nil
}

// RunRefreshLoop periodically refreshes the CA. Blocks until ctx is cancelled.
// Call TryInitialFetch first.
func (r *CARefresher) RunRefreshLoop(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.refresh(); err != nil {
				log.Errorf("Failed to refresh KubeVirt CA: %v", err)
			}
		}
	}
}

// Start fetches the CA immediately, then refreshes periodically.
// Blocks until ctx is cancelled.
func (r *CARefresher) Start(ctx context.Context) error {
	if err := r.TryInitialFetch(); err != nil {
		return err
	}
	r.RunRefreshLoop(ctx)
	return nil
}

func (r *CARefresher) refresh() error {
	ca, err := r.fetchCA()
	if err != nil {
		return err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(ca) {
		return errors.New("no valid certificates found in CA bundle")
	}
	concurrency.WithLock(&r.mu, func() {
		r.pool = pool
	})
	log.Info("KubeVirt CA refreshed successfully")
	return nil
}

// TLSConfig returns a *tls.Config that validates KubeVirt client certs.
// The returned config dynamically reads the latest CA pool on each handshake
// via GetConfigForClient.
func (r *CARefresher) TLSConfig() *tls.Config {
	cfg := &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		MinVersion: tls.VersionTLS12,
	}
	// Capture cfg so the callback can forward Certificates set by the caller.
	cfg.GetConfigForClient = func(*tls.ClientHelloInfo) (*tls.Config, error) {
		pool := concurrency.WithRLock1(&r.mu, func() *x509.CertPool {
			return r.pool
		})
		if pool == nil {
			return nil, errors.New("CA pool not initialized")
		}
		return &tls.Config{
			Certificates: cfg.Certificates,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    pool,
			MinVersion:   tls.VersionTLS12,
		}, nil
	}
	return cfg
}
