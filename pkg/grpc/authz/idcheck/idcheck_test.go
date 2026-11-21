package idcheck

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// fakeIDAuthorizer is a simple IdentityBasedAuthorizer that returns a preset error.
type fakeIDAuthorizer struct {
	err error
}

func (f fakeIDAuthorizer) AuthorizeByIdentity(_ authn.Identity) error {
	return f.err
}

func TestWrap_Authorized(t *testing.T) {
	errCustom := errors.New("custom error")

	ctrl := gomock.NewController(t)
	id := mocks.NewMockIdentity(ctrl)
	ctxWithID := authn.ContextWithIdentity(context.Background(), id, t)

	tests := map[string]struct {
		ctx        context.Context
		idAuth     IdentityBasedAuthorizer
		methodName string
		wantErr    error
	}{
		"no identity in context returns NoCredentials": {
			ctx:     context.Background(),
			idAuth:  fakeIDAuthorizer{},
			wantErr: errox.NoCredentials,
		},
		"identity present and authorizer allows": {
			ctx:    ctxWithID,
			idAuth: fakeIDAuthorizer{},
		},
		"identity present but authorizer denies": {
			ctx:     ctxWithID,
			idAuth:  fakeIDAuthorizer{err: errCustom},
			wantErr: errCustom,
		},
		"method name is ignored": {
			ctx:        ctxWithID,
			idAuth:     fakeIDAuthorizer{},
			methodName: "/some.Service/Method",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			a := Wrap(tt.idAuth)
			assert.NotNil(it, a)
			assert.ErrorIs(it, a.Authorized(tt.ctx, tt.methodName), tt.wantErr)
		})
	}
}

func TestServiceType_AuthorizeByIdentity(t *testing.T) {
	required := storage.ServiceType_SENSOR_SERVICE
	wrong := storage.ServiceType_CENTRAL_SERVICE

	tests := map[string]struct {
		service *storage.ServiceIdentity
		wantErr error
	}{
		"nil service returns NoCredentials": {
			service: nil,
			wantErr: errox.NoCredentials,
		},
		"matching service type is authorized": {
			service: &storage.ServiceIdentity{Type: required},
		},
		"mismatching service type returns NotAuthorized": {
			service: &storage.ServiceIdentity{Type: wrong},
			wantErr: errox.NotAuthorized,
		},
		"unknown service type returns NotAuthorized": {
			service: &storage.ServiceIdentity{Type: storage.ServiceType_UNKNOWN_SERVICE},
			wantErr: errox.NotAuthorized,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			ctrl := gomock.NewController(t)
			id := mocks.NewMockIdentity(ctrl)
			id.EXPECT().Service().Return(tt.service)

			err := serviceType(required).AuthorizeByIdentity(id)
			assert.ErrorIs(it, err, tt.wantErr)
		})
	}
}

func TestServiceTypeAuthorizers(t *testing.T) {
	wrongType := storage.ServiceType_UNKNOWN_SERVICE

	tests := map[string]struct {
		constructor func() authz.Authorizer
		allowedType storage.ServiceType
	}{
		"CentralOnly allows Central": {
			constructor: CentralOnly,
			allowedType: storage.ServiceType_CENTRAL_SERVICE,
		},
		"SensorsOnly allows Sensor": {
			constructor: SensorsOnly,
			allowedType: storage.ServiceType_SENSOR_SERVICE,
		},
		"SensorRegistrantsOnly allows Registrant": {
			constructor: SensorRegistrantsOnly,
			allowedType: storage.ServiceType_REGISTRANT_SERVICE,
		},
		"ScannerOnly allows Scanner": {
			constructor: ScannerOnly,
			allowedType: storage.ServiceType_SCANNER_SERVICE,
		},
		"ScannerV4IndexerOnly allows ScannerV4Indexer": {
			constructor: ScannerV4IndexerOnly,
			allowedType: storage.ServiceType_SCANNER_V4_INDEXER_SERVICE,
		},
		"ScannerV4MatcherOnly allows ScannerV4Matcher": {
			constructor: ScannerV4MatcherOnly,
			allowedType: storage.ServiceType_SCANNER_V4_MATCHER_SERVICE,
		},
		"CollectorOnly allows Collector": {
			constructor: CollectorOnly,
			allowedType: storage.ServiceType_COLLECTOR_SERVICE,
		},
		"AdmissionControlOnly allows AdmissionControl": {
			constructor: AdmissionControlOnly,
			allowedType: storage.ServiceType_ADMISSION_CONTROL_SERVICE,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			a := tt.constructor()
			assert.NotNil(it, a)

			it.Run("allows correct service type", func(iit *testing.T) {
				ctrl := gomock.NewController(iit)
				id := mocks.NewMockIdentity(ctrl)
				id.EXPECT().Service().Return(&storage.ServiceIdentity{Type: tt.allowedType})
				ctx := authn.ContextWithIdentity(iit.Context(), id, iit)
				assert.NoError(iit, a.Authorized(ctx, ""))
			})

			it.Run("rejects wrong service type", func(iit *testing.T) {
				ctrl := gomock.NewController(iit)
				id := mocks.NewMockIdentity(ctrl)
				id.EXPECT().Service().Return(&storage.ServiceIdentity{Type: wrongType})
				ctx := authn.ContextWithIdentity(iit.Context(), id, iit)
				assert.ErrorIs(iit, a.Authorized(ctx, ""), errox.NotAuthorized)
			})

			it.Run("rejects no identity in context", func(iit *testing.T) {
				assert.ErrorIs(iit, a.Authorized(iit.Context(), ""), errox.NoCredentials)
			})

			it.Run("rejects nil service", func(iit *testing.T) {
				ctrl := gomock.NewController(iit)
				id := mocks.NewMockIdentity(ctrl)
				id.EXPECT().Service().Return(nil)
				ctx := authn.ContextWithIdentity(iit.Context(), id, iit)
				assert.ErrorIs(iit, a.Authorized(ctx, ""), errox.NoCredentials)
			})
		})
	}
}
