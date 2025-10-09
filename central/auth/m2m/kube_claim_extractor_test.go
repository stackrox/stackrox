package m2m

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/authentication/v1"
)

func Test_kubeClaimExtractor(t *testing.T) {
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

	t.Run("Kubernetes Token", func(t *testing.T) {
		e := newClaimExtractorFromConfig(&storage.AuthMachineToMachineConfig{
			Id:   "id1",
			Type: storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT,
		})
		trs := &v1.TokenReviewStatus{
			User: v1.UserInfo{
				Username: "username",
				UID:      "uid",
				Groups:   []string{"gr1", "gr2"},
				Extra: map[string]v1.ExtraValue{
					"extra": {"ev1"},
				},
			},
			Audiences: []string{"aud1", "aud2"},
		}
		claims, err := e.ExtractClaims(tokenFromReview(trs))
		assert.NoError(t, err)

		roxClaims, err := e.ExtractRoxClaims(claims)
		assert.NoError(t, err)

		assert.Equal(t, tokens.RoxClaims{
			Name: "username",
			ExternalUser: &tokens.ExternalUserClaim{
				UserID:   "uid",
				FullName: "username",
				Attributes: map[string][]string{
					"sub":    {"username"},
					"aud":    {"aud1", "aud2"},
					"uid":    {"uid"},
					"groups": {"gr1", "gr2"},
					"extra":  {"ev1"},
				},
			}},
			roxClaims)
	})
}
