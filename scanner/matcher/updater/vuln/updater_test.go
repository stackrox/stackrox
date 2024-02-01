package vuln

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/quay/claircore/libvuln/updates"
	"github.com/stackrox/rox/scanner/datastore/postgres/mocks"
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

func testHTTPServer(t *testing.T) (*httptest.Server, time.Time) {
	now, err := http.ParseTime(time.Now().UTC().Format(http.TimeFormat))
	require.NoError(t, err)
	buf := strings.NewReader("test")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "test-file", now, buf)
	}))
	t.Cleanup(srv.Close)
	return srv, now
}

func TestUpdate(t *testing.T) {
	srv, now := testHTTPServer(t)

	locker := &testLocker{
		locker: updates.NewLocalLockSource(),
		fail:   true,
	}
	metadataStore := mocks.NewMockMatcherMetadataStore(gomock.NewController(t))
	u := &Updater{
		locker:        locker,
		pool:          nil,
		metadataStore: metadataStore,
		client:        srv.Client(),
		url:           srv.URL,
		root:          t.TempDir(),
		skipGC:        true,
		importVulns: func(_ context.Context, _ io.Reader) error {
			return nil
		},
	}

	// Skip update when locking fails.
	err := u.Update(context.Background())
	assert.NoError(t, err)

	locker.fail = false

	// Successful update.
	metadataStore.EXPECT().
		GetLastVulnerabilityUpdate(gomock.Any()).
		Return(now.Add(-time.Minute), nil)
	metadataStore.EXPECT().
		SetLastVulnerabilityUpdate(gomock.Any(), now).
		Return(nil)
	err = u.Update(context.Background())
	assert.NoError(t, err)

	// No update.
	metadataStore.EXPECT().
		GetLastVulnerabilityUpdate(gomock.Any()).
		Return(now.Add(time.Minute), nil)
	err = u.Update(context.Background())
	assert.NoError(t, err)
}

func TestFetch(t *testing.T) {
	srv, now := testHTTPServer(t)

	u := &Updater{
		client: srv.Client(),
		url:    srv.URL,
		root:   t.TempDir(),
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
