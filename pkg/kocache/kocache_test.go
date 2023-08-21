package kocache

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
)

func TestStartOffline(t *testing.T) {
	tests := map[string]struct {
		initial *options
		want    *options
	}{
		"Transition from offline should be offline": {
			initial: &options{
				StartOnline: false,
			},
			want: &options{
				StartOnline: false,
			},
		},
		"Transition from online should be offline": {
			initial: &options{
				StartOnline: true,
			},
			want: &options{
				StartOnline: false,
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			o := tt.initial
			StartOffline()(o)
			assert.Equal(t, tt.want.StartOnline, o.StartOnline)
		})
	}
}

func Test_applyDefaults(t *testing.T) {
	tests := map[string]struct {
		initial *options
		want    *options
	}{
		"Default values should be as defined": {
			initial: &options{},
			want: &options{
				ObjMemLimit:      1048576,
				ObjHardLimit:     10485760,
				CleanupThreshold: 10,
				CleanupAge:       300000000000,
				ErrorCleanUpAge:  30000000000,
				CleanupInterval:  60000000000,
				StartOnline:      true,
				ModifyRequest:    nil,
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			o := tt.initial
			applyDefaults(o)
			assert.Equal(t, tt.want, o)
		})
	}
}

func Test_koCache_GoOffline(t *testing.T) {
	tests := map[string]struct {
		startOnline bool
	}{
		"Starting online":  {startOnline: true},
		"Starting offline": {startOnline: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			fn := func(o *options) {
				if tt.startOnline {
					o.StartOnline = true
				}
			}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(errors.New("test ended"))
			c := New(ctx, nil, "URL", fn)
			c.GoOffline()
			assert.False(t, c.IsOnline())
		})
	}
}

func Test_koCache_GoOnline(t *testing.T) {
	for _, startOnline := range []bool{true, false} {
		t.Run(fmt.Sprintf("Start online: %t", startOnline), func(t *testing.T) {
			fn := func(o *options) {
				o.StartOnline = startOnline
			}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(errors.New("test ended"))
			c := New(ctx, nil, "URL", fn)
			c.GoOnline()
			assert.True(t, c.IsOnline())
		})
	}
}

func Test_koCache_cleanup(t *testing.T) {
	t.Skip("test unimplemented")
}

func Test_koCache_getOrAddEntry(t *testing.T) {
	errEntryExpired := errors.New("entry expired")
	en1 := &entry{
		done:         concurrency.NewErrorSignal(),
		references:   sync.WaitGroup{},
		data:         nil,
		creationTime: time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local),
		lastAccess:   0,
	}
	expiredEntry := &entry{
		done:         concurrency.NewErrorSignal(),
		references:   sync.WaitGroup{},
		data:         nil,
		creationTime: time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local),
		lastAccess:   0,
	}
	expiredEntry.done.SignalWithError(errEntryExpired)
	assert.Error(t, expiredEntry.DoneSig().Wait())
	assert.ErrorIs(t, expiredEntry.DoneSig().Wait(), errEntryExpired)
	assert.True(t, expiredEntry.DoneSig().IsDone())

	tests := map[string]struct {
		entries            map[string]*entry
		key                string
		centralReachable   bool
		centralReplyStatus int
		centralReplyError  error
		centralReplyBody   string
		wantCentralCall    bool
		wantErr            bool
	}{
		"Existing non-expired entry shall be found": {
			entries:          map[string]*entry{"en1": en1},
			key:              "en1",
			wantCentralCall:  false,
			centralReachable: true,
			wantErr:          false,
		},
		"Existing expired entry shall be found and replaced by a fresh one from Central": {
			entries:            map[string]*entry{"expired": expiredEntry},
			key:                "expired",
			wantCentralCall:    true,
			centralReachable:   true,
			centralReplyStatus: 200,
			wantErr:            false,
		},
		"Non-existing entry shall trigger a call to central": {
			entries:            make(map[string]*entry),
			key:                "en2",
			wantCentralCall:    true,
			centralReachable:   true,
			centralReplyStatus: 200,
			wantErr:            false,
		},
		"Call to central in offline mode shall not be attempted and yield an error": {
			entries:          make(map[string]*entry),
			key:              "en2",
			wantCentralCall:  false,
			centralReachable: false,
			wantErr:          true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			cli := newMockHTTPClient(tt.centralReplyStatus, tt.centralReplyError, io.NopCloser(bytes.NewBufferString(tt.centralReplyBody)))
			c := New(context.Background(), cli, "/")
			c.entries = tt.entries
			switch tt.centralReachable {
			case true:
				c.GoOnline()
			case false:
				c.GoOffline()
			}

			got, err := c.getOrAddEntry(tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.wantCentralCall {
					// Wait until `Populate` method finishes
					err, ok := got.DoneSig().WaitWithTimeout(time.Second)
					assert.True(t, ok)
					assert.NoError(t, err)
				}
			}
			assert.Equal(t, tt.wantCentralCall, cli.HasBeenCalled())
		})
	}
}

func newMockHTTPClient(code int, err error, body io.ReadCloser) *mockClient {
	return &mockClient{
		err:  err,
		code: code,
		body: body,
	}
}

type mockClient struct {
	err           error
	code          int
	body          io.ReadCloser
	hasBeenCalled bool
}

func (m *mockClient) HasBeenCalled() bool {
	return m.hasBeenCalled
}

func (m *mockClient) Do(_ *http.Request) (*http.Response, error) {
	m.hasBeenCalled = true
	if m.err != nil {
		return &http.Response{
			StatusCode: m.code,
			Header:     nil,
			Body:       m.body,
		}, m.err
	}
	return &http.Response{
		StatusCode: m.code,
		Header:     nil,
		Body:       m.body,
	}, nil
}
