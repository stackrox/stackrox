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
	"github.com/stackrox/rox/pkg/coalescer"
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

	// defaultRefreshInterval is the staleness threshold that forces a
	// handshake-triggered refetch (see CARefresher doc comment).
	defaultRefreshInterval = 1 * time.Hour
	defaultFetchTimeout    = 10 * time.Second

	// gRPC method for the KubeVirt System.CABundle RPC.
	// Proto: kubevirt.io/kubevirt/pkg/vsock/system/v1/system.proto
	caBundleMethod = "/kubevirt.vsock.system.v1.System/CABundle"

	// coalesceKey is the single key coalesced fetches are keyed on: there is
	// only one CA bundle to fetch per CARefresher instance.
	coalesceKey = "kubevirt-ca"
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
// In VSOCK "global" mode it is: a single instance runs on CID 2/port 1 for
// the lifetime of the node. In namespace-isolated ("local") mode -
// introduced by KubeVirt VEP 222, see
// https://github.com/kubevirt/enhancements/blob/main/veps/sig-compute/222-vsock-netns-vep/vsock-netns-vep.md#change-4-on-demand-vsock-ca-service
// and merged in https://github.com/kubevirt/kubevirt/pull/18067 - the
// service exists only for the duration of a single virt-handler
// dial+handshake (see pkg/virt-handler/vsock/{dial,servers,refcount}.go in
// kubevirt/kubevirt) and is torn down immediately after. A fetch on any
// independent schedule (a timer, or "once at startup") has no relationship
// to that window and will, in practice, almost always miss it. The VEP
// itself describes the intended fix: "This happens as part of the TLS
// negotiation - no explicit synchronization between virt-handler and the
// guest is needed", i.e. the fetch belongs inside the handshake, not
// alongside it.
//
// CARefresher's cache is therefore populated reactively, and only
// reactively: TLSConfig's GetConfigForClient callback calls
// ensureFreshPool, which fetches inline, synchronously, as part of the
// handshake itself - guaranteeing the fetch happens exactly when a window
// (if one is needed at all) is open, because the incoming connection *is*
// what opens it. This works unchanged in "global" mode too, since the
// service is always reachable there.
//
// Concurrent callers (e.g. multiple simultaneous handshakes) are coalesced
// into a single underlying fetch via coalescer, so a stampede never results
// in redundant dials to virt-handler.
type CARefresher struct {
	mu        sync.RWMutex
	pool      *x509.CertPool
	fetchedAt time.Time

	interval     time.Duration
	fetchTimeout time.Duration
	fetchCA      func(ctx context.Context) ([]byte, error)
	coalesce     *coalescer.Coalescer[*x509.CertPool]
}

// NewCARefresher creates a refresher with an empty cache. There is no
// separate warm-up step: the cache warms itself lazily on the first call
// to TLSConfig's GetConfigForClient. An independent fetch attempt ahead of
// a real handshake would have no relationship to KubeVirt's on-demand CA
// window in namespace-isolated VSOCK mode anyway, so it would just be
// wasted effort - see the CARefresher doc comment.
func NewCARefresher(opts ...CARefresherOption) *CARefresher {
	r := &CARefresher{
		interval:     defaultRefreshInterval,
		fetchTimeout: defaultFetchTimeout,
		fetchCA:      fetchKubeVirtCA,
		coalesce:     coalescer.New[*x509.CertPool](),
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

// ensureFreshPool returns the cached CA pool if it is populated and not
// older than r.interval, fetching a new one otherwise. Concurrent callers
// are coalesced into a single underlying fetch.
//
// A failed (re)fetch is not fatal as long as some pool - however stale - is
// already cached: certificates signed by an old CA remain valid until they
// individually expire, so serving the last known-good pool is strictly more
// available than failing every handshake until the next successful fetch.
// Only a cache that has never been populated even once - the state at boot,
// before any fetch has ever succeeded - propagates the error, since there is
// then truly nothing to validate a handshake against.
func (r *CARefresher) ensureFreshPool(ctx context.Context) (*x509.CertPool, error) {
	if pool, ok := r.freshCachedPool(); ok {
		return pool, nil
	}

	pool, err := r.coalesce.Coalesce(ctx, coalesceKey, func() (*x509.CertPool, error) {
		// Re-check: another caller may have refreshed the cache while we
		// were waiting to enter the coalesced section.
		if pool, ok := r.freshCachedPool(); ok {
			return pool, nil
		}

		// ctx here belongs to whichever single caller happened to trigger
		// this fetch, but the result is shared with every other concurrent
		// caller waiting on it too. Don't let that one caller's own
		// cancellation (e.g. its handshake being torn down) abort the fetch
		// for everyone else: detach from ctx's cancellation/deadline with
		// WithoutCancel, and bound the fetch by r.fetchTimeout alone.
		fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), r.fetchTimeout)
		defer cancel()

		ca, fetchErr := r.fetchCA(fetchCtx)
		if fetchErr != nil {
			return nil, fetchErr
		}
		newPool := x509.NewCertPool()
		if !newPool.AppendCertsFromPEM(ca) {
			return nil, errors.New("no valid certificates found in CA bundle")
		}

		concurrency.WithLock(&r.mu, func() {
			r.pool = newPool
			r.fetchedAt = time.Now()
		})
		log.Info("KubeVirt CA refreshed successfully")
		return newPool, nil
	})
	if err == nil {
		return pool, nil
	}
	if stale, fetchedAt, ok := r.cachedPool(); ok {
		log.Warnf("KubeVirt CA refresh failed, reusing last known-good CA (age: %v): %v",
			time.Since(fetchedAt), err)
		return stale, nil
	}
	return nil, err
}

// freshCachedPool returns the cached pool and true if it is populated and
// not older than r.interval.
func (r *CARefresher) freshCachedPool() (*x509.CertPool, bool) {
	return concurrency.WithRLock2(&r.mu, func() (*x509.CertPool, bool) {
		if r.pool == nil || time.Since(r.fetchedAt) >= r.interval {
			return nil, false
		}
		return r.pool, true
	})
}

// cachedPool returns the cached pool, the time it was fetched, and true if
// it is populated, regardless of staleness. There is currently no maximum
// age past which a cached pool stops being reused here - a stale pool is
// always preferred over failing handshakes closed - so fetchedAt exists
// purely for observability (see ensureFreshPool's fallback log line) rather
// than to bound anything.
func (r *CARefresher) cachedPool() (*x509.CertPool, time.Time, bool) {
	type result struct {
		pool      *x509.CertPool
		fetchedAt time.Time
		ok        bool
	}
	res := concurrency.WithRLock1(&r.mu, func() result {
		return result{pool: r.pool, fetchedAt: r.fetchedAt, ok: r.pool != nil}
	})
	return res.pool, res.fetchedAt, res.ok
}

// TLSConfig returns a *tls.Config that presents serverCert and validates
// KubeVirt client certs. The returned config fetches a fresh CA pool on
// each handshake via GetConfigForClient whenever the cache is empty or
// stale - see the CARefresher doc comment for why this, not any
// independent schedule, is what makes CA verification actually work in
// namespace-isolated VSOCK mode.
//
// serverCert is baked in at construction time rather than left for the
// caller to set on the returned config afterward, so there is no window in
// which a config with no certificate could be handed to a listener.
func (r *CARefresher) TLSConfig(serverCert tls.Certificate) *tls.Config {
	return &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS12,
		GetConfigForClient: func(info *tls.ClientHelloInfo) (*tls.Config, error) {
			pool, err := r.ensureFreshPool(info.Context())
			if err != nil {
				return nil, fmt.Errorf("fetching KubeVirt CA for TLS handshake: %w", err)
			}
			return &tls.Config{
				Certificates: []tls.Certificate{serverCert},
				ClientAuth:   tls.RequireAndVerifyClientCert,
				ClientCAs:    pool,
				MinVersion:   tls.VersionTLS12,
			}, nil
		},
	}
}
