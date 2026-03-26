package tlsprofile

import (
	"context"
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func configv1Scheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = configv1.Install(s)
	return s
}

func fakeClientWithAPIServer(apiServer *configv1.APIServer) ctrlClient.Client {
	return fake.NewClientBuilder().WithScheme(configv1Scheme()).WithObjects(apiServer).Build()
}

func TestFetchProfile_NotFound(t *testing.T) {
	funcs := interceptor.Funcs{
		Get: func(_ context.Context, _ ctrlClient.WithWatch, key ctrlClient.ObjectKey, obj ctrlClient.Object, _ ...ctrlClient.GetOption) error {
			if _, ok := obj.(*configv1.APIServer); ok {
				return k8serrors.NewNotFound(schema.GroupResource{
					Group:    "config.openshift.io",
					Resource: "apiservers",
				}, key.Name)
			}
			return nil
		},
	}
	client := fake.NewClientBuilder().WithScheme(configv1Scheme()).WithInterceptorFuncs(funcs).Build()
	clusterTLS, err := FetchProfile(context.Background(), client)
	require.NoError(t, err)
	assert.Nil(t, clusterTLS)
}

func TestFetchProfile_GetError(t *testing.T) {
	funcs := interceptor.Funcs{
		Get: func(_ context.Context, _ ctrlClient.WithWatch, _ ctrlClient.ObjectKey, _ ctrlClient.Object, _ ...ctrlClient.GetOption) error {
			return k8serrors.NewServiceUnavailable("api server unavailable")
		},
	}
	client := fake.NewClientBuilder().WithScheme(configv1Scheme()).WithInterceptorFuncs(funcs).Build()
	clusterTLS, err := FetchProfile(context.Background(), client)
	assert.Error(t, err)
	assert.Nil(t, clusterTLS)
}

func TestFetchProfile_LegacyAdherence(t *testing.T) {
	for _, adherence := range []configv1.TLSAdherencePolicy{
		configv1.TLSAdherencePolicyNoOpinion,
		configv1.TLSAdherencePolicyLegacyAdheringComponentsOnly,
	} {
		t.Run(string(adherence), func(t *testing.T) {
			apiServer := &configv1.APIServer{
				ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
				Spec: configv1.APIServerSpec{
					TLSAdherence: adherence,
					TLSSecurityProfile: &configv1.TLSSecurityProfile{
						Type: configv1.TLSProfileIntermediateType,
					},
				},
			}
			clusterTLS, err := FetchProfile(context.Background(), fakeClientWithAPIServer(apiServer))
			require.NoError(t, err)
			require.NotNil(t, clusterTLS)
			assert.Nil(t, ConvertProfile(clusterTLS, false), "profile should not be enforced")
		})
	}
}

func TestFetchProfile_StrictWithIntermediateProfile(t *testing.T) {
	apiServer := &configv1.APIServer{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Spec: configv1.APIServerSpec{
			TLSAdherence: configv1.TLSAdherencePolicyStrictAllComponents,
			TLSSecurityProfile: &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileIntermediateType,
			},
		},
	}

	clusterTLS, err := FetchProfile(context.Background(), fakeClientWithAPIServer(apiServer))
	require.NoError(t, err)
	require.NotNil(t, clusterTLS)
	profile := ConvertProfile(clusterTLS, false)
	require.NotNil(t, profile)
	assert.Equal(t, "TLSv1.2", profile.MinVersion)
	assert.NotEmpty(t, profile.CipherSuites)
	assert.NotEmpty(t, profile.OpenSSLCiphers)
	assert.NotContains(t, profile.CipherSuites, "TLS_AES_128_GCM_SHA256")
	assert.NotContains(t, profile.OpenSSLCiphers, "TLS_AES_128_GCM_SHA256")
}

func TestFetchProfile_StrictWithModernProfile(t *testing.T) {
	apiServer := &configv1.APIServer{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Spec: configv1.APIServerSpec{
			TLSAdherence: configv1.TLSAdherencePolicyStrictAllComponents,
			TLSSecurityProfile: &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileModernType,
			},
		},
	}

	clusterTLS, err := FetchProfile(context.Background(), fakeClientWithAPIServer(apiServer))
	require.NoError(t, err)
	profile := ConvertProfile(clusterTLS, false)
	require.NotNil(t, profile)
	assert.Equal(t, "TLSv1.3", profile.MinVersion)
	assert.Empty(t, profile.CipherSuites)
	assert.Empty(t, profile.OpenSSLCiphers)
}

func TestFetchProfile_StrictWithCustomProfile(t *testing.T) {
	apiServer := &configv1.APIServer{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Spec: configv1.APIServerSpec{
			TLSAdherence: configv1.TLSAdherencePolicyStrictAllComponents,
			TLSSecurityProfile: &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileCustomType,
				Custom: &configv1.CustomTLSProfile{
					TLSProfileSpec: configv1.TLSProfileSpec{
						MinTLSVersion: configv1.VersionTLS12,
						Ciphers: []string{
							"ECDHE-ECDSA-AES256-GCM-SHA384",
							"ECDHE-RSA-AES256-GCM-SHA384",
						},
					},
				},
			},
		},
	}

	clusterTLS, err := FetchProfile(context.Background(), fakeClientWithAPIServer(apiServer))
	require.NoError(t, err)
	profile := ConvertProfile(clusterTLS, false)
	require.NotNil(t, profile)
	assert.Equal(t, "TLSv1.2", profile.MinVersion)
	assert.Equal(t, "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384", profile.CipherSuites)
	assert.Equal(t, "ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384", profile.OpenSSLCiphers)
}

func TestFetchProfile_StrictWithNilTLSSecurityProfile(t *testing.T) {
	apiServer := &configv1.APIServer{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Spec: configv1.APIServerSpec{
			TLSAdherence: configv1.TLSAdherencePolicyStrictAllComponents,
		},
	}

	clusterTLS, err := FetchProfile(context.Background(), fakeClientWithAPIServer(apiServer))
	require.NoError(t, err)
	profile := ConvertProfile(clusterTLS, false)
	require.NotNil(t, profile)
	assert.Equal(t, "TLSv1.2", profile.MinVersion)
}

func TestConvertProfile_ForceOverridesLegacyAdherence(t *testing.T) {
	apiServer := &configv1.APIServer{
		ObjectMeta: metav1.ObjectMeta{Name: "cluster"},
		Spec: configv1.APIServerSpec{
			TLSAdherence: configv1.TLSAdherencePolicyNoOpinion,
			TLSSecurityProfile: &configv1.TLSSecurityProfile{
				Type: configv1.TLSProfileIntermediateType,
			},
		},
	}
	clusterTLS, err := FetchProfile(context.Background(), fakeClientWithAPIServer(apiServer))
	require.NoError(t, err)
	profile := ConvertProfile(clusterTLS, true)
	require.NotNil(t, profile)
}
