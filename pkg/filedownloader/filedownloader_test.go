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
	d.Start()

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

func TestWithRetryPolicy_DefaultsToSingleAttempt(t *testing.T) {
	d := New("http://example.com", "/tmp/test", time.Hour)
	assert.Equal(t, 1, d.retryMaxAttempts)
}

func TestWithRetryPolicy_OverridesAttemptsAndBackoff(t *testing.T) {
	d := New("http://example.com", "/tmp/test", time.Hour, WithRetryPolicy(3, 2*time.Second))
	assert.Equal(t, 3, d.retryMaxAttempts)
	assert.Equal(t, 2*time.Second, d.retryBaseBackoff)
}

func TestWithRetryPolicy_PanicsOnMaxAttemptsBelowOne(t *testing.T) {
	assert.Panics(t, func() { WithRetryPolicy(0, time.Second) })
	assert.Panics(t, func() { WithRetryPolicy(-1, time.Second) })
}

func TestWithRetryPolicy_PanicsOnNegativeBaseBackoff(t *testing.T) {
	assert.Panics(t, func() { WithRetryPolicy(3, -time.Second) })
}

func TestWithRetryPolicy_AllowsZeroBaseBackoff(t *testing.T) {
	d := New("http://example.com", "/tmp/test", time.Hour, WithRetryPolicy(3, 0))
	assert.Equal(t, time.Duration(0), d.retryBaseBackoff)
}

func TestDownloadOnce_RetriesOnTransientFailureThenSucceeds(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			if calls.Add(1) <= 2 {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}, nil
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok"))}, nil
		})}

		filePath := filepath.Join(t.TempDir(), "data.json")
		d := New("http://example.invalid/mapping.json", filePath, time.Hour,
			WithHTTPClient(client),
			WithRetryPolicy(3, 2*time.Second),
		)

		require.NoError(t, d.DownloadOnce(t.Context()))
		assert.Equal(t, int32(3), calls.Load())

		data, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, "ok", string(data))
	})
}

func TestDownloadOnce_ExhaustsRetriesAndReturnsError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}, nil
		})}

		filePath := filepath.Join(t.TempDir(), "data.json")
		d := New("http://example.invalid/mapping.json", filePath, time.Hour,
			WithHTTPClient(client),
			WithRetryPolicy(3, time.Millisecond),
		)

		err := d.DownloadOnce(t.Context())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "after 3 attempts", "should say how many attempts were made, not just the last error")
		assert.Equal(t, int32(3), calls.Load(), "should stop after retryMaxAttempts")
	})
}

func TestDownloadOnce_StopsRetryingPromptlyWhenContextCancelledDuringBackoff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			calls.Add(1)
			return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}, nil
		})}

		filePath := filepath.Join(t.TempDir(), "data.json")
		d := New("http://example.invalid/mapping.json", filePath, time.Hour,
			WithHTTPClient(client),
			WithRetryPolicy(3, 2*time.Second),
		)

		ctx, cancel := context.WithCancel(t.Context())
		go func() {
			time.Sleep(10 * time.Millisecond) // fires well before the 2s backoff elapses
			cancel()
		}()

		err := d.DownloadOnce(ctx)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, int32(1), calls.Load(), "should not attempt a retry once ctx is observed cancelled during backoff")
	})
}

func TestRun(t *testing.T) {
	t.Run("should not download immediately, unlike Start", func(t *testing.T) {
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
			done := make(chan error, 1)
			go func() { done <- d.Run(ctx) }()
			defer func() {
				cancel()
				<-done
			}()
			synctest.Wait() // Run is blocked waiting for the first tick

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
			done := make(chan error, 1)
			go func() { done <- d.Run(ctx) }()
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

	t.Run("should stop promptly when the context is cancelled and return ctx.Err()", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			d := New("http://example.invalid/mapping.json", filepath.Join(t.TempDir(), "data.json"), minInterval)

			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan error, 1)
			go func() { done <- d.Run(ctx) }()
			synctest.Wait()

			cancel()
			err := <-done
			assert.ErrorIs(t, err, context.Canceled)
		})
	})
}
