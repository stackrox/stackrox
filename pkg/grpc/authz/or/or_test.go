package or

import (
	"context"
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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
			authorizers: []authz.Authorizer{testutils.Allow(t)},
		},
		"single authorizer denies": {
			authorizers: []authz.Authorizer{testutils.Deny(t)},
			wantErr:     errox.NotAuthorized,
		},
		"all authorizers allow": {
			authorizers: []authz.Authorizer{testutils.Allow(t), testutils.Allow(t), testutils.Allow(t)},
		},
		"all authorizers deny": {
			authorizers: []authz.Authorizer{testutils.Deny(t), testutils.Deny(t), testutils.Deny(t)},
			wantErr:     errox.NotAuthorized,
		},
		"first allows, second denies": {
			authorizers: []authz.Authorizer{testutils.Allow(t), testutils.Deny(t)},
		},
		"first denies, second allows": {
			authorizers: []authz.Authorizer{testutils.Deny(t), testutils.Allow(t)},
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
	second := &testutils.RecordingAuthorizer{}
	err := Or(testutils.Allow(t), second).Authorized(t.Context(), "")
	assert.NoError(t, err)
	assert.Equal(t, 0, second.Calls, "second authorizer must not be called after first succeeds")
}

// TestOr_ForwardsContextAndMethod verifies ctx and methodName reach each sub-authorizer.
func TestOr_ForwardsContextAndMethod(t *testing.T) {
	ctx := context.WithValue(t.Context(), struct{ key string }{"k"}, "v")
	const method = "/some.Service/Method"

	rec := &testutils.RecordingAuthorizer{}
	err := Or(rec).Authorized(ctx, method)
	assert.NoError(t, err)
	assert.Equal(t, 1, rec.Calls)
	assert.Equal(t, []string{method}, rec.Methods)
	assert.Equal(t, []context.Context{ctx}, rec.Contexts)
}

func TestOr_ReturnsAuthorizer(t *testing.T) {
	assert.NotNil(t, Or())
}

// ctxWithService builds a context carrying a mock identity with the given service type.
func ctxWithService(t testing.TB, svcType storage.ServiceType) context.Context {
	t.Helper()
	id := mocks.NewMockIdentity(gomock.NewController(t))
	id.EXPECT().Service().Return(&storage.ServiceIdentity{Type: svcType}).AnyTimes()
	return authn.ContextWithIdentity(t.Context(), id, t)
}

// serviceOrTestCase is a test case shared by TestSensorOr and TestScannerOr.
type serviceOrTestCase struct {
	ctx     context.Context
	inner   authz.Authorizer
	wantErr error
}

// getServiceOrTestCases builds the test cases common to the SensorOr/ScannerOr
// authorizers: the target service type is allowed regardless of the inner
// authorizer, the inner authorizer decides for other identities, and anything
// else is denied.
func getServiceOrTestCases(t testing.TB, targetServiceType, otherServiceType storage.ServiceType) map[string]serviceOrTestCase {
	svc := strings.ToLower(strings.TrimSuffix(targetServiceType.String(), "_SERVICE"))
	return map[string]serviceOrTestCase{
		svc + " identity is allowed": {
			ctx:   ctxWithService(t, targetServiceType),
			inner: testutils.Deny(t),
		},
		"inner authorizer allows non-" + svc: {
			ctx:   context.Background(),
			inner: testutils.Allow(t),
		},
		"wrong service type and inner denies": {
			ctx:     ctxWithService(t, otherServiceType),
			inner:   testutils.Deny(t),
			wantErr: errox.NotAuthorized,
		},
		"no identity and inner denies": {
			ctx:     context.Background(),
			inner:   testutils.Deny(t),
			wantErr: errox.NotAuthorized,
		},
	}
}

func TestSensorOr(t *testing.T) {
	tests := getServiceOrTestCases(t, storage.ServiceType_SENSOR_SERVICE, storage.ServiceType_CENTRAL_SERVICE)
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			a := SensorOr(tt.inner)
			require.NotNil(it, a)
			assert.ErrorIs(it, a.Authorized(tt.ctx, ""), tt.wantErr)
		})
	}
}

func TestScannerOr(t *testing.T) {
	tests := getServiceOrTestCases(t, storage.ServiceType_SCANNER_SERVICE, storage.ServiceType_SENSOR_SERVICE)
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			a := ScannerOr(tt.inner)
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
