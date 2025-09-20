package m2m

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
)

func Test_newClaimsExtractorFromConfig(t *testing.T) {
	t.Run("GitHub Actions", func(t *testing.T) {
		e := newClaimExtractorFromConfig(&storage.AuthMachineToMachineConfig{
			Id:     "id1",
			Issuer: "test",
			Type:   storage.AuthMachineToMachineConfig_GITHUB_ACTIONS,
		})
		testErr := errox.NotImplemented
		_, err := e.ExtractClaims(&IDToken{
			Claims: func(a any) error {
				return testErr
			},
		})
		assert.ErrorIs(t, err, errox.NotImplemented)
		assert.Contains(t, err.Error(), "extracting GitHub Actions claims")

		claims, err := e.ExtractClaims(&IDToken{
			Subject:  "subject",
			Audience: []string{"audience"},
			Claims: func(ghac any) error {
				claims := ghac.(*githubActionClaims)
				claims.Actor = "test"
				claims.ActorID = "testID"
				return nil
			},
		})
		assert.NoError(t, err)
		roxclaims, err := e.ExtractRoxClaims(claims)
		assert.NoError(t, err)
		assert.Equal(t, []string{"test"}, roxclaims.ExternalUser.Attributes["actor"])
	})

	t.Run("Generic", func(t *testing.T) {
		e := newClaimExtractorFromConfig(&storage.AuthMachineToMachineConfig{
			Id:   "id1",
			Type: storage.AuthMachineToMachineConfig_GENERIC,
		})
		testErr := errox.NotImplemented
		_, err := e.ExtractClaims(&IDToken{
			Claims: func(a any) error {
				return testErr
			},
		})
		assert.ErrorIs(t, err, errox.NotImplemented)
		assert.Contains(t, err.Error(), "extracting claims")

		claims, err := e.ExtractClaims(&IDToken{
			Subject:  "subject",
			Audience: []string{"audience"},
			Claims: func(unstructured any) error {
				claims := unstructured.(*map[string]any)
				*claims = map[string]any{
					"email": "some@ema.il",
				}
				return nil
			},
		})
		assert.NoError(t, err)
		roxclaims, err := e.ExtractRoxClaims(claims)
		assert.NoError(t, err)
		assert.Equal(t, "some@ema.il", roxclaims.ExternalUser.FullName)
	})
}
