package filedownloader

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuccessfulDownload(t *testing.T) {
	content := `{"keys": [{"name": "test-key"}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(content))
	}))
	defer server.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")

	d := New(server.URL, filePath, time.Hour, WithHTTPClient(server.Client()))
	require.NoError(t, d.doDownload(t.Context()))

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestHTTPErrorDoesNotModifyFile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")
	original := "original"
	require.NoError(t, os.WriteFile(filePath, []byte(original), 0600))

	d := New(server.URL, filePath, time.Hour, WithHTTPClient(server.Client()))
	err := d.doDownload(t.Context())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected HTTP status 500 Internal Server Error")

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, original, string(data))
}

func TestOversizedResponseRejected(t *testing.T) {
	largeBody := strings.Repeat("x", defaultMaxSize+100)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(largeBody))
	}))
	defer server.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")

	d := New(server.URL, filePath, time.Hour, WithHTTPClient(server.Client()))
	err := d.doDownload(t.Context())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum size")
	assert.NoFileExists(t, filePath)
}

func TestSequentialDownloads(t *testing.T) {
	v1 := `{"version": 1}`
	v2 := `{"version": 2}`

	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if callCount.Add(1) == 1 {
			_, _ = w.Write([]byte(v1))
		} else {
			_, _ = w.Write([]byte(v2))
		}
	}))
	defer server.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")

	d := New(server.URL, filePath, time.Hour, WithHTTPClient(server.Client()))

	require.NoError(t, d.doDownload(t.Context()))
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, v1, string(data))

	require.NoError(t, d.doDownload(t.Context()))
	data, err = os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, v2, string(data))
}

func TestStopSignal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")

	d := New(server.URL, filePath, minInterval, WithHTTPClient(server.Client()))
	require.NoError(t, d.Start(t.Context(), false))

	require.Eventually(t, func() bool {
		_, err := os.Stat(filePath)
		return err == nil
	}, 2*time.Second, 50*time.Millisecond, "downloader did not write the file")

	done := make(chan struct{})
	go func() { d.Stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("downloader did not stop within timeout")
	}
}

func TestStart(t *testing.T) {
	t.Run("should fail fast without downloading when the destination directory cannot be created", func(t *testing.T) {
		for name, waitForInitial := range map[string]bool{"waiting for initial": true, "not waiting for initial": false} {
			t.Run(name, func(t *testing.T) {
				dir := t.TempDir()
				blocker := filepath.Join(dir, "blocker")
				require.NoError(t, os.WriteFile(blocker, []byte("x"), 0600))
				// blocker is a regular file, so MkdirAll for a path nested under it
				// fails - simulating a persistent, non-network directory problem.
				filePath := filepath.Join(blocker, "nested", "data.json")

				var calls atomic.Int32
				client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
					calls.Add(1)
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok"))}, nil
				})}
				d := New("http://example.invalid/mapping.json", filePath, minInterval, WithHTTPClient(client))

				err := d.Start(t.Context(), waitForInitial)
				require.Error(t, err)
				assert.ErrorContains(t, err, "creating directory")
				assert.Equal(t, int32(0), calls.Load(), "should not attempt a download")

				// No goroutine was ever spawned; Stop must still return promptly.
				done := make(chan struct{})
				go func() { d.Stop(); close(done) }()
				select {
				case <-done:
				case <-time.After(2 * time.Second):
					t.Fatal("Stop did not return after a failed Start")
				}
			})
		}
	})

	t.Run("should block on and return the initial download's error when waitForInitial is true", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		t.Cleanup(server.Close)

		d := New(server.URL, filepath.Join(t.TempDir(), "data.json"), minInterval, WithHTTPClient(server.Client()))
		err := d.Start(t.Context(), true)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "503")
	})

	t.Run("should not block when waitForInitial is false, even if the first download fails", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		t.Cleanup(server.Close)

		d := New(server.URL, filepath.Join(t.TempDir(), "data.json"), minInterval, WithHTTPClient(server.Client()))
		require.NoError(t, d.Start(t.Context(), false))
		t.Cleanup(d.Stop)
	})
}

func TestOnCompleteCallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")

	var gotErr error
	var gotDuration time.Duration
	d := New(server.URL, filePath, time.Hour,
		WithHTTPClient(server.Client()),
		WithOnComplete(func(err error, dur time.Duration) {
			gotErr = err
			gotDuration = dur
		}),
	)

	require.NoError(t, d.DownloadOnce(t.Context()))
	assert.NoError(t, gotErr)
	assert.Greater(t, gotDuration, time.Duration(0))
}

func TestOnCompleteCallbackOnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")

	var gotErr error
	d := New(server.URL, filePath, time.Hour,
		WithHTTPClient(server.Client()),
		WithOnComplete(func(err error, _ time.Duration) {
			gotErr = err
		}),
	)

	assert.Error(t, d.DownloadOnce(t.Context()))
	assert.Error(t, gotErr)
	assert.Contains(t, gotErr.Error(), "503")
}

func TestClampsInterval(t *testing.T) {
	d := New("http://example.com", "/tmp/test", time.Millisecond)
	assert.GreaterOrEqual(t, d.interval, minInterval)

	d = New("http://example.com", "/tmp/test", minInterval)
	assert.Equal(t, minInterval, d.interval)

	long := 2 * minInterval
	d = New("http://example.com", "/tmp/test", long)
	assert.Equal(t, long, d.interval)
}

func TestCustomMaxSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("too big"))
	}))
	defer server.Close()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "data.json")

	d := New(server.URL, filePath, time.Hour,
		WithHTTPClient(server.Client()),
		WithMaxSize(3),
	)
	err := d.doDownload(t.Context())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum size")
}

func TestAtomicWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.json")

	require.NoError(t, atomicWriteFile(path, []byte("hello")))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func TestAtomicWriteFile_CreatesParentDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "output.json")

	require.NoError(t, atomicWriteFile(path, []byte("hello")))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}

// roundTripperFunc adapts a function to http.RoundTripper, letting tests
// simulate HTTP responses (including transient failures) with no real
// network I/O, which keeps retry/backoff tests fast and safe to run inside
// a synctest bubble.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestRetryDefaults(t *testing.T) {
	d := New("http://example.com", "/tmp/test", time.Hour)
	assert.Equal(t, defaultRetryMax, d.retryMax)
	assert.Equal(t, defaultRetryWaitMin, d.retryWaitMin)
}

func TestWithRetryMax_OverridesDefault(t *testing.T) {
	d := New("http://example.com", "/tmp/test", time.Hour, WithRetryMax(2))
	assert.Equal(t, 2, d.retryMax)
}

func TestWithRetryWaitMin_OverridesDefault(t *testing.T) {
	d := New("http://example.com", "/tmp/test", time.Hour, WithRetryWaitMin(2*time.Second))
	assert.Equal(t, 2*time.Second, d.retryWaitMin)
}

func TestDownloadOnce_RetriesViaRetryableHTTPThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)

	filePath := filepath.Join(t.TempDir(), "data.json")
	d := New(server.URL, filePath, time.Hour,
		WithRetryMax(2),
		WithRetryWaitMin(time.Millisecond),
	)

	require.NoError(t, d.DownloadOnce(t.Context()))
	assert.Equal(t, int32(3), calls.Load())

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, "ok", string(data))
}

// TestTickLoop exercises tickLoop directly (rather than through Start),
// since that is where Start's periodic behavior after the initial
// download actually lives.
func TestTickLoop(t *testing.T) {
	t.Run("should not download immediately", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			var calls atomic.Int32
			client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok"))}, nil
			})}
			d := New("http://example.invalid/mapping.json", filepath.Join(t.TempDir(), "data.json"), minInterval,
				WithHTTPClient(client),
			)

			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan struct{})
			go func() { d.tickLoop(ctx); close(done) }()
			defer func() {
				cancel()
				<-done
			}()
			synctest.Wait() // tickLoop is blocked waiting for the first tick

			assert.Equal(t, int32(0), calls.Load())
		})
	})

	t.Run("should download on each tick", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			var calls atomic.Int32
			client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
				calls.Add(1)
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok"))}, nil
			})}
			d := New("http://example.invalid/mapping.json", filepath.Join(t.TempDir(), "data.json"), minInterval,
				WithHTTPClient(client),
			)

			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan struct{})
			go func() { d.tickLoop(ctx); close(done) }()
			defer func() {
				cancel()
				<-done
			}()
			synctest.Wait()

			time.Sleep(minInterval)
			synctest.Wait()
			assert.Equal(t, int32(1), calls.Load())

			time.Sleep(minInterval)
			synctest.Wait()
			assert.Equal(t, int32(2), calls.Load(), "should download again on the next tick")
		})
	})

	t.Run("should stop promptly when the context is cancelled", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			d := New("http://example.invalid/mapping.json", filepath.Join(t.TempDir(), "data.json"), minInterval)

			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan struct{})
			go func() { d.tickLoop(ctx); close(done) }()
			synctest.Wait()

			cancel()
			<-done
		})
	})
}
