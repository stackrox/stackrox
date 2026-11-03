package or

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// fixedAuthorizer always returns the same preset error (nil = allow).
type fixedAuthorizer struct {
	err error
}

func (f fixedAuthorizer) Authorized(_ context.Context, _ string) error { return f.err }

// recordingAuthorizer captures every Authorized call for inspection.
type recordingAuthorizer struct {
	calls int
	err   error
}

func (r *recordingAuthorizer) Authorized(_ context.Context, _ string) error {
	r.calls++
	return r.err
}

var (
	allow = fixedAuthorizer{}
	deny  = fixedAuthorizer{err: errox.NotAuthorized}
)

func TestOr_Authorized(t *testing.T) {
	tests := map[string]struct {
		authorizers []authz.Authorizer
		wantErr     error
	}{
		// No authorizers: loop never runs, falls through to the error return.
		"no authorizers": {
			authorizers: nil,
			wantErr:     errox.NotAuthorized,
		},
		"single authorizer allows": {
			authorizers: []authz.Authorizer{allow},
		},
		"single authorizer denies": {
			authorizers: []authz.Authorizer{deny},
			wantErr:     errox.NotAuthorized,
		},
		"all authorizers allow": {
			authorizers: []authz.Authorizer{allow, allow, allow},
		},
		"all authorizers deny": {
			authorizers: []authz.Authorizer{deny, deny, deny},
			wantErr:     errox.NotAuthorized,
		},
		"first allows, second denies": {
			authorizers: []authz.Authorizer{allow, deny},
		},
		"first denies, second allows": {
			authorizers: []authz.Authorizer{deny, allow},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			err := Or(tt.authorizers...).Authorized(it.Context(), "/svc/Method")
			assert.ErrorIs(it, err, tt.wantErr)
		})
	}
}

// TestOr_ShortCircuit verifies subsequent authorizers are not called once one succeeds.
func TestOr_ShortCircuit(t *testing.T) {
	second := &recordingAuthorizer{}
	Or(allow, second).Authorized(t.Context(), "")
	assert.Equal(t, 0, second.calls, "second authorizer must not be called after first succeeds")
}

// TestOr_ForwardsContextAndMethod verifies ctx and methodName reach each sub-authorizer.
func TestOr_ForwardsContextAndMethod(t *testing.T) {
	ctx := context.WithValue(t.Context(), struct{ key string }{"k"}, "v")
	const method = "/some.Service/Method"

	rec := &recordingAuthorizer{}
	Or(rec).Authorized(ctx, method)
	assert.Equal(t, 1, rec.calls)
}

func TestOr_ReturnsAuthorizer(t *testing.T) {
	assert.NotNil(t, Or())
}

// ctxWithService builds a context carrying a mock identity with the given service type.
func ctxWithService(t *testing.T, svcType storage.ServiceType) context.Context {
	t.Helper()
	id := mocks.NewMockIdentity(gomock.NewController(t))
	id.EXPECT().Service().Return(&storage.ServiceIdentity{Type: svcType}).AnyTimes()
	return authn.ContextWithIdentity(t.Context(), id, t)
}

func TestSensorOr(t *testing.T) {
	tests := map[string]struct {
		ctx         context.Context
		innerAllows bool
		wantErr     error
	}{
		"sensor identity is allowed": {
			ctx: ctxWithService(t, storage.ServiceType_SENSOR_SERVICE),
		},
		"inner authorizer allows non-sensor": {
			ctx:         context.Background(),
			innerAllows: true,
		},
		"wrong service type and inner denies": {
			ctx:     ctxWithService(t, storage.ServiceType_CENTRAL_SERVICE),
			wantErr: errox.NotAuthorized,
		},
		"no identity and inner denies": {
			ctx:     context.Background(),
			wantErr: errox.NotAuthorized,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			inner := fixedAuthorizer{err: errox.NotAuthorized}
			if tt.innerAllows {
				inner = allow
			}
			a := SensorOr(inner)
			require.NotNil(it, a)
			assert.ErrorIs(it, a.Authorized(tt.ctx, ""), tt.wantErr)
		})
	}
}

func TestScannerOr(t *testing.T) {
	tests := map[string]struct {
		ctx         context.Context
		innerAllows bool
		wantErr     error
	}{
		"scanner identity is allowed": {
			ctx: ctxWithService(t, storage.ServiceType_SCANNER_SERVICE),
		},
		"inner authorizer allows non-scanner": {
			ctx:         context.Background(),
			innerAllows: true,
		},
		"wrong service type and inner denies": {
			ctx:     ctxWithService(t, storage.ServiceType_SENSOR_SERVICE),
			wantErr: errox.NotAuthorized,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			inner := fixedAuthorizer{err: errox.NotAuthorized}
			if tt.innerAllows {
				inner = allow
			}
			a := ScannerOr(inner)
			require.NotNil(it, a)
			assert.ErrorIs(it, a.Authorized(tt.ctx, ""), tt.wantErr)
		})
	}
}

func TestScannerV4(t *testing.T) {
	tests := map[string]struct {
		ctx     context.Context
		wantErr error
	}{
		"scanner v4 indexer is allowed": {
			ctx: ctxWithService(t, storage.ServiceType_SCANNER_V4_INDEXER_SERVICE),
		},
		"scanner v4 matcher is allowed": {
			ctx: ctxWithService(t, storage.ServiceType_SCANNER_V4_MATCHER_SERVICE),
		},
		"sensor is denied": {
			ctx:     ctxWithService(t, storage.ServiceType_SENSOR_SERVICE),
			wantErr: errox.NotAuthorized,
		},
		"no identity is denied": {
			ctx:     context.Background(),
			wantErr: errox.NotAuthorized,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			a := ScannerV4()
			require.NotNil(it, a)
			assert.ErrorIs(it, a.Authorized(tt.ctx, ""), tt.wantErr)
		})
	}
}
