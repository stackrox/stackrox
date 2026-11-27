package m2m

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
)

func Test_newClaimExtractorFromConfigType(t *testing.T) {
	testNotImplementedErr := errox.NotImplemented

	t.Run("GitHub Actions", func(it *testing.T) {
		e := newClaimExtractorForConfigType(storage.AuthMachineToMachineConfig_GITHUB_ACTIONS)
		_, err := e.ExtractClaims(&IDToken{
			Claims: func(a any) error {
				return testNotImplementedErr
			},
		})
		assert.ErrorIs(it, err, errox.NotImplemented)
		assert.Contains(it, err.Error(), "extracting GitHub Actions claims")

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
		assert.NoError(it, err)
		roxClaims, err := e.ExtractRoxClaims(claims)
		assert.NoError(it, err)
		assert.Equal(it, []string{"test"}, roxClaims.ExternalUser.Attributes["actor"])
	})

	t.Run("Generic", func(it *testing.T) {
		e := newClaimExtractorForConfigType(storage.AuthMachineToMachineConfig_GENERIC)
		_, err := e.ExtractClaims(&IDToken{
			Claims: func(a any) error {
				return testNotImplementedErr
			},
		})
		assert.ErrorIs(it, err, errox.NotImplemented)
		assert.Contains(it, err.Error(), "extracting claims")

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
		assert.NoError(it, err)
		roxClaims, err := e.ExtractRoxClaims(claims)
		assert.NoError(it, err)
		assert.Equal(it, "some@ema.il", roxClaims.ExternalUser.FullName)
	})
}
