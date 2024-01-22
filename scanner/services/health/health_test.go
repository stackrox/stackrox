package health

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockProvider struct {
	ready, live                         bool
	name                                string
	nameCalled, readyCalled, liveCalled bool
}

func (m *mockProvider) Name() string {
	m.nameCalled = true
	return m.name
}

func (m *mockProvider) Ready() bool {
	m.readyCalled = true
	return m.ready
}

func (m *mockProvider) Live() bool {
	m.liveCalled = true
	return m.live
}

func Test_service_check(t *testing.T) {
	type fields struct {
		matcher mockProvider
		indexer mockProvider
	}
	type args struct {
		srv string
	}
	tests := map[string]struct {
		fields fields
		args   args
		check  func(*testing.T, *fields, bool, error)
	}{
		"when service is empty then check everything": {
			args: args{srv: ""},
			check: func(t *testing.T, f *fields, got bool, err error) {
				assert.NoError(t, err)
				assert.True(t, f.indexer.liveCalled || f.indexer.readyCalled || f.matcher.liveCalled || f.matcher.readyCalled)
				assert.False(t, got)
			},
		},
		"when service is matcher then check matcher but not indexer": {
			args: args{srv: "matcher"},
			fields: fields{
				indexer: mockProvider{name: "indexer"},
				matcher: mockProvider{name: "matcher", live: true, ready: true},
			},
			check: func(t *testing.T, f *fields, got bool, err error) {
				assert.NoError(t, err)
				assert.False(t, f.indexer.liveCalled)
				assert.False(t, f.indexer.readyCalled)
				assert.True(t, f.matcher.liveCalled)
				assert.True(t, f.matcher.readyCalled)
				assert.True(t, got)
			},
		},
		"when service is indexer then check indexer but not matcher": {
			args: args{srv: "indexer"},
			fields: fields{
				indexer: mockProvider{name: "indexer", live: true, ready: true},
				matcher: mockProvider{name: "matcher"},
			},
			check: func(t *testing.T, f *fields, got bool, err error) {
				assert.NoError(t, err)
				assert.True(t, f.indexer.liveCalled)
				assert.True(t, f.indexer.readyCalled)
				assert.False(t, f.matcher.liveCalled)
				assert.False(t, f.matcher.readyCalled)
				assert.True(t, got)
			},
		},
		"when service is indexer-liveness then check indexer liveness but not matcher": {
			args: args{srv: "indexer-liveness"},
			fields: fields{
				indexer: mockProvider{name: "indexer", live: true},
				matcher: mockProvider{name: "matcher"},
			},
			check: func(t *testing.T, f *fields, got bool, err error) {
				assert.NoError(t, err)
				assert.True(t, f.indexer.liveCalled)
				assert.False(t, f.indexer.readyCalled)
				assert.False(t, f.matcher.liveCalled)
				assert.False(t, f.matcher.readyCalled)
				assert.True(t, got)
			},
		},
		"when service is indexer-readiness then check indexer readiness but not matcher": {
			args: args{srv: "indexer-readiness"},
			fields: fields{
				indexer: mockProvider{name: "indexer", ready: true},
				matcher: mockProvider{name: "matcher"},
			},
			check: func(t *testing.T, f *fields, got bool, err error) {
				assert.NoError(t, err)
				assert.False(t, f.indexer.liveCalled)
				assert.True(t, f.indexer.readyCalled)
				assert.False(t, f.matcher.liveCalled)
				assert.False(t, f.matcher.readyCalled)
				assert.True(t, got)
			},
		},
		"when unknown service then error": {
			args: args{srv: "something-unknown"},
			fields: fields{
				indexer: mockProvider{name: "indexer"},
				matcher: mockProvider{name: "matcher"},
			},
			check: func(t *testing.T, f *fields, got bool, err error) {
				assert.ErrorContains(t, err, "unknown service")
				assert.False(t, f.indexer.liveCalled)
				assert.False(t, f.indexer.readyCalled)
				assert.False(t, f.matcher.liveCalled)
				assert.False(t, f.matcher.readyCalled)
				assert.False(t, got)
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			s := NewService(&tt.fields.matcher, &tt.fields.indexer)
			got, err := s.check(tt.args.srv)
			tt.check(t, &tt.fields, got, err)
		})
	}
}
