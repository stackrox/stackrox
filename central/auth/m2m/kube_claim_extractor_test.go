package m2m

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
)

func Test_kubeClaimExtractor(t *testing.T) {
	t.Run("success", func(t *testing.T) {

		e := newClaimExtractorFromConfig(&storage.AuthMachineToMachineConfig{
			Id:   "id1",
			Type: storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT,
		})

		claims, err := e.ExtractClaims(&IDToken{
			Claims: func(v any) error {
				*v.(*map[string]any) = map[string]interface{}{
					"aud": []string{"https://example.com"},
					"exp": 1763119831,
					"iat": 1763116231,
					"iss": "https://example.com",
					"jti": "6a5e8681-3b2a-44f2-9462-ecf16f52c779",
					"kubernetes.io": map[string]interface{}{
						"namespace": "stackrox",
						"serviceaccount": map[string]interface{}{
							"name": "config-controller",
							"uid":  "3cd68f8a-7e72-44e7-af17-b283e7027980",
						},
					},
					"nbf": 1763116231,
					"sub": "system:serviceaccount:stackrox:config-controller",
				}
				return nil
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, map[string][]string{
			"aud":                               {"https://example.com"},
			"iss":                               {"https://example.com"},
			"jti":                               {"6a5e8681-3b2a-44f2-9462-ecf16f52c779"},
			"kubernetes.io.namespace":           {"stackrox"},
			"kubernetes.io.serviceaccount.name": {"config-controller"},
			"kubernetes.io.serviceaccount.uid":  {"3cd68f8a-7e72-44e7-af17-b283e7027980"},
			"sub":                               {"system:serviceaccount:stackrox:config-controller"},
		}, claims)
	})

	t.Run("Kubernetes Token error", func(t *testing.T) {
		e := newClaimExtractorFromConfig(&storage.AuthMachineToMachineConfig{
			Id:   "id1",
			Type: storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT,
		})
		testErr := errox.NotImplemented
		_, err := e.ExtractClaims(&IDToken{
			Claims: func(a any) error {
				return testErr
			},
		})
		assert.ErrorIs(t, err, errox.NotImplemented)
		assert.Contains(t, err.Error(), "extracting claims")
	})
}
