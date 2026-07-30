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

	defaultFetchTimeout = 10 * time.Second

	// gRPC method for the KubeVirt System.CABundle RPC.
	// Proto: kubevirt.io/kubevirt/pkg/vsock/system/v1/system.proto
	caBundleMethod = "/kubevirt.vsock.system.v1.System/CABundle"
)

// fetchKubeVirtCA calls the KubeVirt System.CABundle gRPC service on
// VSOCK CID 2 (host), port 1 and returns the CA bundle PEM bytes.
//
// virt-handler serves a gRPC System service on this port (not raw PEM).
// We use a raw-bytes codec to avoid importing kubevirt.io/client-go
// (its init() panics due to glog -v flag conflicts).
//
// ctx bounds the whole call (dial + RPC); callers are responsible for
// attaching a deadline, since this service is not always reachable - see
// CARefresher for why a blocked call here must not block forever.
func fetchKubeVirtCA(ctx context.Context) ([]byte, error) {
	conn, err := grpc.NewClient(
		"passthrough:///vsock",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) {
			// ctx only bounds the gRPC Invoke after the dial completes: vsock.Dial has no
			// context-aware variant, so a hung dial (e.g. a kernel-level VSOCK bug) would
			// leak this goroutine. Not a known operational concern today since VSOCK
			// connects complete near-instantly.
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

// Name returns "proto" so gRPC negotiates the same content-type/codec the
// server already expects; this codec doesn't actually do protobuf
// marshaling itself (see Marshal/Unmarshal above), it just passes raw bytes through.
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

// CARefresher fetches and caches the KubeVirt CA bundle used to verify
// virt-handler's client certificate during the VSOCK TLS handshake.
//
// KubeVirt's System.CABundle service is not always permanently available.
// In namespace-isolated VSOCK mode (KubeVirt VEP 222) the service exists
// only for the duration of a virt-handler dial+handshake, so the fetch
// must happen inside the handshake itself. See:
// https://github.com/kubevirt/enhancements/blob/main/veps/sig-compute/222-vsock-netns-vep/vsock-netns-vep.md#change-4-on-demand-vsock-ca-service
//
// ensureFreshPool fetches the CA on every handshake and falls back to the
// cached pool when the fetch fails. This keeps the CA current across
// rotations without requiring a separate cache-invalidation or retry
// mechanism.
type CARefresher struct {
	mu   sync.RWMutex
	pool *x509.CertPool

	fetchTimeout time.Duration
	fetchCA      func(ctx context.Context) ([]byte, error)
}

// NewCARefresher creates a refresher with an empty cache. The cache
// warms lazily on the first TLS handshake via GetConfigForClient.
func NewCARefresher(opts ...CARefresherOption) *CARefresher {
	r := &CARefresher{
		fetchTimeout: defaultFetchTimeout,
		fetchCA:      fetchKubeVirtCA,
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// CARefresherOption configures the CARefresher.
type CARefresherOption func(*CARefresher)

// WithFetchTimeout bounds how long any single fetch attempt may take
// before giving up.
func WithFetchTimeout(d time.Duration) CARefresherOption {
	return func(r *CARefresher) { r.fetchTimeout = d }
}

// WithFetchFunc overrides the CA fetch function (for testing).
func WithFetchFunc(f func(ctx context.Context) ([]byte, error)) CARefresherOption {
	return func(r *CARefresher) { r.fetchCA = f }
}

// ensureFreshPool fetches a fresh CA on every call and falls back to
// the cached pool when the fetch fails. Only a cold cache with no
// prior successful fetch propagates the error.
func (r *CARefresher) ensureFreshPool(ctx context.Context) (*x509.CertPool, error) {
	pool, err := r.fetchAndCachePool(ctx)
	if err == nil {
		return pool, nil
	}
	if cached, ok := r.cachedPool(); ok {
		log.Warnf("KubeVirt CA fetch failed, reusing cached pool: %v", err)
		return cached, nil
	}
	return nil, fmt.Errorf("no CA pool cached and fetch failed: %w", err)
}

// fetchAndCachePool fetches a fresh CA bundle from KubeVirt, parses it, and
// swaps it into the cache under r.mu's write lock.
func (r *CARefresher) fetchAndCachePool(ctx context.Context) (*x509.CertPool, error) {
	// ctx belongs to the caller whose handshake triggered this fetch. If
	// that handshake is torn down mid-fetch, ctx is cancelled - but the
	// fetch already in flight can still finish and warm the cache for
	// whichever handshake asks next. Don't let this caller's own
	// cancellation abort that: detach from ctx's cancellation/deadline
	// with WithoutCancel, and bound the fetch by r.fetchTimeout alone.
	fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), r.fetchTimeout)
	defer cancel()

	ca, err := r.fetchCA(fetchCtx)
	if err != nil {
		return nil, err
	}
	newPool := x509.NewCertPool()
	if !newPool.AppendCertsFromPEM(ca) {
		return nil, errors.New("no valid certificates found in CA bundle")
	}

	concurrency.WithLock(&r.mu, func() {
		r.pool = newPool
	})
	log.Info("KubeVirt CA refreshed successfully")
	return newPool, nil
}

// cachedPool returns the cached pool and true if it is populated.
// A stale pool is always preferred over failing handshakes.
func (r *CARefresher) cachedPool() (*x509.CertPool, bool) {
	type result struct {
		pool *x509.CertPool
		ok   bool
	}
	res := concurrency.WithRLock1(&r.mu, func() result {
		return result{pool: r.pool, ok: r.pool != nil}
	})
	return res.pool, res.ok
}

// TLSConfig returns a *tls.Config that presents serverCert and validates
// KubeVirt client certs. GetConfigForClient fetches a fresh CA on every
// handshake and returns a config with the latest ClientCAs.
// Go's standard RequireAndVerifyClientCert handles verification.
//
// SessionTicketsDisabled prevents TLS 1.2 session resumption from
// skipping client certificate verification on reconnect.
func (r *CARefresher) TLSConfig(serverCert tls.Certificate) *tls.Config {
	cfg := &tls.Config{
		Certificates:           []tls.Certificate{serverCert},
		ClientAuth:             tls.RequireAndVerifyClientCert,
		MinVersion:             tls.VersionTLS12,
		SessionTicketsDisabled: true,
	}
	cfg.GetConfigForClient = func(info *tls.ClientHelloInfo) (*tls.Config, error) {
		pool, err := r.ensureFreshPool(info.Context())
		if err != nil {
			return nil, fmt.Errorf("fetching KubeVirt CA for TLS handshake: %w", err)
		}
		c := cfg.Clone()
		c.GetConfigForClient = nil
		c.ClientCAs = pool
		return c, nil
	}
	return cfg
}
