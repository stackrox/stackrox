package cmd

import (
	"context"
	"errors"
	"net"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/vsockserver"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	pb "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/vsockframing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

// fakeProvider is a vsockserver.MappingProvider test double for rescanner
// tests; UpdatePath/Bytes are unused by rescanner and always zero.
type fakeProvider struct {
	ready   bool
	path    string
	pathErr error
	hash    string
}

func (f *fakeProvider) Ready() bool  { return f.ready }
func (f *fakeProvider) Hash() string { return f.hash }
func (f *fakeProvider) UpdatePath() pb.RepoCPEMappingUpdatePath {
	return pb.RepoCPEMappingUpdatePath_REPO_CPE_MAPPING_UPDATE_PATH_UNSPECIFIED
}
func (f *fakeProvider) Bytes() ([]byte, error) { return nil, errors.New("not implemented") }
func (f *fakeProvider) Path() (string, error)  { return f.path, f.pathErr }

var _ vsockserver.MappingProvider = (*fakeProvider)(nil)

// fakeGatedProvider adds ScanBusyGate call tracking on top of fakeProvider,
// mirroring SensorUpdater (gated) as opposed to URLUpdater (ungated) so
// tests can verify scanOnce only invokes the gate for providers that
// implement it.
type fakeGatedProvider struct {
	fakeProvider
	busyCalls int
	idleCalls int
}

func (f *fakeGatedProvider) MarkScanBusy()                { f.busyCalls++ }
func (f *fakeGatedProvider) MarkScanIdleAndApplyPending() { f.idleCalls++ }

var (
	_ vsockserver.MappingProvider = (*fakeGatedProvider)(nil)
	_ vsockserver.ScanBusyGate    = (*fakeGatedProvider)(nil)
)

// testRescanner returns a rescanner with a long default interval so tests
// that don't care about the periodic loop never trigger it by accident.
// factsFn is stubbed out so Run never exercises the real discoverFacts,
// which - unlike scanFn - has no test-friendly no-op input; hostPath=""
// resolves to real absolute host paths (e.g. "/etc/pki/entitlement"), not
// a safe default. provider defaults to always-ready when nil, so existing
// tests that don't care about readiness are unaffected.
func testRescanner(provider vsockserver.MappingProvider) *rescanner {
	if provider == nil {
		provider = &fakeProvider{ready: true}
	}
	r := newRescanner(&vsockserver.ReportCache{}, "", provider, time.Hour)
	r.factsFn = func(string) map[string]string { return nil }
	return r
}

func TestRescanner_Run(t *testing.T) {
	t.Run("should publish to cache on each tick", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r := testRescanner(nil)
			var mu sync.Mutex
			var calls int
			r.scanFn = func(_ context.Context, _, _ string) (*v4.IndexReport, error) {
				return concurrency.WithLock2(&mu, func() (*v4.IndexReport, error) {
					calls++
					return &v4.IndexReport{HashId: "ok"}, nil
				})
			}

			ctx, cancel := context.WithCancel(t.Context())
			stopped := r.runAsync(ctx)
			defer func() {
				cancel()
				<-stopped
			}()
			synctest.Wait()

			time.Sleep(r.interval)
			synctest.Wait()
			assert.Equal(t, 1, concurrency.WithLock1(&mu, func() int { return calls }))

			time.Sleep(r.interval)
			synctest.Wait()
			assert.Equal(t, 2, concurrency.WithLock1(&mu, func() int { return calls }), "should rescan again on the next tick")
		})
	})

	t.Run("should retry sooner than the full interval after a failed rescan", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r := testRescanner(nil)
			var mu sync.Mutex
			var calls int
			r.scanFn = func(_ context.Context, _, _ string) (*v4.IndexReport, error) {
				return concurrency.WithLock2(&mu, func() (*v4.IndexReport, error) {
					calls++
					if calls == 1 {
						return nil, errors.New("transient scan error")
					}
					return &v4.IndexReport{HashId: "ok"}, nil
				})
			}

			ctx, cancel := context.WithCancel(t.Context())
			stopped := r.runAsync(ctx)
			defer func() {
				cancel()
				<-stopped
			}()
			synctest.Wait()

			time.Sleep(r.interval)
			synctest.Wait()
			assert.Equal(t, 1, concurrency.WithLock1(&mu, func() int { return calls }))

			time.Sleep(rescanRetryBaseBackoff)
			synctest.Wait()
			assert.Equal(t, 2, concurrency.WithLock1(&mu, func() int { return calls }), "the failed rescan was never retried")

			time.Sleep(rescanRetryBaseBackoff)
			synctest.Wait()
			assert.Equal(t, 2, concurrency.WithLock1(&mu, func() int { return calls }), "a successful rescan should wait the full interval, not keep the retry backoff")
		})
	})

	t.Run("should stop promptly when the context is cancelled", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r := testRescanner(nil)

			ctx, cancel := context.WithCancel(t.Context())
			stopped := r.runAsync(ctx)

			// Run must return promptly once ctx is cancelled: if it
			// doesn't, the bubble deadlocks on the blocked <-stopped below
			// (nothing left to advance the fake clock), and synctest.Test
			// fails the test on deadlock automatically.
			cancel()
			<-stopped
			// Updaters can still fire onChange after Run has returned.
			r.OnMappingChanged()
		})
	})

	t.Run("a periodic tick is a no-op when the mapping is not yet ready", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r := testRescanner(&fakeProvider{ready: false})
			var calls int
			r.scanFn = func(context.Context, string, string) (*v4.IndexReport, error) {
				calls++
				return &v4.IndexReport{}, nil
			}

			ctx, cancel := context.WithCancel(t.Context())
			stopped := r.runAsync(ctx)
			defer func() {
				cancel()
				<-stopped
			}()
			synctest.Wait()

			time.Sleep(r.interval)
			synctest.Wait()

			assert.Zero(t, calls, "scanFn must not be called while the mapping provider isn't ready")
		})
	})

	t.Run("OnMappingChanged triggers an immediate scan attempt", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r := testRescanner(&fakeProvider{ready: true})
			var calls int
			r.scanFn = func(context.Context, string, string) (*v4.IndexReport, error) {
				calls++
				return &v4.IndexReport{}, nil
			}

			ctx, cancel := context.WithCancel(t.Context())
			stopped := r.runAsync(ctx)
			defer func() {
				cancel()
				<-stopped
			}()
			synctest.Wait()

			r.OnMappingChanged()
			synctest.Wait()

			assert.Equal(t, 1, calls, "OnMappingChanged should trigger a scan without waiting for the next tick")
		})
	})

	t.Run("OnMappingChanged failure retries at backoff not the full interval", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r := testRescanner(nil)
			var mu sync.Mutex
			var calls int
			r.scanFn = func(_ context.Context, _, _ string) (*v4.IndexReport, error) {
				return concurrency.WithLock2(&mu, func() (*v4.IndexReport, error) {
					calls++
					if calls == 1 {
						return nil, errors.New("transient scan error")
					}
					return &v4.IndexReport{HashId: "ok"}, nil
				})
			}

			ctx, cancel := context.WithCancel(t.Context())
			stopped := r.runAsync(ctx)
			defer func() {
				cancel()
				<-stopped
			}()
			synctest.Wait()

			r.OnMappingChanged()
			synctest.Wait()
			assert.Equal(t, 1, concurrency.WithLock1(&mu, func() int { return calls }), "OnMappingChanged should trigger a scan immediately")

			time.Sleep(rescanRetryBaseBackoff)
			synctest.Wait()
			assert.Equal(t, 2, concurrency.WithLock1(&mu, func() int { return calls }), "a failed mapping-triggered scan should retry after the backoff")

			time.Sleep(rescanRetryBaseBackoff)
			synctest.Wait()
			assert.Equal(t, 2, concurrency.WithLock1(&mu, func() int { return calls }), "a successful retry should wait the full interval")
		})
	})

	t.Run("a failed scan clears busy via idle-and-apply-pending", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			provider := &fakeGatedProvider{fakeProvider: fakeProvider{ready: true}}
			r := testRescanner(provider)
			r.scanFn = func(context.Context, string, string) (*v4.IndexReport, error) {
				return nil, errors.New("scan failed")
			}

			ctx, cancel := context.WithCancel(t.Context())
			stopped := r.runAsync(ctx)
			defer func() {
				cancel()
				<-stopped
			}()
			synctest.Wait()

			time.Sleep(r.interval)
			synctest.Wait()

			assert.Equal(t, 1, provider.busyCalls)
			assert.Equal(t, 1, provider.idleCalls, "a failed scan must not leave a Sync stuck behind busy forever")
		})
	})

	t.Run("a successful scan clears busy without waiting for GetReport", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			provider := &fakeGatedProvider{fakeProvider: fakeProvider{ready: true}}
			r := testRescanner(provider)
			r.scanFn = func(context.Context, string, string) (*v4.IndexReport, error) {
				return &v4.IndexReport{HashId: "ok"}, nil
			}

			ctx, cancel := context.WithCancel(t.Context())
			stopped := r.runAsync(ctx)
			defer func() {
				cancel()
				<-stopped
			}()
			synctest.Wait()

			time.Sleep(r.interval)
			synctest.Wait()

			assert.Equal(t, 1, provider.busyCalls)
			assert.Equal(t, 1, provider.idleCalls, "a successful scan must apply a deferred Sync itself; GetReport can be hours away")
		})
	})
}

// TestScanOnce_StoresHashFromBeforeTheScan covers a URL refresh that
// replaces the live mapping while scanFn runs: SetReport must keep the
// hash from before the scan.
func TestScanOnce_StoresHashFromBeforeTheScan(t *testing.T) {
	const hashAtScan = "hash-at-scan-start"
	provider := &fakeProvider{ready: true, hash: hashAtScan}
	r := testRescanner(provider)
	r.scanFn = func(context.Context, string, string) (*v4.IndexReport, error) {
		provider.hash = "hash-after-url-refresh"
		return &v4.IndexReport{HashId: "ok"}, nil
	}

	require.NoError(t, r.scanOnce(t.Context()))
	assert.Equal(t, "hash-after-url-refresh", provider.Hash(), "the live mapping may have moved on")
	assert.Equal(t, hashAtScan, mappingHashFromCache(t, r.cache, provider))
}

func mappingHashFromCache(t *testing.T, cache *vsockserver.ReportCache, provider vsockserver.MappingProvider) string {
	t.Helper()
	handler := vsockserver.NewHandler(cache, "test", provider, nil)
	clientConn, serverConn := net.Pipe()
	go handler.HandleConn(serverConn)

	req, err := proto.Marshal(&pb.VMServiceRequest{
		Method: &pb.VMServiceRequest_GetReport{GetReport: &pb.GetReportRequest{}},
	})
	require.NoError(t, err)
	require.NoError(t, vsockframing.WriteFrame(clientConn, req))

	respData, err := vsockframing.ReadFrame(clientConn, 10<<20)
	require.NoError(t, err)
	_ = clientConn.Close()

	var resp pb.VMServiceResponse
	require.NoError(t, proto.Unmarshal(respData, &resp))
	require.NotNil(t, resp.GetGetReport())
	return resp.GetMeta().GetRepoCpeMappingHash()
}
