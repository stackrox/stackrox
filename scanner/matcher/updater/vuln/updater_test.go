package vuln

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/quay/claircore"
	"github.com/quay/claircore/datastore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/libvuln/updates"
	"github.com/quay/claircore/test"
	"github.com/quay/zlog"
	"github.com/rs/zerolog"
	"github.com/stackrox/rox/scanner/datastore/postgres/mocks"
	"github.com/stackrox/rox/scanner/updater/jsonblob"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

var _ updates.LockSource = (*testLocker)(nil)

type testLocker struct {
	locker updates.LockSource
	fail   bool
}

func (t *testLocker) TryLock(ctx context.Context, s string) (context.Context, context.CancelFunc) {
	if t.fail {
		ctx, cancel := context.WithCancel(ctx)
		cancel()
		return ctx, cancel
	}
	return t.locker.TryLock(ctx, s)
}

func (t *testLocker) Lock(ctx context.Context, s string) (context.Context, context.CancelFunc) {
	if t.fail {
		ctx, cancel := context.WithCancel(ctx)
		cancel()
		return ctx, cancel
	}
	return t.locker.Lock(ctx, s)
}

func testHTTPServer(t *testing.T, content func(r *http.Request) io.ReadSeeker) (*httptest.Server, time.Time) {
	now, err := http.ParseTime(time.Now().UTC().Format(http.TimeFormat))
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "test-file", now, content(r))
	}))
	t.Cleanup(srv.Close)
	return srv, now
}

func TestMultiBundleUpdate(t *testing.T) {
	t.Setenv("ROX_SCANNER_V4_MULTI_BUNDLE", "true")

	// TODO(ROX-26236): Test with zst files, as a chunk of the updater function is currently untested.
	srv, now := testHTTPServer(t, func(r *http.Request) io.ReadSeeker {
		accept := r.Header.Get("X-Scanner-V4-Accept")
		if accept != "application/vnd.stackrox.scanner-v4.multi-bundle+zip" {
			t.Fatalf("X-Scanner-V4-Accept header should be set to application/vnd.stackrox.scanner-v4.multi-bundle+zip for multi-bundle")
		}
		var buf bytes.Buffer
		zipWriter := zip.NewWriter(&buf)
		err := zipWriter.Close()
		if err != nil {
			t.Fatalf("Failed to close zip writer: %v", err)
		}
		// Return an empty ZIP file.
		return bytes.NewReader(buf.Bytes())
	})

	locker := &testLocker{
		locker: updates.NewLocalLockSource(),
	}
	store := mocks.NewMockMatcherStore(gomock.NewController(t))
	metadataStore := mocks.NewMockMatcherMetadataStore(gomock.NewController(t))
	u := &Updater{
		locker:        locker,
		store:         store,
		metadataStore: metadataStore,
		client:        srv.Client(),
		urls:          []string{srv.URL},
		root:          t.TempDir(),
		skipGC:        false,
		importFunc:    func(_ context.Context, _ io.Reader) error { return nil },
		retryDelay:    1 * time.Second,
		retryMax:      1,
		distManager:   newDistManager(store),
	}

	// Skip update and error when unable to get previous update time.
	metadataStore.EXPECT().
		GetLastVulnerabilityUpdate(gomock.Any()).
		Return(time.Time{}, errors.New("err"))
	err := u.Update(context.Background())
	assert.Error(t, err)
	assert.Nil(t, u.KnownDistributions())

	dists := []claircore.Distribution{
		{
			ID: "0",
		},
		{
			ID: "1",
		},
	}

	// Successful update.
	metadataStore.EXPECT().
		GetLastVulnerabilityUpdate(gomock.Any()).
		Return(now.Add(-time.Minute), nil)
	metadataStore.EXPECT().
		GCVulnerabilityUpdates(gomock.Any(), gomock.Any(), now).
		Return(nil)
	metadataStore.EXPECT().
		GetLastVulnerabilityUpdate(gomock.Any()).
		Return(now, nil)
	store.EXPECT().
		GC(gomock.Any(), gomock.Any()).
		Return(int64(0), nil)
	store.EXPECT().
		Distributions(gomock.Any()).
		Return(dists, nil)
	err = u.Update(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, dists, u.KnownDistributions())

	// No update.
	metadataStore.EXPECT().
		GetLastVulnerabilityUpdate(gomock.Any()).
		Return(now.Add(time.Minute), nil)
	err = u.Update(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, dists, u.KnownDistributions())
}

func TestFetch(t *testing.T) {
	srv, now := testHTTPServer(t, func(_ *http.Request) io.ReadSeeker {
		return strings.NewReader("test")
	})

	u := &Updater{
		client:     srv.Client(),
		urls:       []string{srv.URL},
		root:       t.TempDir(),
		retryDelay: 1 * time.Second,
		retryMax:   1,
	}

	// Fetch file, as it's modified after the given time.
	f, timestamp, err := u.fetch(context.Background(), time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, f)
	assert.Equal(t, now, timestamp)

	// Fetch file, as it's modified after the given time.
	f, timestamp, err = u.fetch(context.Background(), now.Add(-time.Minute))
	require.NoError(t, err)
	assert.NotNil(t, f)
	assert.Equal(t, now, timestamp)

	// Do not fetch file, as it's not modified after the given time.
	f, timestamp, err = u.fetch(context.Background(), now.Add(time.Minute))
	require.NoError(t, err)
	assert.Nil(t, f)
	assert.Equal(t, time.Time{}, timestamp)
}

func TestFetchRCBundle(t *testing.T) {
	var paths []string
	now, err := http.ParseTime(time.Now().UTC().Format(http.TimeFormat))
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path == "/v1-rc/vulnerabilities.zip" {
			http.ServeContent(w, r, "test-file", now, strings.NewReader("rc"))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	u := &Updater{
		client:     srv.Client(),
		urls:       []string{srv.URL + "/v1-rc/vulnerabilities.zip", srv.URL + "/v1/vulnerabilities.zip"},
		root:       t.TempDir(),
		retryDelay: 1 * time.Second,
		retryMax:   1,
	}

	f, timestamp, err := u.fetch(context.Background(), time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, f)
	assert.Equal(t, now, timestamp)
	assert.Equal(t, []string{"/v1-rc/vulnerabilities.zip"}, paths)
}

func TestFetchRCBundleFallback(t *testing.T) {
	var paths []string
	now, err := http.ParseTime(time.Now().UTC().Format(http.TimeFormat))
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/v1-rc/vulnerabilities.zip":
			http.NotFound(w, r)
		case "/v1/vulnerabilities.zip":
			http.ServeContent(w, r, "test-file", now, strings.NewReader("ga"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	u := &Updater{
		client:     srv.Client(),
		urls:       []string{srv.URL + "/v1-rc/vulnerabilities.zip", srv.URL + "/v1/vulnerabilities.zip"},
		root:       t.TempDir(),
		retryDelay: 1 * time.Second,
		retryMax:   1,
	}

	f, timestamp, err := u.fetch(context.Background(), time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, f)
	assert.Equal(t, now, timestamp)
	assert.Equal(t, []string{"/v1-rc/vulnerabilities.zip", "/v1/vulnerabilities.zip"}, paths)
}

func TestUpdater_Initialized(t *testing.T) {
	t.Run("when initialized then return ready", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		metaMock := mocks.NewMockMatcherMetadataStore(ctrl)
		u := Updater{
			metadataStore: metaMock,
		}
		u.initialized.Store(true)
		got := u.Initialized(context.Background())
		assert.True(t, got, `expecting "ready" got "not ready"`)
	})

	t.Run("when not initialized and get last update is empty then return not ready", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		metaMock := mocks.NewMockMatcherMetadataStore(ctrl)
		metaMock.
			EXPECT().
			GetLastVulnerabilityUpdate(gomock.Any())
		u := Updater{
			metadataStore: metaMock,
		}
		got := u.Initialized(context.Background())
		assert.False(t, got, `expecting "not ready" got "ready"`)
	})

	t.Run("when not initialized and get last update is not empty then return ready", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		metaMock := mocks.NewMockMatcherMetadataStore(ctrl)
		metaMock.
			EXPECT().
			GetLastVulnerabilityUpdate(gomock.Any()).
			Return(time.Now(), nil) // Non-zero
		u := Updater{
			metadataStore: metaMock,
		}
		got := u.Initialized(context.Background())
		assert.True(t, got, `expecting "ready" got "not ready"`)
	})

	t.Run("when not initialized and get last update fails then log return not ready", func(t *testing.T) {
		b := &bytes.Buffer{}
		l := zerolog.New(b)
		zlog.Set(&l)
		ctx := zlog.Test(context.Background(), t)
		ctrl := gomock.NewController(t)
		metaMock := mocks.NewMockMatcherMetadataStore(ctrl)
		metaMock.
			EXPECT().
			GetLastVulnerabilityUpdate(gomock.Any()).
			Return(time.Unix(0, 0), errors.New("last update failed (fake error)"))
		u := Updater{
			metadataStore: metaMock,
		}
		u.initialized.Store(false)
		got := u.Initialized(ctx)
		assert.False(t, got, `expecting "not ready" got "ready"`)
		assert.Contains(t, `"did not get previous vuln update timestamp"`, b.String())
		assert.Contains(t, `"error":"last update failed (fake error)"`, b.String())
		assert.Contains(t, `"level":"error"`, b.String())
	})
}

func TestUpdater_Import(t *testing.T) {
	ctx := zlog.Test(context.Background(), t)
	ctrl := gomock.NewController(t)

	// Represents one vulnerability or enrichment iteration.
	type iteration struct {
		O   *driver.UpdateOperation
		V   []*claircore.Vulnerability
		E   []*driver.EnrichmentRecord
		Err string // Set to fail the iteration.
	}

	// Mock jsonblob.Iterate with fake iterations. Operations, vulnerabilities and
	// enrichments are yielded in order. If Err is set, yields the operation followed
	// by an error, unless the operation is nil, then yields error immediately.
	mockIterate := func(t *testing.T, ops ...*iteration) func(_ io.Reader) (jsonblob.OperationIter, func() error) {
		var err error
		return func(_ io.Reader) (jsonblob.OperationIter, func() error) {
			opIt := func(yield func(*driver.UpdateOperation, jsonblob.RecordIter) bool) {
				for _, o := range ops {
					if o.O == nil && o.Err != "" {
						err = errors.New(o.Err)
						break
					}
					ok := yield(o.O, func(yield func(*claircore.Vulnerability, *driver.EnrichmentRecord) bool) {
						if o.Err != "" {
							err = errors.New(o.Err)
							return
						}
						for _, v := range o.V {
							if !yield(v, nil) {
								break
							}
						}
						for _, e := range o.E {
							if !yield(nil, e) {
								break
							}
						}
					})
					if !ok {
						break
					}
				}
			}
			itErr := func() error {
				return err
			}
			return opIt, itErr
		}
	}

	t.Run("when operation exists then skip updates", func(t *testing.T) {
		// First operation, no records. Second, with records.
		want := []*iteration{
			{
				O: &driver.UpdateOperation{
					Kind:        driver.VulnerabilityKind,
					Updater:     "fake-updater",
					Fingerprint: "fake-fingerprint",
				},
			},
			{
				O: &driver.UpdateOperation{
					Kind:        driver.VulnerabilityKind,
					Updater:     "another-fake-updater",
					Fingerprint: "another-fake-fingerprint",
				},
				V: test.GenUniqueVulnerabilities(5, "another-fake-updater"),
			},
		}

		// Database already has the same operation.
		storeMock := mocks.NewMockMatcherStore(ctrl)
		var gotVulns []*claircore.Vulnerability
		gomock.InOrder(
			storeMock.
				EXPECT().
				GetUpdateOperations(gomock.Any(), driver.VulnerabilityKind, "fake-updater").
				Return(map[string][]driver.UpdateOperation{"fake-updater": {
					driver.UpdateOperation{Fingerprint: "fake-fingerprint"},
				}}, nil),
			storeMock.
				EXPECT().
				GetUpdateOperations(gomock.Any(), driver.VulnerabilityKind, "another-fake-updater").
				Return(map[string][]driver.UpdateOperation{"another-fake-updater": {}}, nil),
			storeMock.
				EXPECT().
				UpdateVulnerabilitiesIter(gomock.Any(), "another-fake-updater", driver.Fingerprint("another-fake-fingerprint"), gomock.Any()).
				Do(func(_, _, _ any, it datastore.VulnerabilityIter) {
					it(func(v *claircore.Vulnerability, err error) bool {
						assert.NoError(t, err)
						gotVulns = append(gotVulns, v)
						return true
					})
				}),
		)

		u := &Updater{store: storeMock}
		u.iterateFunc = mockIterate(t, want...)

		err := u.Import(ctx, nil)

		assert.NoError(t, err)
		if !cmp.Equal(gotVulns, want[1].V) {
			t.Error(cmp.Diff(gotVulns, want[1].V))
		}
	})

	t.Run("when new vuln and enrichment operations then update", func(t *testing.T) {
		// One operation, one vulnerability.
		const vulnUpdater = "fake-vuln-updater"
		const vulnFingerprint = "fake-vuln-fingerprint"

		const enrichUpdater = "fake-enrich-updater"
		const enrichFingerprint = "fake-enrich-fingerprint"

		var want, got []*iteration

		want = append(want,
			&iteration{
				O: &driver.UpdateOperation{
					Kind:        driver.VulnerabilityKind,
					Updater:     vulnUpdater,
					Fingerprint: vulnFingerprint,
				},
				V: test.GenUniqueVulnerabilities(10, vulnUpdater),
			},
			&iteration{
				O: &driver.UpdateOperation{
					Kind:        driver.EnrichmentKind,
					Updater:     enrichUpdater,
					Fingerprint: enrichFingerprint,
				},
				E: func() (ens []*driver.EnrichmentRecord) {
					for _, e := range test.GenEnrichments(15) {
						ep := e
						ens = append(ens, &ep)
					}
					return
				}(),
			})

		storeMock := mocks.NewMockMatcherStore(ctrl)
		for i := range want {
			got = append(got, &iteration{})
			wantIter := want[i]
			gotIter := got[i]
			opCall := storeMock.
				EXPECT().
				GetUpdateOperations(gomock.Any(), wantIter.O.Kind, wantIter.O.Updater).
				Return(map[string][]driver.UpdateOperation{wantIter.O.Updater: {}}, nil).
				Do(func(_ any, kind driver.UpdateKind, updater string) {
					gotIter.O = &driver.UpdateOperation{
						Kind:        kind,
						Updater:     updater,
						Fingerprint: wantIter.O.Fingerprint,
					}
				})
			switch wantIter.O.Kind {
			case driver.VulnerabilityKind:
				storeMock.
					EXPECT().
					UpdateVulnerabilitiesIter(gomock.Any(), wantIter.O.Updater, wantIter.O.Fingerprint, gomock.Any()).
					Do(func(_, _, _ any, it datastore.VulnerabilityIter) {
						it(func(v *claircore.Vulnerability, err error) bool {
							assert.NoError(t, err)
							gotIter.V = append(gotIter.V, v)
							return true
						})
					}).
					After(opCall)
			case driver.EnrichmentKind:
				storeMock.
					EXPECT().
					UpdateEnrichmentsIter(gomock.Any(), wantIter.O.Updater, wantIter.O.Fingerprint, gomock.Any()).
					Do(func(_, _, _ any, it datastore.EnrichmentIter) {
						it(func(e *driver.EnrichmentRecord, err error) bool {
							assert.NoError(t, err)
							gotIter.E = append(gotIter.E, e)
							return true
						})
					}).
					After(opCall)
			}
		}

		u := &Updater{store: storeMock}
		u.iterateFunc = mockIterate(t, want...)
		err := u.Import(ctx, nil)

		assert.NoError(t, err)
		if !cmp.Equal(got, want) {
			t.Error(cmp.Diff(got, want))
		}
	})

	t.Run("when iteration fails then stop and return the error", func(t *testing.T) {
		var want, got []*iteration

		want = append(want,
			&iteration{
				O: &driver.UpdateOperation{
					Kind:        driver.VulnerabilityKind,
					Updater:     "fake-updater",
					Fingerprint: "fake-updater-fingerprint",
				},
				V: test.GenUniqueVulnerabilities(10, "fake-updater"),
			},
			&iteration{
				O: &driver.UpdateOperation{
					Kind:        driver.VulnerabilityKind,
					Updater:     "another-fake-updater",
					Fingerprint: "fake-updater-fingerprint",
				},
				Err: "fake error in the iteration",
			})

		storeMock := mocks.NewMockMatcherStore(ctrl)
		got = append(got, &iteration{}, &iteration{})

		gomock.InOrder(
			// First iteration.
			storeMock.
				EXPECT().
				GetUpdateOperations(gomock.Any(), want[0].O.Kind, want[0].O.Updater).
				Return(map[string][]driver.UpdateOperation{want[0].O.Updater: {}}, nil).
				Do(func(_ any, kind driver.UpdateKind, updater string) {
					got[0].O = &driver.UpdateOperation{
						Kind:        kind,
						Updater:     updater,
						Fingerprint: want[0].O.Fingerprint,
					}
				}),
			storeMock.
				EXPECT().
				UpdateVulnerabilitiesIter(gomock.Any(), want[0].O.Updater, want[0].O.Fingerprint, gomock.Any()).
				Do(func(_, _, _ any, it datastore.VulnerabilityIter) {
					it(func(v *claircore.Vulnerability, err error) bool {
						assert.NoError(t, err)
						got[0].V = append(got[0].V, v)
						return true
					})
				}),

			// Second iteration.
			storeMock.
				EXPECT().
				GetUpdateOperations(gomock.Any(), want[1].O.Kind, want[1].O.Updater).
				Return(map[string][]driver.UpdateOperation{want[1].O.Updater: {}}, nil).
				Do(func(_ any, kind driver.UpdateKind, updater string) {
					got[1].O = &driver.UpdateOperation{
						Kind:        kind,
						Updater:     updater,
						Fingerprint: want[1].O.Fingerprint,
					}
				}),
			storeMock.
				EXPECT().
				UpdateVulnerabilitiesIter(gomock.Any(), want[1].O.Updater, want[1].O.Fingerprint, gomock.Any()).
				DoAndReturn(func(_, _, _ any, it datastore.VulnerabilityIter) (any, error) {
					var iterErr error
					it(func(v *claircore.Vulnerability, err error) bool {
						assert.Error(t, err, "fake error")
						got[1].Err = err.Error()
						iterErr = fmt.Errorf("iterating on vulnerabilities: %w", err)
						return false
					})
					return nil, iterErr
				}),
		)

		u := &Updater{store: storeMock}
		u.iterateFunc = mockIterate(t, want...)
		err := u.Import(ctx, nil)

		assert.Error(t, err, "iterating on vulnerabilities: fake error")
		if !cmp.Equal(got, want) {
			t.Error(cmp.Diff(got, want))
		}
	})
}
