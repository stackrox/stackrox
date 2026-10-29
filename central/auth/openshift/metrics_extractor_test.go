package openshift

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	providerMocks "github.com/stackrox/rox/pkg/auth/authproviders/mocks"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authn"
	identityMocks "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type extractorFunc func(context.Context, requestinfo.RequestInfo) (authn.Identity, *authn.ExtractorError)

func (f extractorFunc) IdentityForRequest(ctx context.Context, ri requestinfo.RequestInfo) (authn.Identity, *authn.ExtractorError) {
	return f(ctx, ri)
}

func TestMetricsExtractor(t *testing.T) {
	for name, tc := range map[string]struct {
		providerID string
		certCN     string
		grants     map[string]storage.Access
		wantErr    bool
	}{
		"built-in scoped read": {
			providerID: authProviderID,
			certCN:     env.SecureMetricsClientCertCN.Setting(),
			grants: map[string]storage.Access{
				"Deployment": storage.Access_READ_ACCESS, "Administration": storage.Access_NO_ACCESS,
			},
		},
		"built-in token with extra role granting global read": {
			providerID: authProviderID,
			certCN:     env.SecureMetricsClientCertCN.Setting(),
			grants: map[string]storage.Access{
				"Deployment": storage.Access_READ_ACCESS, "Administration": storage.Access_READ_ACCESS,
			},
			wantErr: true,
		},
		"built-in write": {
			providerID: authProviderID,
			certCN:     env.SecureMetricsClientCertCN.Setting(),
			grants:     map[string]storage.Access{"Deployment": storage.Access_READ_WRITE_ACCESS},
			wantErr:    true,
		},
		"built-in unknown resource": {
			providerID: authProviderID,
			certCN:     env.SecureMetricsClientCertCN.Setting(),
			grants:     map[string]storage.Access{"UnknownResource": storage.Access_READ_ACCESS},
			wantErr:    true,
		},
		"custom provider": {
			providerID: "custom-provider",
			certCN:     env.SecureMetricsClientCertCN.Setting(),
			grants:     map[string]storage.Access{"Administration": storage.Access_READ_ACCESS},
		},
		"custom certificate on built-in provider": {
			providerID: authProviderID,
			certCN:     "custom-scraper",
			grants:     map[string]storage.Access{"Administration": storage.Access_READ_ACCESS},
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			provider := providerMocks.NewMockProvider(ctrl)
			provider.EXPECT().ID().Return(tc.providerID)
			id := identityMocks.NewMockIdentity(ctrl)
			id.EXPECT().ExternalAuthProvider().Return(provider)
			if tc.providerID == authProviderID {
				id.EXPECT().Attributes().Return(map[string][]string{
					authproviders.NameAttribute: {tc.certCN},
				})
				if tc.certCN == env.SecureMetricsClientCertCN.Setting() {
					id.EXPECT().Permissions().Return(tc.grants)
				}
			}
			extractor := metricsExtractor{extractorFunc(func(context.Context, requestinfo.RequestInfo) (authn.Identity, *authn.ExtractorError) {
				return id, nil
			})}
			actual, err := extractor.IdentityForRequest(t.Context(), requestinfo.RequestInfo{})
			if tc.wantErr {
				require.NotNil(t, err)
				assert.Nil(t, actual)
			} else {
				require.Nil(t, err)
				assert.Same(t, id, actual)
			}
		})
	}
}

func TestMetricsExtractorPreservesDelegatedError(t *testing.T) {
	delegatedErr := authn.NewExtractorError("userpki", "invalid certificate", nil)
	extractor := metricsExtractor{extractorFunc(func(context.Context, requestinfo.RequestInfo) (authn.Identity, *authn.ExtractorError) {
		return nil, delegatedErr
	})}
	id, err := extractor.IdentityForRequest(t.Context(), requestinfo.RequestInfo{})
	assert.Nil(t, id)
	assert.Same(t, delegatedErr, err)
}
